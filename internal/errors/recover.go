package errors

import (
	"fmt"
	"runtime/debug"
)

// PanicError wraps a recovered panic with stack trace
type PanicError struct {
	Value      interface{} // Panic value
	StackTrace string      // Stack trace at panic
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v\n%s", e.Value, e.StackTrace)
}

// SafeGo runs a function in a goroutine with panic recovery
// Any panics are converted to errors and sent to the error channel
func SafeGo(fn func() error, errChan chan<- error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := &PanicError{
					Value:      r,
					StackTrace: string(debug.Stack()),
				}
				select {
				case errChan <- err:
				default:
					// Error channel full or closed, panic was lost
				}
			}
		}()

		if err := fn(); err != nil {
			select {
			case errChan <- err:
			default:
				// Error channel full or closed, error was lost
			}
		}
	}()
}

// Recover wraps a function with panic recovery
// Returns any panic as a PanicError
func Recover(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{
				Value:      r,
				StackTrace: string(debug.Stack()),
			}
		}
	}()
	return fn()
}

// RecoverWithResult wraps a function with panic recovery and result
func RecoverWithResult[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			result = zero
			err = &PanicError{
				Value:      r,
				StackTrace: string(debug.Stack()),
			}
		}
	}()
	return fn()
}
