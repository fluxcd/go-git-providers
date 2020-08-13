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
	"github.com/fluxcd/go-git-providers/validation"
)

// MakeRepositoryCreateOptions returns a RepositoryCreateOptions based off the mutator functions
// given to e.g. RepositoriesClient.Create(). The returned validation error may be ignored in the
// case that the client allows e.g. other license templates than those that are common.
// validation.ErrFieldEnumInvalid is returned if the license template doesn't match known values.
func MakeRepositoryCreateOptions(opts ...RepositoryCreateOption) (RepositoryCreateOptions, error) {
	o := &RepositoryCreateOptions{}
	for _, opt := range opts {
		opt.ApplyToRepositoryCreateOptions(o)
	}
	return *o, o.ValidateOptions()
}

// RepositoryReconcileOption is an interface for applying options to when reconciling repositories.
type RepositoryReconcileOption interface {
	// RepositoryCreateOption is embedded, as reconcile uses the create options.
	RepositoryCreateOption
}

// RepositoryCreateOption is an interface for applying options to when creating repositories.
type RepositoryCreateOption interface {
	// ApplyToRepositoryCreateOptions should apply relevant options to the target.
	ApplyToRepositoryCreateOptions(target *RepositoryCreateOptions)
}

// RepositoryCreateOptions specifies optional options when creating a repository.
type RepositoryCreateOptions struct {
	// AutoInit can be set to true in order to automatically initialize the Git repo with a
	// README.md and optionally a license in the first commit.
	// Default: nil (which means "false, don't create")
	AutoInit *bool

	// LicenseTemplate lets the user specify a license template to use when AutoInit is true.
	// Default: nil.
	// Available options: See the LicenseTemplate enum.
	LicenseTemplate *LicenseTemplate
}

// ApplyToRepositoryCreateOptions applies the options defined in the options struct to the
// target struct that is being completed.
func (opts *RepositoryCreateOptions) ApplyToRepositoryCreateOptions(target *RepositoryCreateOptions) {
	// Go through each field in opts, and apply it to target if set
	if opts.AutoInit != nil {
		target.AutoInit = opts.AutoInit
	}
	if opts.LicenseTemplate != nil {
		target.LicenseTemplate = opts.LicenseTemplate
	}
}

// ValidateInfo validates that the options are valid.
func (opts *RepositoryCreateOptions) ValidateOptions() error {
	errs := validation.New("RepositoryCreateOptions")
	if opts.LicenseTemplate != nil {
		errs.Append(ValidateLicenseTemplate(*opts.LicenseTemplate), *opts.LicenseTemplate, "LicenseTemplate")
	}
	return errs.Error()
}
