package election

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

// - Stat event (notify changed Term or Role)
// - Cluster event (added followers, addMember, delMember)
func (e *Election) Subscribe() *Subscription {
	panic("")
}
