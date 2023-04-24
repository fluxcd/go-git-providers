/*
Copyright 2023 The Flux CD contributors.

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

package azuredevops

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
)

const (
	alreadyExistsMagicString = "TF400948: A Git repository with the name"
)

// validateIdentityFields makes sure the type of the IdentityRef is supported, and the domain is as expected.
func validateIdentityFields(ref gitprovider.IdentityRef, expectedDomain string) error {
	// Make sure the expected domain is used
	if ref.GetDomain() != expectedDomain {
		return fmt.Errorf("domain %q not supported by this client: %w", ref.GetDomain(), gitprovider.ErrDomainUnsupported)
	}
	// Make sure the right type of identityref is used
	switch ref.GetType() {
	case gitprovider.IdentityTypeOrganization, gitprovider.IdentityTypeUser:
		return nil
	case gitprovider.IdentityTypeSuborganization:
		return fmt.Errorf("gitlab doesn't support sub-organizations: %w", gitprovider.ErrNoProviderSupport)
	}
	return fmt.Errorf("invalid identity type: %v: %w", ref.GetType(), gitprovider.ErrInvalidArgument)
}

// validateOrganizationRef makes sure the OrganizationRef is valid for GitHub's usage.
func validateOrganizationRef(ref gitprovider.OrganizationRef, expectedDomain string) error {
	// Make sure the OrganizationRef fields are valid
	if err := validation.ValidateTargets("OrganizationRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateOrgRepositoryRef makes sure the OrgRepositoryRef is valid for GitHub's usage.
func validateRepositoryRef(ref gitprovider.OrgRepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("OrgRepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// handleHTTPError checks the type of err, and returns typed variants of it
// However, it _always_ keeps the original error too, and just wraps it in a MultiError
// The consumer must use errors.Is and errors.As to check for equality and get data out of it.
func handleHTTPError(err error) error {
	// Short-circuit quickly if possible, allow always piping through this function
	if err == nil {
		return nil
	}
	azureDevopsErrorResponse := azuredevops.WrappedError{}
	if errors.As(err, &azureDevopsErrorResponse) {
		httpErr := gitprovider.HTTPError{
			Response: &http.Response{
				StatusCode: *azureDevopsErrorResponse.StatusCode,
				Status:     *azureDevopsErrorResponse.TypeKey,
			},
			ErrorMessage: azureDevopsErrorResponse.Error(),
			Message:      *azureDevopsErrorResponse.Message,
		}
		// Check for invalid credentials, and return a typed error in that case
		if *azureDevopsErrorResponse.StatusCode == http.StatusForbidden ||
			*azureDevopsErrorResponse.StatusCode == http.StatusUnauthorized {
			return validation.NewMultiError(err,
				&gitprovider.InvalidCredentialsError{HTTPError: httpErr},
			)
		}
		// Check for 404 Not Found
		if *azureDevopsErrorResponse.StatusCode == http.StatusNotFound {
			return validation.NewMultiError(err, gitprovider.ErrNotFound)
		}
		// Check for already exists errors
		if strings.Contains(*azureDevopsErrorResponse.Message, alreadyExistsMagicString) {
			return validation.NewMultiError(err, gitprovider.ErrAlreadyExists)
		}
		// Otherwise, return a generic *HTTPError
		return validation.NewMultiError(err, &httpErr)
	}
	// Do nothing, just pipe through the unknown err
	return err
}
