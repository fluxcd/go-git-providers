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
	"net/url"
	"reflect"
	"testing"
)

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
			errors:     []error{ErrFieldEnumInvalid, ErrFieldRequired},
			expectedOk: true,
		},
		{
			name:        "cast to custom type embedded in error list, success",
			errors:      []error{ErrFieldEnumInvalid, ErrFieldRequired, targetVal},
			expectedVal: targetVal,
			expectedOk:  true,
		},
		{
			name:       "cast to custom type not in error list, fail",
			errors:     []error{ErrFieldEnumInvalid, ErrFieldRequired},
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

func TestMultiError_Is(t *testing.T) {
	type fields struct {
		Errors []error
	}
	type args struct {
		target error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "has wrapped required error",
			fields: fields{
				Errors: []error{ErrFieldInvalid, fmt.Errorf("wrapped with cause: %w", ErrFieldRequired)},
			},
			args: args{
				target: ErrFieldRequired,
			},
			want: true,
		},
		{
			name: "doesn't have error",
			fields: fields{
				Errors: []error{ErrFieldInvalid, fmt.Errorf("wrapped with cause: %w", ErrFieldEnumInvalid)},
			},
			args: args{
				target: ErrFieldRequired,
			},
			want: false,
		},
		{
			name: "is multierror",
			fields: fields{
				Errors: []error{ErrFieldInvalid, fmt.Errorf("wrapped with cause: %w", ErrFieldEnumInvalid)},
			},
			args: args{
				target: &MultiError{},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &MultiError{
				Errors: tt.fields.Errors,
			}
			if got := e.Is(tt.args.target); got != tt.want {
				t.Errorf("MultiError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ExpectErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedErrs []error
		errCount     uint8
	}{
		{
			name:         "normal wrapping",
			err:          fmt.Errorf("wrapped error: %w", ErrFieldInvalid),
			expectedErrs: []error{ErrFieldInvalid},
		},
		{
			name:         "multierror wrapping",
			err:          &MultiError{Errors: []error{ErrFieldInvalid, ErrFieldEnumInvalid}},
			expectedErrs: []error{&MultiError{}, ErrFieldInvalid, ErrFieldEnumInvalid},
		},
		{
			name:         "one error not wrapped",
			err:          fmt.Errorf("wrapped error: %w", ErrFieldInvalid),
			expectedErrs: []error{ErrFieldEnumInvalid},
			errCount:     1,
		},
		{
			name:         "two different fmt.Errors should not be equal",
			err:          fmt.Errorf("wrapped error: %w", ErrFieldInvalid),
			expectedErrs: []error{fmt.Errorf("other error: %w", ErrFieldInvalid)},
			errCount:     1,
		},
		{
			name:         "two different fmt.Errors should not be equal",
			err:          fmt.Errorf("wrapped error: %w", ErrFieldInvalid),
			expectedErrs: []error{fmt.Errorf("other error")},
			errCount:     1,
		},
		{
			name:         "two different errorStrings should not match",
			err:          errors.New("first error"),
			expectedErrs: []error{errors.New("other error")},
			errCount:     1,
		},
		{
			name:         "two errors not found in multierror",
			err:          &MultiError{Errors: []error{ErrFieldInvalid}},
			expectedErrs: []error{&MultiError{}, ErrFieldRequired, ErrFieldEnumInvalid},
			errCount:     2,
		},
		{
			name:     "expected no error, got one",
			err:      fmt.Errorf("wrapped error: %w", ErrFieldInvalid),
			errCount: 1,
		},
		{
			name: "expected no error, got none",
			err:  nil,
		},
		{
			name:         "struct types",
			err:          &url.Error{Op: "foo", URL: "bar", Err: fmt.Errorf("baz")},
			expectedErrs: []error{&url.Error{}},
		},
		{
			name:         "multierror struct propagation",
			err:          &MultiError{Errors: []error{&url.Error{}}},
			expectedErrs: []error{&MultiError{}, &url.Error{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t2 := &fakeT{t, 0}
			TestExpectErrors(t2, "", tt.err, tt.expectedErrs...)
			if t2.calledErrorf != tt.errCount {
				t.Errorf("TestExpectErrors() errCount = %d, wanted %d", t2.calledErrorf, tt.errCount)
			}
		})
	}
}

type fakeT struct {
	*testing.T
	calledErrorf uint8
}

func (t *fakeT) Errorf(format string, args ...interface{}) {
	t.calledErrorf++
}
