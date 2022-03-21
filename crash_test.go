package reroutine

import (
	"testing"
)

func TestLogPanic_Types(t *testing.T) {
	logPanic("foobar")
	logPanic(10)
}
