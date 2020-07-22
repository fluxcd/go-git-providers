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
	"errors"
	"fmt"
	"reflect"
	"testing"
)

const multiErrorStr = `multiple errors occurred: 
- validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid
- validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid
- validation error for Foo.Bar: field is required`

func Test_validationErrorList_Error(t *testing.T) {
	myCustomValidationErr := fmt.Errorf("value cannot contain dashes: %w", ErrFieldInvalid)
	tests := []struct {
		name           string
		structName     string
		errs           []error
		usageFunc      func(*validationErrorList)
		expectedErrs   []error
		expectedErrStr string
	}{
		{
			name:         "nil errors",
			errs:         nil,
			expectedErrs: nil,
		},
		{
			name:         "zero errors",
			errs:         []error{},
			expectedErrs: nil,
		},
		{
			name:         "many nil errors",
			errs:         []error{nil, nil, nil},
			expectedErrs: nil,
		},
		{
			name:       "append nil error",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Append(nil, nil, "Bar")
			},
			errs:         []error{nil, nil, nil},
			expectedErrs: nil,
		},
		{
			name:       "one required, one existing nil error ignored",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Required("Bar")
			},
			errs:           []error{nil},
			expectedErrs:   []error{ErrFieldRequired},
			expectedErrStr: "validation error for Foo.Bar: field is required",
		},
		{
			name:       "one invalid, many field paths",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Invalid("myvalue", "Bar", "Baz", "Hey", "There")
			},
			expectedErrs:   []error{ErrFieldInvalid},
			expectedErrStr: "validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid",
		},
		{
			name:       "one invalid, using a custom error",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Append(myCustomValidationErr, "my-value", "Bar")
			},
			expectedErrs:   []error{ErrFieldInvalid},
			expectedErrStr: "validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid",
		},
		{
			name:       "return multiple errors",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Append(myCustomValidationErr, "my-value", "Bar")
				errs.Invalid("myvalue", "Bar", "Baz", "Hey", "There")
				errs.Required("Bar")
			},
			// We expect errors.Is to return true for all of these types
			expectedErrs:   []error{&MultiError{}, ErrFieldInvalid, ErrFieldRequired, myCustomValidationErr},
			expectedErrStr: multiErrorStr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := newValidationErrorList(tt.structName)
			el.errs = tt.errs
			// Run the usage func if specified
			if tt.usageFunc != nil {
				tt.usageFunc(el)
			}
			err := el.Error()
			// Loop through all expected errors and make sure that errors.Is returns true
			for _, expectedErr := range tt.expectedErrs {
				if !errors.Is(err, expectedErr) {
					t.Errorf("validationErrorList.Error() error = %v, wantErr %v", err, expectedErr)
				}
			}
			// Make sure the error string matches the expected one
			if err != nil {
				errStr := err.Error()
				if errStr != tt.expectedErrStr {
					t.Errorf("validationErrorList.Error() error string = %q, wanted %q", errStr, tt.expectedErrStr)
				}
			}
		})
	}
}

type customErrorType struct {
	data string
}

func (e *customErrorType) Error() string {
	return fmt.Sprintf("I'm custom, with data: %s", e.data)
}

func TestMultiError_As(t *testing.T) {
	targetVal := &customErrorType{data: "foo"}
	tests := []struct {
		name        string
		errors      []error
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			name:       "cast to MultiError, containing data, success",
			errors:     []error{ErrAlreadyExists, ErrDomainUnsupported},
			expectedOk: true,
		},
		{
			name:        "cast to custom type embedded in error list, success",
			errors:      []error{ErrAlreadyExists, ErrDomainUnsupported, targetVal},
			expectedVal: targetVal,
			expectedOk:  true,
		},
		{
			name:       "cast to custom type not in error list, fail",
			errors:     []error{ErrAlreadyExists, ErrDomainUnsupported},
			expectedOk: false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := &MultiError{
				Errors: tt.errors,
			}
			// Ideally, there would be a tt.target embedded which could let us only have one
			// errors.As() call, but due to how errors.As() works, we need to call it with the
			// exact type of the error it should be casted into (not behind a generic interface{}),
			// so hence we need to split this up per-case
			var got bool
			var target interface{}
			switch i {
			case 0:
				localTarget := &MultiError{}
				got = errors.As(multiErr, &localTarget)
				target = localTarget
			case 1:
				localTarget := &customErrorType{}
				got = errors.As(multiErr, &localTarget)
				target = localTarget
			case 2:
				localTarget := &customErrorType{}
				got = errors.As(multiErr, &localTarget)
				target = localTarget
			}

			// Make sure the return value was expected
			if got != tt.expectedOk {
				t.Errorf("errors.As(%T, %T) = %v, want %v", multiErr, target, got, tt.expectedOk)
			}
			// Make sure target was updated to include the right struct data
			if tt.expectedVal != nil && !reflect.DeepEqual(target, tt.expectedVal) {
				t.Errorf("Expected errors.As() to populate target (%v) to equal %v", target, tt.expectedVal)
			}
		})
	}
}
