package election

import (
	"errors"
	"net"
)

type Subscription struct {
	C      <-chan Stat
	Cancel func()
}

func (e *Election) Term() uint64 {
	return e.state.term()
}

func (e *Election) Role() Role {
	return e.state.role()
}

func (e *Election) Leader() *net.UDPAddr {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.leaderAddr
}

func (e *Election) Cluster() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	members := make([]string, 0, len(e.memberlist))
	for m := range e.memberlist {
		members = append(members, m)
	}

	return members
}

// Consistency for the cluster is the role of the upper layer.
// We can add logic to stop working when there is no consistency
// for the cluster.
func (e *Election) AddMember(member string) error {
	if _, err := net.ResolveTCPAddr("tcp", member); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.memberlist[member]; ok {
		return errors.New("already exist")
	}
	e.memberlist[member] = struct{}{}
	return nil
}

func (e *Election) DelMember(member string) error {
	if _, err := net.ResolveTCPAddr("tcp", member); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.memberlist[member]; !ok {
		return errors.New("not exist")
	}
	delete(e.memberlist, member)
	return nil
}

// - Stat event (notify changed Term or Role)
// - Cluster event (added followers, addMember, delMember)
func (e *Election) Subscribe() *Subscription {
	panic("")
}
