# Reroutine

[![codecov](https://codecov.io/gh/clarkmcc/go-reroutine/branch/master/graph/badge.svg?token=aTphaWyObN)](https://codecov.io/gh/clarkmcc/go-reroutine)

Easily restart go-routines when they panic. This package provides an easy way to restart a go-routine when it panics but to ignore a restart if it returned normally. It behaves identically to `go func()` but re-calls the provided function if the function panicked. This is useful for long-running worker routines that don't maintain their own state.

## Installation
    go get github.com/clarkmcc/go-reroutine

## Example
### Basic
Calling `reroutine.Go` behaves exactly like calling a function inside a regular go-routine (`go func()`) where the operation is non-blocking. In the following example, the provided function will be restarted on panic until the stop channel is closed.
```go
stop := make(chan struct{})
reroutine.Go(stop, func() {
  // Do something here that could panic and should be resumed on panic
})
```

### Full
The following example illustrates how a go-routine can panic and restart to continue its process of incrementing `i`. In this case, the go-routine closes the stop channel when its incremented `i` three times, and panics on every iteration.
```go
// Create a counter. Once this gets to 3 we want to stop restarting. Until then, we want to
// panic on every increment to prove that we're restarting the go-routine through panics and
// continuing the work.
i := int64(0)

reroutine.Go(stop, func() {
  for {
    // Increment until we get to 3, then stop restarting
    if atomic.AddInt64(&i, 1) == 3 {
      close(stop)
    }
		
    // Panic on every iteration
    panic("panicked")
  }
})

// Make sure that the incrementation was performed
if atomic.LoadInt64(&i) != 3 {
  panic("expected three iterations")
}
```

### Blocking Restart
Sometimes it might be useful to block until the panicking go-routine is able to successfully complete.
```go
stop := make(chan struct{})
reroutine.BlockingGo(stop, func() {
  // Do something that we should wait for
})
```
