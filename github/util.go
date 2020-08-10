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

package github

import (
	"errors"
	"fmt"
	"net/http"

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/google/go-github/v32/github"
)

const (
	alreadyExistsMagicString = "name already exists on this account"
	rateLimitDocURL          = "https://developer.github.com/v3/#rate-limiting"
)

// TODO: Guard better against nil pointer dereference panics in this package, also
// validate data coming from the server

// validateRepositoryRef makes sure the RepositoryRef is valid for Github's usage.
func validateRepositoryRef(ref gitprovider.RepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("RepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateOrganizationRef makes sure the OrganizationRef is valid for Github's usage.
func validateOrganizationRef(ref gitprovider.OrganizationRef, expectedDomain string) error {
	// Make sure the OrganizationRef fields are valid
	if err := validation.ValidateTargets("OrganizationRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateIdentityRef makes sure the IdentityRef is valid for Github's usage.
func validateIdentityRef(ref gitprovider.IdentityRef, expectedDomain string) error {
	// Make sure the ref is valid
	if err := validation.ValidateTargets("IdentityRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateIdentityFields makes sure the type of the IdentityRef is supported, and the domain is as expected
func validateIdentityFields(ref gitprovider.IdentityRef, expectedDomain string) error {
	// Make sure the expected domain is used
	if ref.GetDomain() != expectedDomain {
		return gitprovider.ErrDomainUnsupported
	}
	// Make sure the right type of identityref is used
	switch ref.GetType() {
	case gitprovider.IdentityTypeOrganization, gitprovider.IdentityTypeUser:
		return nil
	case gitprovider.IdentityTypeSuborganization:
		return gitprovider.ErrProviderNoSupport
	}
	return gitprovider.ErrInvalidArgument
}

// resolveOrg returns the organization name if ref is an organization, or
// an empty string if it is an user account.
func resolveOrg(ref gitprovider.IdentityRef) string {
	// If the ref is an organization, return its name
	if ref.GetType() == gitprovider.IdentityTypeOrganization {
		return ref.GetIdentity()
	}
	// If the ref is an user account, return an empty string
	return ""
}

func repositoryFromAPI(apiObj *github.Repository, ref gitprovider.RepositoryRef) *gitprovider.Repository {
	repo := &gitprovider.Repository{
		RepositoryInfo: gitprovider.FromRepositoryRef(ref),
		InternalHolder: gitprovider.WithInternal(apiObj),
	}
	if apiObj.Description != nil && len(*apiObj.Description) != 0 {
		repo.Description = apiObj.Description
	}
	if apiObj.DefaultBranch != nil && len(*apiObj.DefaultBranch) != 0 {
		repo.DefaultBranch = apiObj.DefaultBranch
	}
	if apiObj.Visibility != nil && len(*apiObj.Visibility) != 0 {
		// TODO: What should we do if *apiObj.Visibility wouldn't be any of the already-known values?
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility(*apiObj.Visibility))
	}
	return repo
}

func repositoryToAPI(repo *gitprovider.Repository) *github.Repository {
	apiObj := &github.Repository{
		Name:          &repo.RepositoryName,
		Description:   repo.Description,
		DefaultBranch: repo.DefaultBranch,
	}
	if repo.Visibility != nil {
		apiObj.Visibility = gitprovider.StringVar(string(*repo.Visibility))
	}
	return apiObj
}

func applyRepoCreateOptions(apiObj *github.Repository, opts gitprovider.RepositoryCreateOptions) {
	apiObj.AutoInit = opts.AutoInit
	if opts.LicenseTemplate != nil {
		apiObj.LicenseTemplate = gitprovider.StringVar(string(*opts.LicenseTemplate))
	}
}

func organizationFromAPI(apiObj *github.Organization, domain string) *gitprovider.Organization {
	return &gitprovider.Organization{
		OrganizationInfo: gitprovider.OrganizationInfo{
			Domain:       domain,
			Organization: *apiObj.Login,
		},
		InternalHolder: gitprovider.WithInternal(apiObj),
		Name:           apiObj.Name,
		Description:    apiObj.Description,
	}
}

// handleHTTPError checks the type of err, and returns typed variants of it
// However, it _always_ keeps the original error too, and just wraps it in a MultiError
// The consumer must use errors.Is and errors.As to check for equality and get data out of it
func handleHTTPError(err error) error {
	ghRateLimitError := &github.RateLimitError{}
	ghErrorResponse := &github.ErrorResponse{}
	if errors.As(err, &ghRateLimitError) {
		// Convert go-github's RateLimitError to our similar error type
		return validation.NewMultiError(err, &gitprovider.RateLimitError{
			HTTPError: gitprovider.HTTPError{
				Response:         ghRateLimitError.Response,
				ErrorMessage:     ghRateLimitError.Error(),
				Message:          ghRateLimitError.Message,
				DocumentationURL: rateLimitDocURL,
			},
			Limit:     ghRateLimitError.Rate.Limit,
			Remaining: ghRateLimitError.Rate.Remaining,
			Reset:     ghRateLimitError.Rate.Reset.Time,
		})
	} else if errors.As(err, &ghErrorResponse) {
		httpErr := gitprovider.HTTPError{
			Response:         ghErrorResponse.Response,
			ErrorMessage:     ghErrorResponse.Error(),
			Message:          ghErrorResponse.Message,
			DocumentationURL: ghErrorResponse.DocumentationURL,
		}
		// Check for invalid credentials, and return a typed error in that case
		if ghErrorResponse.Response.StatusCode == http.StatusForbidden ||
			ghErrorResponse.Response.StatusCode == http.StatusUnauthorized {
			return validation.NewMultiError(err,
				&gitprovider.InvalidCredentialsError{HTTPError: httpErr},
			)
		}
		// Check for 404 Not Found
		if ghErrorResponse.Response.StatusCode == http.StatusNotFound {
			return validation.NewMultiError(err, gitprovider.ErrNotFound)
		}
		// Check for already exists errors
		for _, validationErr := range ghErrorResponse.Errors {
			if validationErr.Message == alreadyExistsMagicString {
				return validation.NewMultiError(err, gitprovider.ErrAlreadyExists)
			}
		}
		// Otherwise, return a generic *HTTPError
		return validation.NewMultiError(err, &httpErr)
	}
	// Do nothing, just pipe through the unknown err
	return err
}

// allPages runs fn for each page, expecting a HTTP request to be made and returned during that call.
// allPages expects that the data is saved in fn to an outer variable.
// allPages calls fn as many times as needed to get all pages, and modifies opts for each call.
func allPages(opts *github.ListOptions, fn func() (*github.Response, error)) error {
	for {
		resp, err := fn()
		if err != nil {
			return err
		}
		if resp.NextPage == 0 {
			return nil
		}
		opts.Page = resp.NextPage
	}
}

func validateRepoInfo(desired gitprovider.RepositoryInfo, givenRef *gitprovider.RepositoryInfo) error {
	// Make sure the given reference matches c.info, if set. If givenRef isn't set, just return
	if givenRef == nil {
		return nil
	}
	// Make sure the request matches the given validated info
	if ok, _ := gitprovider.Equals(desired, givenRef); ok {
		return nil
	}
	return fmt.Errorf("repository information in req doesn't match the client: %w", gitprovider.ErrInvalidArgument)
}
