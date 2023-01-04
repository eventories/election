package election

import (
	"encoding/json"
	"net"
)

const (
	// Common
	pingType = byte(0) + iota
	pongType
	newTermType
	voteType
	voteMeType
	notifyLeaderType
)

type Msg interface {
	Kind() byte

	// 'sender' is a field for storing message recipients. This is a
	// field for responding, and is not included when sending to the
	// other node.
	SetSender(*net.UDPAddr)
	Sender() *net.UDPAddr
}

type (
	pingMsg struct {
		Term   uint64
		sender *net.UDPAddr
	}

	// pongMsg is the response to pingMsg. Transmits the Term and Leader
	// of the node. With this message, Follwer verifies that the Leader
	// is working normally.
	pongMsg struct {
		Term   uint64
		Leader string
		sender *net.UDPAddr
	}

	// newTermMsg is a broadcast message when the Leader is not working
	// normally. The node receiving this message checks whether the
	// Leader is actually not working, and if it is true, waits for a
	// random time (300~500ms) and broadcast voteMeMsg to other nodes.
	newTermMsg struct {
		Term   uint64
		sender *net.UDPAddr
	}

	// voteMsg is the response message to voteMeMsg. It has no special
	// conditions and responds to the message it receives first.
	voteMsg struct {
		Term   uint64
		sender *net.UDPAddr
	}

	// voteMeMsg is sent after waiting for a random amount of time
	// after the Leader election process starts.
	voteMeMsg struct {
		Term   uint64
		sender *net.UDPAddr
	}

	// notifyLeaderMsg is a message that the elected node broadcasts to
	// otehr nodes when the node election process is complete.
	notifyLeaderMsg struct {
		Term   uint64
		Leader string
		sender *net.UDPAddr
	}
)

func decodePacket(b []byte) (Msg, error) {
	var (
		kind    = b[0]
		payload = b[1:]
	)

	var msg Msg
	switch kind {
	case pingType:
		var p pingMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p

	case pongType:
		var p pongMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p

	case voteType:
		var p voteMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p

	case voteMeType:
		var p voteMeMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p

	case newTermType:
		var p newTermMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p

	case notifyLeaderType:
		var p notifyLeaderMsg
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, err
		}
		msg = &p
	}

	return msg, nil
}

// sendMsg decodes the Msg and transmits it to the destination. During
// decoding, the type of Msg is added to the first byte.
func sendMsg(w *net.UDPConn, to *net.UDPAddr, msg Msg) (int, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	rb := make([]byte, len(b)+1)
	rb[0] = msg.Kind()
	copy(rb[1:], b)

	return w.WriteToUDP(rb, to)
}

func (p *pingMsg) Sender() *net.UDPAddr        { return p.sender }
func (p *pingMsg) SetSender(addr *net.UDPAddr) { p.sender = addr }
func (*pingMsg) Kind() byte                    { return pingType }

func (p *pongMsg) Sender() *net.UDPAddr        { return p.sender }
func (p *pongMsg) SetSender(addr *net.UDPAddr) { p.sender = addr }
func (*pongMsg) Kind() byte                    { return pongType }

func (n *newTermMsg) Sender() *net.UDPAddr        { return n.sender }
func (n *newTermMsg) SetSender(addr *net.UDPAddr) { n.sender = addr }
func (*newTermMsg) Kind() byte                    { return newTermType }

func (v *voteMeMsg) Sender() *net.UDPAddr        { return v.sender }
func (v *voteMeMsg) SetSender(addr *net.UDPAddr) { v.sender = addr }
func (*voteMeMsg) Kind() byte                    { return voteMeType }

func (v *voteMsg) Sender() *net.UDPAddr        { return v.sender }
func (v *voteMsg) SetSender(addr *net.UDPAddr) { v.sender = addr }
func (*voteMsg) Kind() byte                    { return voteType }

func (n *notifyLeaderMsg) Sender() *net.UDPAddr        { return n.sender }
func (n *notifyLeaderMsg) SetSender(addr *net.UDPAddr) { n.sender = addr }
func (*notifyLeaderMsg) Kind() byte                    { return notifyLeaderType }
