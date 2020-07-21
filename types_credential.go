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

const (
	// by default, deploy keys are read-only
	defaultDeployKeyReadOnly = true
)

// RepositoryCredential is a credential that allows access (either read-only or read-write) to the repo
type RepositoryCredential interface {
	Object

	// GetRepositoryRef gets the repository that this credential is associated with
	GetRepositoryRef() RepositoryRef

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
var _ Creator = &DeployKey{}
var _ Deletor = &DeployKey{}

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
	Repository RepositoryInfo `json:"repository"`
}

// GetType returns the RepositoryCredentialType for this DeployKey
func (dk *DeployKey) GetRepositoryRef() RepositoryRef {
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

// Default defaults the DeployKey, implementing the Creator interface
func (dk *DeployKey) Default() {
	if dk.ReadOnly == nil {
		dk.ReadOnly = boolVar(defaultDeployKeyReadOnly)
	}
}

// ValidateCreate validates the object at POST-time and implements the Creator interface
func (dk *DeployKey) ValidateCreate() error {
	errs := newValidationErrorList("DeployKey")
	if len(dk.Name) == 0 {
		errs.Required("Name")
	}
	if len(dk.Key) == 0 {
		errs.Required("Key")
	}
	// Don't care about the RepositoryInfo, as that information is coming from
	// the RepositoryClient. In the client, we make sure that they equal.
	return errs.Error()
}

// ValidateDelete validates the object at DELETE-time and implements the Deletor interface
func (dk *DeployKey) ValidateDelete() error {
	errs := newValidationErrorList("DeployKey")
	if len(dk.Name) == 0 {
		errs.Required("Name")
	}
	return errs.Error()
}
