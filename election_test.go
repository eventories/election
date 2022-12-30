package election

import (
	"fmt"
	"testing"
	"time"
)

func TestSeungbae(t *testing.T) {
	e1, _ := NewElection(&Config{"127.0.0.1:8551", []string{"127.0.0.1:8552"}, nil})
	e2, _ := NewElection(&Config{"127.0.0.1:8552", []string{"127.0.0.1:8551"}, nil})
	e3, _ := NewElection(&Config{"127.0.0.1:8553", []string{"127.0.0.1:8551", "127.0.0.1:8552"}, nil})
	e4, _ := NewElection(&Config{"127.0.0.1:8554", []string{"127.0.0.1:8551", "127.0.0.1:8552", "127.0.0.1:8552"}, nil})
	e5, _ := NewElection(&Config{"127.0.0.1:8555", []string{"127.0.0.1:8551", "127.0.0.1:8552", "127.0.0.1:8553", "127.0.0.1:8554"}, nil})
	e1.Run()
	e2.Run()

	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("** register node 3 **")
		e1.memberlist["127.0.0.1:8553"] = struct{}{}
		e2.memberlist["127.0.0.1:8553"] = struct{}{}
		e3.Run()
	}()

	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("** register node 4 **")
		e1.memberlist["127.0.0.1:8554"] = struct{}{}
		e2.memberlist["127.0.0.1:8554"] = struct{}{}
		e3.memberlist["127.0.0.1:8554"] = struct{}{}
		e4.Run()
	}()

	go func() {
		time.Sleep(15 * time.Second)
		fmt.Println("** shutdown Leader **")
		if e1.state.role() == Leader {
			e1.Stop()
		}
		if e2.state.role() == Leader {
			e2.Stop()
		}
	}()

	go func() {
		time.Sleep(25 * time.Second)
		fmt.Println("** register node 5 **")
		e1.memberlist["127.0.0.1:8555"] = struct{}{}
		e2.memberlist["127.0.0.1:8555"] = struct{}{}
		e3.memberlist["127.0.0.1:8555"] = struct{}{}
		e4.memberlist["127.0.0.1:8555"] = struct{}{}
		e5.Run()
	}()

	done := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Second)
		close(done)
	}()

	for {
		select {
		case <-done:
			e1.Stop()
			e2.Stop()
			e3.Stop()
			e4.Stop()
			e5.Stop()
			fmt.Println("done")
			return
		default:
			time.Sleep(time.Second)
			fmt.Println("e1", e1.state.role().String(), e1.state.term())
			fmt.Println("e2", e2.state.role().String(), e2.state.term())
			fmt.Println("e3", e3.state.role().String(), e3.state.term())
			fmt.Println("e4", e4.state.role().String(), e4.state.term())
			fmt.Println("e5", e5.state.role().String(), e5.state.term())
			fmt.Println()
		}

	}
}
