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
