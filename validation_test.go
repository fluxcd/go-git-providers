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
	"testing"
)

const multiErrorStr = `multiple errors occurred: 
- validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid
- validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid
- validation error for Foo.Bar: field is required`

func Test_validationErrorList_Error(t *testing.T) {
	tests := []struct {
		name           string
		structName     string
		errs           []error
		usageFunc      func(*validationErrorList)
		expectedErr    error
		expectedErrStr string
	}{
		{
			name:        "nil errors",
			errs:        nil,
			expectedErr: nil,
		},
		{
			name:        "zero errors",
			errs:        []error{},
			expectedErr: nil,
		},
		{
			name:        "many nil errors",
			errs:        []error{nil, nil, nil},
			expectedErr: nil,
		},
		{
			name:       "append nil error",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Append(nil, nil, "Bar")
			},
			errs:        []error{nil, nil, nil},
			expectedErr: nil,
		},
		{
			name:       "one required, one existing nil error ignored",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Required("Bar")
			},
			errs:           []error{nil},
			expectedErr:    ErrFieldRequired,
			expectedErrStr: "validation error for Foo.Bar: field is required",
		},
		{
			name:       "one invalid, many field paths",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				errs.Invalid("myvalue", "Bar", "Baz", "Hey", "There")
			},
			expectedErr:    ErrFieldInvalid,
			expectedErrStr: "validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid",
		},
		{
			name:       "one invalid, using more context and append",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				validationErr := fmt.Errorf("value cannot contain dashes: %w", ErrFieldInvalid)
				errs.Append(validationErr, "my-value", "Bar")
			},
			expectedErr:    ErrFieldInvalid,
			expectedErrStr: "validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid",
		},
		{
			name:       "return multiple errors",
			structName: "Foo",
			usageFunc: func(errs *validationErrorList) {
				validationErr := fmt.Errorf("value cannot contain dashes: %w", ErrFieldInvalid)
				errs.Append(validationErr, "my-value", "Bar")
				errs.Invalid("myvalue", "Bar", "Baz", "Hey", "There")
				errs.Required("Bar")
			},
			expectedErr:    &MultiError{},
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
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("validationErrorList.Error() error = %v, wantErr %v", err, tt.expectedErr)
			}
			if err != nil {
				errStr := err.Error()
				if errStr != tt.expectedErrStr {
					t.Errorf("validationErrorList.Error() error string = %q, wanted %q", errStr, tt.expectedErrStr)
				}
			}
		})
	}
}
