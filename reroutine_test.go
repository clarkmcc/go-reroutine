// Copyright (C) 2022 Print Tracker, LLC - All Rights Reserved
//
// Unauthorized copying of this file, via any medium is strictly prohibited
// as this source code is proprietary and confidential. Dissemination of this
// information or reproduction of this material is strictly forbidden unless
// prior written permission is obtained from Print Tracker, LLC.

package reroutine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGo(t *testing.T) {
	t.Run("Stop channel", func(t *testing.T) {
		stop := make(chan struct{})
		i := int32(0)
		BlockingGo(stop, func() {
			for {
				if atomic.AddInt32(&i, 1) == 3 {
					close(stop)
				}
				panic("panicked")
			}
		})
		if atomic.LoadInt32(&i) != 3 {
			t.Error("expected three iterations")
		}
	})

	t.Run("Tomb", func(t *testing.T) {
		ts := mockTomb{}
		ts.Go(func() error {
			<-ts.Dying()
			return nil
		})

		i := int32(0)
		BlockingGoTomb(&ts, func() error {
			for {
				if atomic.AddInt32(&i, 1) == 3 {
					ts.Kill(nil)
				}
				panic("panicked")
			}
			return nil
		})
		if atomic.LoadInt32(&i) != 3 {
			t.Error("expected three iterations")
		}
	})
}

// A mockTomb tracks the lifecycle of one or more goroutines as alive,
// dying or dead, and the reason for their death.
//
// See the package documentation for details.
type mockTomb struct {
	m      sync.Mutex
	alive  int
	dying  chan struct{}
	dead   chan struct{}
	reason error

	// context.Context is available in Go 1.7+.
	parent interface{}
	child  map[interface{}]childContext
}

type childContext struct {
	context interface{}
	cancel  func()
	done    <-chan struct{}
}

var (
	ErrStillAlive = errors.New("tomb: still alive")
	ErrDying      = errors.New("tomb: dying")
)

func (t *mockTomb) init() {
	t.m.Lock()
	if t.dead == nil {
		t.dead = make(chan struct{})
		t.dying = make(chan struct{})
		t.reason = ErrStillAlive
	}
	t.m.Unlock()
}

// Dead returns the channel that can be used to wait until
// all goroutines have finished running.
func (t *mockTomb) Dead() <-chan struct{} {
	t.init()
	return t.dead
}

// Dying returns the channel that can be used to wait until
// t.Kill is called.
func (t *mockTomb) Dying() <-chan struct{} {
	t.init()
	return t.dying
}

// Wait blocks until all goroutines have finished running, and
// then returns the reason for their death.
func (t *mockTomb) Wait() error {
	t.init()
	<-t.dead
	t.m.Lock()
	reason := t.reason
	t.m.Unlock()
	return reason
}

// Go runs f in a new goroutine and tracks its termination.
//
// If f returns a non-nil error, t.Kill is called with that
// error as the death reason parameter.
//
// It is f's responsibility to monitor the tomb and return
// appropriately once it is in a dying state.
//
// It is safe for the f function to call the Go method again
// to create additional tracked goroutines. Once all tracked
// goroutines return, the Dead channel is closed and the
// Wait method unblocks and returns the death reason.
//
// Calling the Go method after all tracked goroutines return
// causes a runtime panic. For that reason, calling the Go
// method a second time out of a tracked goroutine is unsafe.
func (t *mockTomb) Go(f func() error) {
	t.init()
	t.m.Lock()
	defer t.m.Unlock()
	select {
	case <-t.dead:
		panic("tomb.Go called after all goroutines terminated")
	default:
	}
	t.alive++
	go t.run(f)
}

func (t *mockTomb) run(f func() error) {
	err := f()
	t.m.Lock()
	defer t.m.Unlock()
	t.alive--
	if t.alive == 0 || err != nil {
		t.kill(err)
		if t.alive == 0 {
			close(t.dead)
		}
	}
}

// Kill puts the tomb in a dying state for the given reason,
// closes the Dying channel, and sets Alive to false.
//
// Althoguh Kill may be called multiple times, only the first
// non-nil error is recorded as the death reason.
//
// If reason is ErrDying, the previous reason isn't replaced
// even if nil. It's a runtime error to call Kill with ErrDying
// if t is not in a dying state.
func (t *mockTomb) Kill(reason error) {
	t.init()
	t.m.Lock()
	defer t.m.Unlock()
	t.kill(reason)
}

func (t *mockTomb) kill(reason error) {
	if reason == ErrStillAlive {
		panic("tomb: Kill with ErrStillAlive")
	}
	if reason == ErrDying {
		if t.reason == ErrStillAlive {
			panic("tomb: Kill with ErrDying while still alive")
		}
		return
	}
	if t.reason == ErrStillAlive {
		t.reason = reason
		close(t.dying)
		for _, child := range t.child {
			child.cancel()
		}
		t.child = nil
		return
	}
	if t.reason == nil {
		t.reason = reason
		return
	}
}

// Killf calls the Kill method with an error built providing the received
// parameters to fmt.Errorf. The generated error is also returned.
func (t *mockTomb) Killf(f string, a ...interface{}) error {
	err := fmt.Errorf(f, a...)
	t.Kill(err)
	return err
}

// Err returns the death reason, or ErrStillAlive if the tomb
// is not in a dying or dead state.
func (t *mockTomb) Err() (reason error) {
	t.init()
	t.m.Lock()
	reason = t.reason
	t.m.Unlock()
	return
}

// Alive returns true if the tomb is not in a dying or dead state.
func (t *mockTomb) Alive() bool {
	return t.Err() == ErrStillAlive
}
