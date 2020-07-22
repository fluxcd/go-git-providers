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

package gitprovider

import (
	"fmt"
	"strings"
)

// MultiError is a holder struct for multiple errors returned at once
// Each of the errors might wrap their own underlying error
type MultiError struct {
	Errors []error
}

// Error implements the error interface
func (e *MultiError) Error() string {
	errStr := ""
	for _, err := range e.Errors {
		errStr += fmt.Sprintf("\n- %s", err.Error())
	}
	return fmt.Sprintf("multiple errors occurred: %s", errStr)
}

func (e *MultiError) Is(target error) bool {
	_, ok := target.(*MultiError)
	return ok
}

// newValidationErrorList creates a new validationErrorList struct for the given struct name
func newValidationErrorList(name string) *validationErrorList {
	return &validationErrorList{name, nil}
}

// validationErrorList is a wrapper struct that helps with writing validation functions where many
// distinct errors might occur at the same time (e.g. for the same object). One alternative could be
// to return an error directly when found in validation, but that leaves the user with a fraction of
// the information needed to fix the problem. The Error() error method of this struct might return
// *MultiError to inform the user about all things that need fixing.
type validationErrorList struct {
	// name describes the name of the object being validated
	name string
	// errs is a list of errors that have occurred
	errs []error
}

// Required is a helper method for Append, registering ErrFieldRequired as the cause, along with what field
// caused the error. fieldPaths should contain the names of all nested sub-fields (of the struct) that caused
// the error.
func (el *validationErrorList) Required(fieldPaths ...string) {
	el.Append(ErrFieldRequired, nil, fieldPaths...)
}

// Invalid is a helper method for Append, registering ErrFieldInvalid as the cause, along with what field
// caused the error. fieldPaths should contain the names of all nested sub-fields (of the struct) that caused
// the error. Specifying the value that was invalid is also supported
func (el *validationErrorList) Invalid(value interface{}, fieldPaths ...string) {
	el.Append(ErrFieldInvalid, value, fieldPaths...)
}

// Append registers a validation error in the internal list, capturing the value and the field that
// caused the problem.
func (el *validationErrorList) Append(err error, value interface{}, fieldPaths ...string) {
	// If there wasn't an error, just return directly
	if err == nil {
		return
	}
	// Construct the path to the error-causing field as a dot-separated string, beginning with the name
	// of the struct
	fieldPath := strings.Join(append([]string{el.name}, fieldPaths...), ".")
	// Conditionally show the string-formatted value in the error message
	valStr := ""
	if value != nil {
		valStr = fmt.Sprintf(" (value: %v)", value)
	}
	// Append the error to the list, wrapping the underlying error
	el.errs = append(el.errs, fmt.Errorf("validation error for %s%s: %w", fieldPath, valStr, err))
}

// Error returns an aggregated error (or nil), based on the errors that have been registered
// A *MultiError is returned if there are multiple errors. Users of this function might use
// multiErr := &MultiError{}; errors.As(err, multiErr) or errors.Is(err, multiErr) to detect
// that many errors were returned
func (el *validationErrorList) Error() error {
	// If there aren't any errors in the list, return nil quickly
	if len(el.errs) == 0 {
		return nil
	}
	// Filter the errors to make sure they are non-nil, so no nil errors by accident
	// are counted
	filteredErrs := make([]error, 0, len(el.errs))
	for _, err := range el.errs {
		if err != nil {
			filteredErrs = append(filteredErrs, err)
		}
	}
	// If there aren't any non-nil errors, return nil
	if len(filteredErrs) == 0 {
		return nil
	}
	// If there is only one error in the filtered list, return that specific one
	if len(filteredErrs) == 1 {
		return filteredErrs[0]
	}
	// Otherwise, return all of the errors wrapped in a *MultiError
	return &MultiError{Errors: filteredErrs}
}
