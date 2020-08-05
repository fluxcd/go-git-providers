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

import "github.com/fluxcd/go-git-providers/validation"

const (
	// by default, deploy keys are read-only
	defaultDeployKeyReadOnly = true
)

// RepositoryCredential is a credential that allows access (either read-only or read-write) to the repo
type RepositoryCredential interface {
	Object

	// GetRepositoryRef gets the repository that this credential is associated with
	GetRepositoryRef() *RepositoryRef

	// GetType returns the type of the credential
	GetType() RepositoryCredentialType

	// GetName returns a name (or title/description) of the credential
	GetName() string

	// GetData returns the key that will be authorized to access the repo, this can e.g. be a SSH public key
	GetData() []byte

	// IsReadOnly returns whether this credential is authorized to write to the repository or not
	IsReadOnly() bool
}

// DeployKey implements the RepositoryCredential interface.
// DeployKey can be created and deleted
var _ RepositoryCredential = &DeployKey{}
var _ Creatable = &DeployKey{}
var _ Deletable = &DeployKey{}

// DeployKey represents a short-lived credential (e.g. an SSH public key) used for accessing a repository
type DeployKey struct {
	// DeployKey embeds InternalHolder for accessing the underlying object
	// +optional
	InternalHolder `json:",inline"`

	// Name is the human-friendly interpretation of what the key is for (and does)
	// +required
	Name string `json:"name"`

	// Key specifies the public part of the deploy (e.g. SSH) key
	// +required
	Key []byte `json:"key"`

	// ReadOnly specifies whether this DeployKey can write to the repository or not
	// Default value at POST-time: true
	// +optional
	ReadOnly *bool `json:"readOnly"`

	// Repository specifies the information about what repository this deploy key is associated with.
	// It is populated in .Get() and .List() calls.
	// When creating, this field is optional. However, if specified, it must match the RepositoryRef
	// given to the client
	// +optional
	Repository *RepositoryRef `json:"repository"`
}

// GetRepositoryRef returns the RepositoryRef for this DeployKey
// Make sure to nil-check this before using, as it's an optional field set at GET time
func (dk *DeployKey) GetRepositoryRef() *RepositoryRef {
	return dk.Repository
}

// GetType returns the RepositoryCredentialType for this DeployKey
func (dk *DeployKey) GetType() RepositoryCredentialType {
	return RepositoryCredentialTypeDeployKey
}

// GetName returns the name (or title/description) for this DeployKey
func (dk *DeployKey) GetName() string {
	return dk.Name
}

// GetData returns the SSH public key that can access the repository for this DeployKey
func (dk *DeployKey) GetData() []byte {
	return dk.Key
}

// IsReadOnly returns whether this deploy key has read-only or read-write access to the repo
func (dk *DeployKey) IsReadOnly() bool {
	if dk.ReadOnly == nil {
		return defaultDeployKeyReadOnly
	}
	return *dk.ReadOnly
}

// Default defaults the DeployKey, implementing the Creatable interface
func (dk *DeployKey) Default() {
	if dk.ReadOnly == nil {
		dk.ReadOnly = BoolVar(defaultDeployKeyReadOnly)
	}
}

// ValidateCreate validates the object at POST-time and implements the Creatable interface
func (dk *DeployKey) ValidateCreate() error {
	validator := validation.New("DeployKey")
	// Common validation
	dk.validateNameAndRepository(validator)
	// Key is a required field
	if len(dk.Key) == 0 {
		validator.Required("Key")
	}
	// Don't care about the RepositoryRef, as that information is coming from
	// the RepositoryClient. In the client, we make sure that they equal.
	return validator.Error()
}

// ValidateDelete validates the object at DELETE-time and implements the Deletable interface
func (dk *DeployKey) ValidateDelete() error {
	validator := validation.New("DeployKey")
	// Common validation
	dk.validateNameAndRepository(validator)
	return validator.Error()
}

func (dk *DeployKey) validateNameAndRepository(validator validation.Validator) {
	// Make sure we've set the name of the team
	if len(dk.Name) == 0 {
		validator.Required("Name")
	}
	// Validate the Repository if it is set. It most likely _shouldn't be_ (there's no need to,
	// as it's only set at GET-time), but if it is, make sure fields are ok. The RepositoryClient
	// should make sure that if set, it also needs to match the client's RepositoryRef.
	if dk.Repository != nil {
		dk.Repository.ValidateFields(validator)
	}
}
