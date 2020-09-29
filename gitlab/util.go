package gitlab

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/xanzy/go-gitlab"
)

const (
	alreadyExistsMagicString = "name: [has already been taken]"
	alreadySharedWithGroup   = "already shared with this group"
	masterBranchName         = "master"
)

func getRepoPath(ref gitprovider.RepositoryRef) string {
	return fmt.Sprintf("%s/%s", ref.GetIdentity(), ref.GetRepository())
}

// allPages runs fn for each page, expecting a HTTP request to be made and returned during that call.
// allPages expects that the data is saved in fn to an outer variable.
// allPages calls fn as many times as needed to get all pages, and modifies opts for each call.
// There is no need to wrap the resulting error in handleHTTPError(err), as that's already done.
func allGroupPages(opts *gitlab.ListGroupsOptions, fn func() (*gitlab.Response, error)) error {
	for {
		resp, err := fn()
		if err != nil {
			return handleHTTPError(err)
		}
		if resp.NextPage == 0 {
			return nil
		}
		opts.Page = resp.NextPage
	}
}

func allSubgroupPages(opts *gitlab.ListSubgroupsOptions, fn func() (*gitlab.Response, error)) error {
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

func allGroupProjectPages(opts *gitlab.ListGroupProjectsOptions, fn func() (*gitlab.Response, error)) error {
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

func allGroupMemberPages(opts *gitlab.ListGroupMembersOptions, fn func() (*gitlab.Response, error)) error {
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

func allProjectPages(opts *gitlab.ListProjectsOptions, fn func() (*gitlab.Response, error)) error {
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

func allProjectUserPages(opts *gitlab.ListProjectUserOptions, fn func() (*gitlab.Response, error)) error {
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

func allDeployKeyPages(opts *gitlab.ListProjectDeployKeysOptions, fn func() (*gitlab.Response, error)) error {
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

// validateUserRepositoryRef makes sure the UserRepositoryRef is valid for GitHub's usage.
func validateUserRepositoryRef(ref gitprovider.UserRepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("UserRepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateOrgRepositoryRef makes sure the OrgRepositoryRef is valid for GitHub's usage.
func validateOrgRepositoryRef(ref gitprovider.OrgRepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("OrgRepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateUserRef makes sure the UserRef is valid for GitHub's usage.
func validateUserRef(ref gitprovider.UserRef, expectedDomain string) error {
	// Make sure the OrganizationRef fields are valid
	if err := validation.ValidateTargets("UserRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateAPIObject creates a Validatior with the specified name, gives it to fn, and
// depending on if any error was registered with it; either returns nil, or a MultiError
// with both the validation error and ErrInvalidServerData, to mark that the server data
// was invalid.
func validateAPIObject(name string, fn func(validation.Validator)) error {
	v := validation.New(name)
	fn(v)
	// If there was a validation error, also mark it specifically as invalid server data
	if err := v.Error(); err != nil {
		return validation.NewMultiError(err, gitprovider.ErrInvalidServerData)
	}
	return nil
}

func validateProjectAPI(apiObj *gitlab.Project) error {
	return validateAPIObject("GitLab.Project", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
		// Make sure visibility is valid if set
		if apiObj.Visibility != "" {
			v := gitprovider.RepositoryVisibility(apiObj.Visibility)
			validator.Append(gitprovider.ValidateRepositoryVisibility(v), v, "Visibility")
		}
		// Set default branch to master if unset
		if apiObj.DefaultBranch == "" {
			apiObj.DefaultBranch = masterBranchName
		}
	})
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
		return fmt.Errorf("github doesn't support sub-organizations: %w", gitprovider.ErrNoProviderSupport)
	}
	return fmt.Errorf("invalid identity type: %v: %w", ref.GetType(), gitprovider.ErrInvalidArgument)
}

// handleHTTPError checks the type of err, and returns typed variants of it
// However, it _always_ keeps the original error too, and just wraps it in a MultiError
// The consumer must use errors.Is and errors.As to check for equality and get data out of it.
func handleHTTPError(err error) error {
	// Short-circuit quickly if possible, allow always piping through this function
	if err == nil {
		return nil
	}
	glErrorResponse := &gitlab.ErrorResponse{}
	if errors.As(err, &glErrorResponse) {
		httpErr := gitprovider.HTTPError{
			Response:     glErrorResponse.Response,
			ErrorMessage: glErrorResponse.Error(),
			Message:      glErrorResponse.Message,
		}
		// Check for invalid credentials, and return a typed error in that case
		if glErrorResponse.Response.StatusCode == http.StatusForbidden ||
			glErrorResponse.Response.StatusCode == http.StatusUnauthorized {
			return validation.NewMultiError(err,
				&gitprovider.InvalidCredentialsError{HTTPError: httpErr},
			)
		}
		// Check for 404 Not Found
		if glErrorResponse.Response.StatusCode == http.StatusNotFound {
			return validation.NewMultiError(err, gitprovider.ErrNotFound)
		}
		// Check for already exists errors
		if strings.Contains(glErrorResponse.Message, alreadyExistsMagicString) {
			return validation.NewMultiError(err, gitprovider.ErrAlreadyExists)
		}
		// Otherwise, return a generic *HTTPError
		return validation.NewMultiError(err, &httpErr)
	}
	// Do nothing, just pipe through the unknown err
	return err
}
