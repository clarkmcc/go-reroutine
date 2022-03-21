# go-reroutine

[![codecov](https://codecov.io/gh/clarkmcc/go-reroutine/branch/master/graph/badge.svg?token=aTphaWyObN)](https://codecov.io/gh/clarkmcc/go-reroutine)

Easily keep go-routines alive through panics.

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

### Advanced
The following example illustrates how a go-routine can panic and restart to continue its process of incrementing `i`. In this case, the go-routine closes the stop channel when its incremented `i` three times, and panics on every iteration.
```go
c
i := int64(0)

reroutine.Go(stop, func() {
    for {
        if atomic.AddInt64(&i, 1) == 3 {
            close(stop)
        }
        panic("panicked")
    }
})

if atomic.LoadInt64(&i) != 3 {
    panic("expected three iterations")
}
```