/*
Copyright 2020 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"errors"
	"fmt"
	"testing"
)

// MultiError is a holder struct for multiple errors returned at once
// Each of the errors might wrap their own underlying error.
// In order to check whether an error returned from a function was a
// *MultiError, you can do:
//
// 		multiErr := &MultiError{}
// 		if errors.Is(err, multiErr) { // do things }
//
// In order to get the value of the *MultiError (embedded somewhere
// in the chain, in order to access the sub-errors), you can do:
//
// 		multiErr := &MultiError{}
// 		if errors.As(err, &multiErr) { // multiErr contains sub-errors, do things }
//
// It is also possible to access sub-errors from a MultiError directly, using
// errors.As and errors.Is. Example:
//
// 		multiErr := &MultiError{Errors: []error{ErrFieldRequired, ErrFieldInvalid}}
//		if errors.Is(multiErr, ErrFieldInvalid) { // will return true, as ErrFieldInvalid is contained }
//
//		type customError struct { data string }
//		func (e *customError) Error() string { return "custom" + data }
// 		multiErr := &MultiError{Errors: []error{ErrFieldRequired, &customError{"my-value"}}}
//		target := &customError{}
//		if errors.As(multiErr, &target) { // target.data will now be "my-value" }
type MultiError struct {
	Errors []error
}

// Error implements the error interface on the pointer type of MultiError.Error
// This enforces callers to always return &MultiError{} for consistency
func (e *MultiError) Error() string {
	errStr := ""
	for _, err := range e.Errors {
		errStr += fmt.Sprintf("\n- %s", err.Error())
	}
	return fmt.Sprintf("multiple errors occurred: %s", errStr)
}

// Is implements the interface used by errors.Is in order to check if two errors are the same.
// This function recursively checks all contained errors
func (e *MultiError) Is(target error) bool {
	// If target is a MultiError, return that target is a match
	_, ok := target.(*MultiError)
	if ok {
		return true
	}
	// Loop through the contained errors, and check if there is any of them that match
	// target. If so, return true.
	for _, err := range e.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// As implements the interface used by errors.As in order to get the value of an embedded
// struct error of this MultiError
func (e *MultiError) As(target interface{}) bool {
	// There is no need to check for if target is a MultiError, as it it would be, this function
	// wouldn't be called.

	// Loop through all the errors and run errors.As() on them. Exit when found
	for _, err := range e.Errors {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// TestExpectErrors loops through all expected errors and make sure that errors.Is returns true
// for all of them
func TestExpectErrors(t testing.TB, funcName string, err error, expectedErrs []error) {
	for _, expectedErr := range expectedErrs {
		if !errors.Is(err, expectedErr) {
			t.Errorf("%s() error = %v, wanted %v", funcName, err, expectedErr)
		}
	}
}
