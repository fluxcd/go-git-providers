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

import "fmt"

// TODO: Comment and unit-test this file

func newValidationErrorList(name string) *validationErrorList {
	return &validationErrorList{name, nil}
}

type validationErrorList struct {
	name string
	errs []error
}

func (el *validationErrorList) Required(fieldName string) {
	el.Append(fieldName, nil, ErrFieldRequired)
}

func (el *validationErrorList) Invalid(fieldName string, value interface{}) {
	el.Append(fieldName, value, ErrFieldInvalid)
}

func (el *validationErrorList) Append(fieldName string, value interface{}, err error) {
	if err == nil {
		return
	}
	if len(fieldName) != 0 {
		fieldName = "." + fieldName
	}
	valStr := ""
	if value != nil {
		valStr = fmt.Sprintf(" (value: %v)", value)
	}
	el.errs = append(el.errs, fmt.Errorf("validation error for %s%s%s: %w", el.name, fieldName, valStr, err))
	return
}

func (el *validationErrorList) Error() error {
	if len(el.errs) == 0 {
		return nil
	}
	filteredErrs := make([]error, 0, len(el.errs))
	for _, err := range el.errs {
		if err != nil {
			filteredErrs = append(filteredErrs, err)
		}
	}
	// Return the same error
	if len(filteredErrs) == 1 {
		return filteredErrs[0]
	}
	return &MultipleErrors{Errors: filteredErrs}
}
