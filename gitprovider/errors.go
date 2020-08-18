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
	"net/http"
	"time"
)

var (
	// ErrNoProviderSupport describes that the provider doesn't support the requested feature.
	ErrNoProviderSupport = errors.New("no provider support for this feature")
	// ErrDomainUnsupported describes the case where e.g. a GitHub provider used for trying to get
	// information from e.g. "gitlab.com".
	ErrDomainUnsupported = errors.New("the client doesn't support handling requests for this domain")

	// ErrNotTopLevelOrganization describes the case where it's mandatory to specify a top-level organization
	// (e.g. to access teams), but a sub-organization was passed as the OrganizationRef.
	ErrNotTopLevelOrganization = errors.New("expected top-level organization, received sub-organization instead")
	// ErrInvalidArgument describes a generic error where an invalid argument have been specified to a function.
	ErrInvalidArgument = errors.New("invalid argument specified")
	// ErrUnexpectedEvent describes a case where something really unexpected happened in the program.
	ErrUnexpectedEvent = errors.New("an unexpected error occurred")

	// ErrAlreadyExists is returned by .Create() requests if the given resource already exists.
	// Use .Reconcile() instead if you want to idempotently create the resource.
	ErrAlreadyExists = errors.New("resource already exists, cannot create object. Use Reconcile() to create it idempotently")
	// ErrNotFound is returned by .Get() and .Update() calls if the given resource doesn't exist.
	ErrNotFound = errors.New("the requested resource was not found")
	// ErrInvalidServerData is returned when the server returned invalid data, e.g. missing required fields in the response.
	ErrInvalidServerData = errors.New("got invalid data from server, don't know how to handle")

	// ErrURLUnsupportedScheme is returned if an URL without the HTTPS scheme is parsed.
	ErrURLUnsupportedScheme = errors.New("unsupported URL scheme, only HTTPS supported")
	// ErrURLUnsupportedParts is returned if an URL with fragment, query values and/or user information is parsed.
	ErrURLUnsupportedParts = errors.New("URL cannot have fragments, query values nor user information")
	// ErrURLInvalid is returned if an URL is invalid when parsing.
	ErrURLInvalid = errors.New("invalid organization, user or repository URL")
	// ErrURLMissingRepoName is returned if there is no repository name in the URL.
	ErrURLMissingRepoName = errors.New("missing repository name")

	// ErrInvalidClientOptions is the error returned when calling NewClient() with
	// invalid options (e.g. specifying mutually exclusive options).
	ErrInvalidClientOptions = errors.New("invalid options given to NewClient()")
	// ErrDestructiveCallDisallowed happens when the client isn't set up with WithDestructiveAPICalls()
	// but a destructive action is called.
	ErrDestructiveCallDisallowed = errors.New("destructive call was blocked, disallowed by client")
	// ErrInvalidTransportChainReturn is returned if a ChainableRoundTripperFunc returns nil, which is invalid.
	ErrInvalidTransportChainReturn = errors.New("the return value of a ChainableRoundTripperFunc must not be nil")
)

// HTTPError is an error that contains context about the HTTP request/response that failed.
type HTTPError struct {
	// HTTP response that caused this error.
	Response *http.Response `json:"-"`
	// Full error message, human-friendly and formatted.
	ErrorMessage string `json:"errorMessage"`
	// Message about what happened.
	Message string `json:"message"`
	// Where to find more information about the error.
	DocumentationURL string `json:"documentationURL"`
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return e.ErrorMessage
}

// RateLimitError is an error, extending HTTPError, that contains context about rate limits.
type RateLimitError struct {
	// RateLimitError extends HTTPError.
	HTTPError `json:",inline"`

	// The number of requests per hour the client is currently limited to.
	Limit int `json:"limit"`
	// The number of remaining requests the client can make this hour.
	Remaining int `json:"remaining"`
	// The timestamp at which point the current rate limit will reset.
	Reset time.Time `json:"reset"`
}

// ValidationError is an error, extending HTTPError, that contains context about failed server-side validation.
type ValidationError struct {
	// RateLimitError extends HTTPError.
	HTTPError `json:",inline"`

	// Errors contain context about what validation(s) failed.
	Errors []ValidationErrorItem `json:"errors"`
}

// ValidationErrorItem represents a single invalid field in an invalid request.
type ValidationErrorItem struct {
	// Resource on which the error occurred.
	Resource string `json:"resource"`
	// Field on which the error occurred.
	Field string `json:"field"`
	// Code for the validation error.
	Code string `json:"code"`
	// Message describing the error. Errors with Code == "custom" will always have this set.
	Message string `json:"message"`
}

// InvalidCredentialsError describes that that the request login credentials (e.g. an Oauth2 token)
// was invalid (i.e. a 401 Unauthorized or 403 Forbidden status was returned). This does NOT mean that
// "the login was successful but you don't have permission to access this resource". In that case, a
// 404 Not Found error would be returned.
type InvalidCredentialsError struct {
	// InvalidCredentialsError extends HTTPError.
	HTTPError `json:",inline"`
}
