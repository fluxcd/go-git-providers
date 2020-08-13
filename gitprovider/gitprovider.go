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

import "context"

// ProviderID is a typed string for a given Git provider
// The provider constants are defined in their respective packages.
type ProviderID string

// CreatableInfo is an interface which all *Info objects that can be created
// (using the Client) should implement.
type CreatableInfo interface {
	// ValidateInfo validates the object at {Object}.Set() and POST-time, before defaulting.
	// Set (non-nil) and required fields should be validated.
	ValidateInfo() error
	// Default will be run after validation, setting optional pointer fields to their
	// default values before doing the POST request.
	Default()
}

// GenericUpdatable is an interface which all objects that can be updated
// using the Client implement.
type GenericUpdatable interface {
	// Update will apply the desired state in this object to the server.
	// Only set fields will be respected (i.e. PATCH behaviour).
	// In order to apply changes to this object, use the .Set({Resource}Info) error
	// function, or cast .APIObject() to a pointer to the provider-specific type
	// and set custom fields there.
	//
	// ErrNotFound is returned if the resource does not exist.
	//
	// The internal API object will be overridden with the received server data.
	Update(ctx context.Context) error
}

// GenericDeletable is an interface which all objects that can be deleted
// using the Client implement.
type GenericDeletable interface {
	// Delete deletes the current resource irreversible.
	//
	// ErrNotFound is returned if the resource doesn't exist anymore.
	Delete(ctx context.Context) error
}

// GenericReconcilable is an interface which all objects that can be reconciled
// using the Client implement.
type GenericReconcilable interface {
	// Reconcile makes sure the desired state in this object (called "req" here) becomes
	// the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	//
	// The internal API object will be overridden with the received server data if actionTaken == true.
	Reconcile(ctx context.Context) (actionTaken bool, err error)
}

// Object is the interface all types should implement.
type Object interface {
	// APIObject returns the underlying value that was returned from the server.
	APIObject() interface{}
}

// OrganizationBound describes an object that is bound to a given organization, e.g. a team.
type OrganizationBound interface {
	// Organization returns the OrganizationRef associated with this object.
	Organization() OrganizationRef
}

// RepositoryBound describes an object that is bound to a given repository, e.g. a deploy key.
type RepositoryBound interface {
	// Repository returns the RepositoryRef associated with this object.
	Repository() RepositoryRef
}
