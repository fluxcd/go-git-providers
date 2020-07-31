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
	"fmt"
	"testing"
)

const multiErrorStr = `multiple errors occurred: 
- validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid
- validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid
- validation error for Foo.Bar: field is required`

func Test_validator_Error(t *testing.T) {
	myCustomValidationErr := fmt.Errorf("value cannot contain dashes: %w", ErrFieldInvalid)
	tests := []struct {
		name           string
		structName     string
		errs           []error
		usageFunc      func(*validator)
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
			usageFunc: func(errs *validator) {
				errs.Append(nil, nil, "Bar")
			},
			errs:         []error{nil, nil, nil},
			expectedErrs: nil,
		},
		{
			name:       "one required, one existing nil error ignored",
			structName: "Foo",
			usageFunc: func(errs *validator) {
				errs.Required("Bar")
			},
			errs:           []error{nil},
			expectedErrs:   []error{ErrFieldRequired},
			expectedErrStr: "validation error for Foo.Bar: field is required",
		},
		{
			name:       "one invalid, many field paths",
			structName: "Foo",
			usageFunc: func(errs *validator) {
				errs.Invalid("myvalue", "Bar", "Baz", "Hey", "There")
			},
			expectedErrs:   []error{ErrFieldInvalid},
			expectedErrStr: "validation error for Foo.Bar.Baz.Hey.There (value: myvalue): field is invalid",
		},
		{
			name:       "one invalid, using a custom error",
			structName: "Foo",
			usageFunc: func(errs *validator) {
				errs.Append(myCustomValidationErr, "my-value", "Bar")
			},
			expectedErrs:   []error{ErrFieldInvalid},
			expectedErrStr: "validation error for Foo.Bar (value: my-value): value cannot contain dashes: field is invalid",
		},
		{
			name:       "return multiple errors",
			structName: "Foo",
			usageFunc: func(errs *validator) {
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
			v := New(tt.structName).(*validator)
			v.errs = tt.errs
			// Run the usage func if specified
			if tt.usageFunc != nil {
				tt.usageFunc(v)
			}
			err := v.Error()
			// Make sure the error embeds the following expected errors
			TestExpectErrors(t, "validator.Error", err, tt.expectedErrs...)
			// Make sure the error string matches the expected one
			if err != nil {
				errStr := err.Error()
				if errStr != tt.expectedErrStr {
					t.Errorf("validator.Error() error string = %q, wanted %q", errStr, tt.expectedErrStr)
				}
			}
		})
	}
}

type fakeValidateTarget struct {
	errs []error
}

func (t *fakeValidateTarget) ValidateFields(v Validator) {
	for _, err := range t.errs {
		v.Append(err, nil)
	}
}

func TestValidateTargets(t *testing.T) {
	tests := []struct {
		name         string
		structName   string
		targets      []ValidateTarget
		expectedErrs []error
	}{
		{
			name:       "multiple passing",
			structName: "IdentityRef",
			targets: []ValidateTarget{
				&fakeValidateTarget{},
				&fakeValidateTarget{},
			},
		},
		{
			name:       "two failing for one specific",
			structName: "IdentityRef",
			targets: []ValidateTarget{
				&fakeValidateTarget{[]error{ErrFieldRequired, ErrFieldInvalid}},
				&fakeValidateTarget{},
			},
			expectedErrs: []error{&MultiError{}, ErrFieldRequired, ErrFieldInvalid},
		},
		{
			name:       "two failing, one for each",
			structName: "IdentityRef",
			targets: []ValidateTarget{
				&fakeValidateTarget{[]error{ErrFieldInvalid}},
				&fakeValidateTarget{[]error{ErrFieldRequired}},
			},
			expectedErrs: []error{&MultiError{}, ErrFieldRequired, ErrFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargets(tt.structName, tt.targets...)
			TestExpectErrors(t, "ValidateTargets", err, tt.expectedErrs...)
		})
	}
}
