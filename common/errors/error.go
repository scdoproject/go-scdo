/**
*  @file
*  @copyright defined in scdo/LICENSE
 */

package errors

import (
	"errors"
	"fmt"
)

// New returns an error that formats as the given text.
func New(text string) error {
	return errors.New(text)
}

// scdoError represents a scdo error with code and message.
type scdoError struct {
	code ErrorCode
	msg  string
}

// scdoParameterizedError represents a scdo error with code and parameterized message.
// For type safe of common used business error, developer could define a concrete error to process.
type scdoParameterizedError struct {
	scdoError
	parameters []interface{}
}

func newScdoError(code ErrorCode, msg string) error {
	return &scdoError{code, msg}
}

// Error implements the error interface.
func (err *scdoError) Error() string {
	return err.msg
}

// Get returns a scdo error with specified error code.
func Get(code ErrorCode) error {
	err, found := constErrors[code]
	if !found {
		return fmt.Errorf("system internal error, cannot find the error code %v", code)
	}

	return err
}

// Create creates a scdo error with specified error code and parameters.
func Create(code ErrorCode, args ...interface{}) error {
	errFormat, found := parameterizedErrors[code]
	if !found {
		return fmt.Errorf("system internal error, cannot find the error code %v", code)
	}

	return &scdoParameterizedError{
		scdoError: scdoError{code, fmt.Sprintf(errFormat, args...)},
		parameters: args,
	}
}
