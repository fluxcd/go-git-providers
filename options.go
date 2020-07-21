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

// RepositoryCreateOptionsFunc is a function mutating the given RepositoryCreateOptions argument
// This function is passed a variable amount into RepositoriesClient.Create()
type RepositoryCreateOptionsFunc func(*RepositoryCreateOptions)

// WithRepositoryCreateOptions lets the user set the desired RepositoryCreateOptions arguments
func WithRepositoryCreateOptions(desired RepositoryCreateOptions) RepositoryCreateOptionsFunc {
	return func(opts *RepositoryCreateOptions) {
		*opts = desired
	}
}

// MakeRepositoryCreateOptions returns a RepositoryCreateOptions based off the mutator functions
// given to e.g. RepositoriesClient.Create().
func MakeRepositoryCreateOptions(fns ...RepositoryCreateOptionsFunc) RepositoryCreateOptions {
	opts := &RepositoryCreateOptions{}
	for _, fn := range fns {
		fn(opts)
	}
	return *opts
}

// RepositoryCreateOptions specifies optional options when creating a repository
type RepositoryCreateOptions struct {
	// AutoInit can be set to true in order to automatically initialize the Git repo with a
	// README.md and optionally a license in the first commit.
	// Default: false
	AutoInit *bool

	// LicenseTemplate lets the user specify a license template to use when AutoInit is true
	// Default: nil
	// Available options: See the LicenseTemplate enum
	LicenseTemplate *LicenseTemplate
}
