package reroutine

import (
	"fmt"
	"log"
	"runtime"
)

var (
	// ReallyCrash controls the behavior of HandleCrash and now defaults
	// true. It's still exposed so components can optionally set to false
	// to restore prior behavior.
	ReallyCrash = false
	PrintError  = func(str string) {
		log.Print(str)
	}
)

// PanicHandlers is a list of functions which will be invoked when a panic happens.
var PanicHandlers = []func(interface{}){logPanic}

// HandleCrash simply catches a crash and logs an error. Meant to be called via
// defer.  Additional context-specific handlers can be provided, and will be
// called in case of panic.  HandleCrash actually crashes, after calling the
// handlers and logging the panic message.
//
// E.g., you can provide one or more additional handlers for something like shutting down go routines gracefully.
func HandleCrash(additionalHandlers ...func(interface{})) {
	if r := recover(); r != nil {
		for _, fn := range PanicHandlers {
			fn(r)
		}
		for _, fn := range additionalHandlers {
			fn(r)
		}
		if ReallyCrash {
			// Actually proceed to panic.
			panic(r)
		}
	}
}

// logPanic logs the caller tree when a panic occurs (except in the special case of http.ErrAbortHandler).
func logPanic(r interface{}) {
	// Same as stdlib http server code. Manually allocate stack trace buffer size
	// to prevent excessively large logs
	const size = 64 << 10
	stacktrace := make([]byte, size)
	stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]
	if _, ok := r.(string); ok {
		PrintError(fmt.Sprintf("Observed a panic: %s\n%s", r, stacktrace))
	} else {
		PrintError(fmt.Sprintf("Observed a panic: %#v (%v)\n%s", r, r, stacktrace))
	}
}
