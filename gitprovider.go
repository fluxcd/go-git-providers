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
)

var (
	// ErrProviderNoSupport describes that the provider doesn't support the requested feature
	ErrProviderNoSupport = errors.New("no provider support for this feature")
	// ErrDomainUnsupported describes the case where e.g. a Github provider used for trying to get
	// information from e.g. gitlab.com
	ErrDomainUnsupported = errors.New("the given client doesn't support handling requests for this domain")

	// ErrUnauthorized describes that that the request was unauthorized by the Git provider server
	ErrUnauthorized = errors.New("unauthorized request for this resource")
	// ErrRateLimited is returned if a request hit the rate limit of the Git provider server
	ErrRateLimited = errors.New("hit rate limit")

	// ErrNotTopLevelOrganization describes the case where it's mandatory to specify a top-level organization
	// (e.g. for accessing teams), but a sub-organization was passed as the OrganizationRef
	ErrNotTopLevelOrganization = errors.New("expected top-level organization, received sub-organization instead")

	// ErrAlreadyExists is returned by .Create() requests if the given resource already exists.
	// Use .Reconcile() instead if you want to idempotently create the resource
	ErrAlreadyExists = errors.New("resource already exists, cannot create object. Use Reconcile() to create it idempotently")
	// ErrNotFound is returned by .Get() and .Update() calls if the given resource doesn't exist
	ErrNotFound = errors.New("the requested resource was not found")

	// ErrFieldRequired specifies the case where a required field isn't populated at use time
	ErrFieldRequired = errors.New("field is required")
	// ErrFieldInvalid specifies the case where a field isn't populated in a valid manner
	ErrFieldInvalid = errors.New("field is invalid")
	// ErrFieldEnumInvalid specifies the case where the given value isn't part of the known values in the enum
	ErrFieldEnumInvalid = errors.New("field value isn't known to this enum")

	// ErrURLUnsupportedScheme is returned if an URL without the HTTPS scheme is parsed
	ErrURLUnsupportedScheme = errors.New("unsupported URL scheme, only HTTPS supported")
	// ErrURLUnsupportedParts is returned if an URL with fragment, query values and/or user information is parsed
	ErrURLUnsupportedParts = errors.New("URL cannot have fragments, query values nor user information")
	// ErrURLInvalid is returned if an URL is invalid when parsing
	ErrURLInvalid = errors.New("invalid organization or repository URL")
	// ErrURLMissingRepoName is returned if there is no repository name in the URL
	ErrURLMissingRepoName = errors.New("missing repository name")
)

// ProviderID is a typed string for a given Git provider
// The provider constants are defined in their respective packages
type ProviderID string

// Creatable is an interface which all objects that can be created
// (using the Client) should implement
type Creatable interface {
	// ValidateCreate will be run in every .Create() client call, before defaulting
	// Set (non-nil) and required fields should be validated
	ValidateCreate() error
	// Default will be run after validation, setting optional pointer fields to their
	// default values before doing the POST request
	Default()
}

// Updatable is an interface which all objects that can be updated
// (using the Client) should implement
type Updatable interface {
	// ValidateUpdate will be run in every .Update() client call
	// Set (non-nil) and required fields should be validated, if needed
	// No defaulting happens for update calls
	ValidateUpdate() error
}

// Deletable is an interface which all objects that can be deleted
// (using the Client) should implement
type Deletable interface {
	// ValidateDelete will be run in every .Delete() client call
	// Set (non-nil) and required fields should be validated, if needed
	// No defaulting happens for delete calls
	ValidateDelete() error
}

// Object is the interface all types should implement
type Object interface {
	// GetInternal returns the underlying struct that's used
	GetInternal() interface{}
}

// InternalHolder can be embedded into other structs to implement the Object interface
type InternalHolder struct {
	// Internal contains the underlying object.
	// +optional
	Internal interface{} `json:"-"`
}

// GetInternal implements the Object interface
func (ih InternalHolder) GetInternal() interface{} {
	return ih.Internal
}
