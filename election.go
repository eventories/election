package election

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

type Election struct {
	conn      *net.UDPConn
	localaddr *net.UDPAddr
	logger    *log.Logger

	memberlist map[string]struct{} // Cluster node list.
	state      *state              // Node information. Include Role, Term, Vote state.

	leaderAddr *net.UDPAddr

	// Common task channels
	pingCh         chan *pingMsg
	pongCh         chan *pongMsg
	voteCh         chan *voteMsg
	voteMeCh       chan *voteMeMsg
	newTermCh      chan uint64
	notifyLeaderCh chan *notifyLeaderMsg
	stopCh         chan struct{}

	// API channels
	addMember chan *addMemberMsg
	delMember chan *delMemberMsg
}

func NewElection(cfg *Config) (*Election, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "[Election]", log.LstdFlags)
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:55031"
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	addr, _ := net.ResolveUDPAddr("udp", cfg.ListenAddr)

	memberlist := make(map[string]struct{})
	for _, node := range cfg.Cluster {
		memberlist[node] = struct{}{}
	}

	election := &Election{
		localaddr:      addr,
		logger:         cfg.Logger,
		memberlist:     memberlist,
		state:          newState(),
		leaderAddr:     nil,
		pingCh:         make(chan *pingMsg, 1),
		pongCh:         make(chan *pongMsg, 1),
		voteCh:         make(chan *voteMsg, 1),
		voteMeCh:       make(chan *voteMeMsg, 1),
		notifyLeaderCh: make(chan *notifyLeaderMsg, 1),
		newTermCh:      make(chan uint64, 1),
		addMember:      make(chan *addMemberMsg, 1),
		delMember:      make(chan *delMemberMsg, 1),
		stopCh:         make(chan struct{}, 1),
	}

	cfg.Logger.Printf(
		"create succeed, localaddr: %s, cluster: %v, role: %s, term: %d\n",
		election.localaddr,
		election.memberlist,
		election.state.role().String(),
		election.state.term(),
	)

	return election, nil
}

func (e *Election) Run() (err error) {
	if e.state.role() != Shutdown {
		return errors.New("already run")
	}
	e.conn, err = net.ListenUDP("udp", e.localaddr)
	if err != nil {
		return err
	}

	go e.readLoop()
	go e.runCandidate() // Start Candidate first.

	e.logger.Printf("%s loop started\n", e.localaddr)
	return nil
}

func (e *Election) Stop() {
	if e.state.role() == Shutdown {
		return
	}
	close(e.stopCh)
	e.conn.Close()
	e.state.setRole(Shutdown)
	e.state.setTerm(0)
}

func (e *Election) readLoop() {
	for {
		b := make([]byte, 128)
		n, sender, err := e.conn.ReadFromUDP(b)
		if err != nil {
			if e.state.role() == Shutdown {
				return
			}
			e.logger.Printf("ReadFromUDP failure: %v\n", err)
			continue
		}

		msg, err := decodePacket(b[:n])
		if err != nil {
			e.logger.Printf("decodePacket failure: %v\n", err)
			continue
		}

		msg.SetSender(sender)

		e.handle(msg)
	}
}

func (e *Election) handle(msg Msg) {
	switch msg.Kind() {
	case pingType:
		e.pingCh <- msg.(*pingMsg)

	case pongType:
		e.pongCh <- msg.(*pongMsg)

	case voteType:
		e.voteCh <- msg.(*voteMsg)

	case voteMeType:
		e.voteMeCh <- msg.(*voteMeMsg)

	case newTermType:
		e.newTermCh <- msg.(*newTermMsg).Term

	case notifyLeaderType:
		e.notifyLeaderCh <- msg.(*notifyLeaderMsg)

	case addMemberType:
		e.addMember <- msg.(*addMemberMsg)

	case delMemberType:
		e.delMember <- msg.(*delMemberMsg)
	}
}

// Multiple loops for each role should not run simultaneously. One
// loop is executed per role.

func (e *Election) runLeader() {
	e.state.setRole(Leader)

	for {
		select {
		case ping := <-e.pingCh:
			sendMsg(e.conn, ping.sender, &pongMsg{e.state.term(), e.leaderAddr.String(), nil})

		case <-e.pongCh:
			continue

		case <-e.voteCh:
			continue

		case voteMe := <-e.voteMeCh:
			sendMsg(e.conn, voteMe.sender, &pongMsg{e.state.term(), e.leaderAddr.String(), nil})

		case <-e.newTermCh:
			continue

		case noti := <-e.notifyLeaderCh:
			addr, _ := net.ResolveUDPAddr("udp", noti.Leader)
			e.leaderAddr = addr
			e.state.setTerm(noti.Term)

			sendMsg(e.conn, e.leaderAddr, &addMemberMsg{e.state.term(), e.conn.LocalAddr().String(), nil})

			go e.runFollower()

			return

		// TODO (dbadoy): The cluster node list must be shared by all nodes.
		case member := <-e.addMember:
			e.memberlist[member.Member] = struct{}{}

		case member := <-e.delMember:
			delete(e.memberlist, member.Member)

		case <-e.stopCh:
			return
		}
	}
}

