// Copyright (C) 2022 Print Tracker, LLC - All Rights Reserved
//
// Unauthorized copying of this file, via any medium is strictly prohibited
// as this source code is proprietary and confidential. Dissemination of this
// information or reproduction of this material is strictly forbidden unless
// prior written permission is obtained from Print Tracker, LLC.

package reroutine

// Go starts the function do in a go-routine and restarts it only if it panics
// until the stop channel is closed. If the go-routine returns without panic,
// then it is not restarted. This function returns immediately.
func Go(stopChan <-chan struct{}, do func()) {
	go BlockingGo(stopChan, do)
}

// BlockingGo is the same as Go but does not return until the provided function
// returns without panicking or the context is cancelled.
func BlockingGo(stopChan <-chan struct{}, do func()) {
	start := make(chan struct{})
	go func() {
		start <- struct{}{}
	}()
	for {
		select {
		case <-stopChan:
			return
		case _, ok := <-start:
			if !ok {
				return
			}
			go func() {
				defer HandleCrash(func(_ interface{}) {
					start <- struct{}{}
				})
				do()
				close(start)
			}()
		}
	}
}

// Tomb is the minimum required interface to operate reroutine against a Tomb instance
type Tomb interface {
	// Dying returns the channel that can be used to wait until the tomb is killed.
	Dying() <-chan struct{}
	// Go runs f in a new goroutine and tracks its termination.
	Go(func() error)
}

// GoTomb is similar to Go except that it operates using a tomb.Tomb instance instead of
// a context.
func GoTomb(ts Tomb, do func() error) {
	go BlockingGoTomb(ts, do)
}

// BlockingGoTomb is like GoTomb but does not return until the provided function
// returns without panicking or the context is cancelled.
func BlockingGoTomb(ts Tomb, do func() error) {
	start := make(chan struct{})
	go func() {
		start <- struct{}{}
	}()
	for _ = range start {
		select {
		case <-ts.Dying():
			return
		default:
		}
		ts.Go(func() error {
			defer HandleCrash(func(_ interface{}) {
				start <- struct{}{}
			})
			err := do()
			// Function completed without panic, don't restart
			close(start)
			return err
		})
	}
}
