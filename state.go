package election

import (
	"sync"
	"sync/atomic"
)

type Role uint32

const (
	Leader = iota
	Follower
	Candidate
	Shutdown
)

func (r Role) String() string {
	switch r {
	case Leader:
		return "Leader"
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Shutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

type state struct {
	currentTerm uint64
	currentRole Role
	vote        map[uint64]struct{}

	mu sync.RWMutex
}

func newState() *state {
	return &state{
		currentTerm: 0,
		currentRole: Shutdown,
		vote:        make(map[uint64]struct{}),
	}
}

func (s *state) voted(term uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.vote[term]
	return ok
}

func (s *state) voting(term uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vote[term] = struct{}{}
}

func (s *state) term() uint64 {
	return atomic.LoadUint64(&s.currentTerm)
}

func (s *state) setTerm(term uint64) {
	atomic.StoreUint64(&s.currentTerm, term)
}

func (s *state) role() Role {
	roleAddr := (*uint32)(&s.currentRole)
	return Role(atomic.LoadUint32(roleAddr))
}

func (s *state) setRole(r Role) {
	stateAddr := (*uint32)(&s.currentRole)
	atomic.StoreUint32(stateAddr, uint32(r))
}