func (e *Election) runFollower() {
	e.state.setRole(Follower)

	interval := time.NewTicker(3 * time.Second)
	defer interval.Stop()
	for {
		select {
		case ping := <-e.pingCh:
			sendMsg(e.conn, ping.sender, &pongMsg{e.state.term(), e.leaderAddr.String(), nil})

		case <-e.pongCh:
			continue

		case <-e.voteCh:
			continue

		case <-e.voteMeCh:
			continue

		case term := <-e.newTermCh:
			if e.state.term() >= term {
				continue
			}

			if err := e.ping(e.leaderAddr, 100*time.Millisecond); err == nil {
				continue
			}

			go e.runCandidate()

			return

		case noti := <-e.notifyLeaderCh:
			addr, _ := net.ResolveUDPAddr("udp", noti.Leader)
			e.leaderAddr = addr
			e.state.setTerm(noti.Term)

			sendMsg(e.conn, e.leaderAddr, &addMemberMsg{e.state.term(), e.conn.LocalAddr().String(), nil})

		case member := <-e.addMember:
			if e.state.term() != member.Term {
				continue
			}
			sendMsg(e.conn, e.leaderAddr, &addMemberMsg{e.state.term(), member.Member, nil})

		case member := <-e.delMember:
			if e.state.term() != member.Term {
				continue
			}
			sendMsg(e.conn, e.leaderAddr, &delMemberMsg{e.state.term(), member.Member, nil})

		case <-interval.C:
			if err := e.ping(e.leaderAddr, 300*time.Millisecond); err == nil {
				continue
			}

			e.broadcast(&newTermMsg{e.state.term(), nil})

			go e.runCandidate()

			return

		case <-e.stopCh:
			return
		}
	}
}

func (e *Election) runCandidate() {
	e.state.setRole(Candidate)
	e.state.setTerm(e.state.term() + 1)

	// Random time sleep
	rand.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(rand.Intn(500)+300) * time.Millisecond)

	var (
		total   = 0
		want    = 0
		timeout = time.Second
	)

	electionTimeout := time.NewTimer(timeout)
	defer electionTimeout.Stop()

	for {
		select {
		case <-electionTimeout.C:
			if e.state.voted(e.state.term()) {
				continue
			}

			e.broadcast(&voteMeMsg{e.state.term(), nil})

			// reset
			total = len(e.memberlist)

			// Broadcast votingMeMsg means voting for self.
			want = 1

			// Accept Solo mode.
			if total == 0 {
				go e.runLeader()
				return
			}

			electionTimeout.Reset(timeout)

		case <-e.pingCh:
			continue

		case pong := <-e.pongCh:
			e.state.setTerm(pong.Term)
			e.leaderAddr = pong.sender

			go e.runFollower()

			return

		case vote := <-e.voteCh:
			if e.state.term() != vote.Term {
				continue
			}

			want++

			if want < total/2+1 {
				continue
			}

			e.broadcast(&notifyLeaderMsg{e.state.term(), e.conn.LocalAddr().String(), nil})

			go e.runLeader()

			return

		case voteMe := <-e.voteMeCh:
			if e.state.term() > voteMe.Term {
				continue
			}
			if e.state.term() < voteMe.Term {
				e.state.setTerm(voteMe.Term)
			}

			// Already voted.
			if e.state.voted(e.state.term()) {
				continue
			}

			sendMsg(e.conn, voteMe.sender, &voteMsg{e.state.term(), nil})
			e.state.voting(e.state.term())

		case <-e.newTermCh:
			continue

		case noti := <-e.notifyLeaderCh:
			addr, _ := net.ResolveUDPAddr("udp", noti.Leader)
			e.leaderAddr = addr
			e.state.setTerm(noti.Term)

			sendMsg(e.conn, e.leaderAddr, &addMemberMsg{e.state.term(), e.conn.LocalAddr().String(), nil})

			go e.runFollower()

			return

		case <-e.addMember:
			continue

		case <-e.delMember:
			continue

		case <-e.stopCh:
			return
		}
	}
}

func (e *Election) broadcast(msg Msg) {
	for member := range e.memberlist {
		addr, _ := net.ResolveUDPAddr("udp", member)
		go sendMsg(e.conn, addr, msg)
	}
}

func (e *Election) ping(to *net.UDPAddr, timeout time.Duration) error {
	if _, err := sendMsg(e.conn, to, &pingMsg{}); err != nil {
		return err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case pong := <-e.pongCh:
		if e.state.term() > pong.Term {
			return errors.New("invalid localnode term")
		}
	case <-timer.C:
		return errors.New("timeout")
	}

	return nil
}
