package stash

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

const (
	alreadyExistsMagicString = "name: [has already been taken]"
	alreadySharedWithGroup   = "already shared with this group"
)

var (
	masterBranchName = "master"
)

func getRepoPath(ref gitprovider.RepositoryRef) string {
	return fmt.Sprintf("%s/%s", ref.GetIdentity(), ref.GetRepository())
}

// allPages runs fn for each page, expecting a HTTP request to be made and returned during that call.
// allPages expects that the data is saved in fn to an outer variable.
// allPages calls fn as many times as needed to get all pages, and modifies opts for each call.
// There is no need to wrap the resulting error in handleHTTPError(err), as that's already done.
func allPages(opts *ListOptions, fn func() (*Paging, error)) error {
	for {
		resp, err := fn()
		if err != nil {
			return err
		}
		if resp.IsLastPage {
			return nil
		}
		opts.Start = resp.NextPageStart
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

func validateProjectAPI(apiObj *Project) error {
	return validateAPIObject("Stash.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

// validateOrganizationRef makes sure the OrganizationRef is valid for stash usage.
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
		return fmt.Errorf("stash doesn't support sub-organizations: %w", gitprovider.ErrNoProviderSupport)
	}
	return fmt.Errorf("invalid identity type: %v: %w", ref.GetType(), gitprovider.ErrInvalidArgument)
}

// addPaging adds paging elements to URI query
func addPaging(query *url.Values, opts *ListOptions) *url.Values {
	if opts == nil {
		return query
	}

	if query == nil {
		query = &url.Values{}
	}

	if opts.Limit != 0 {
		query.Add("limit", strconv.Itoa(int(opts.Limit)))
	}

	if opts.Start != 0 {
		query.Add("start", strconv.Itoa(int(opts.Start)))
	}
	return query
}

// setKeyValue adds or replaces a element in URI query
func setKeyValues(query *url.Values, key string, value string) *url.Values {
	if query == nil {
		query = &url.Values{}
	}

	query.Set(key, value)
	return query
}

// newURI builds stash URI
func newURI(elements ...string) string {
	return strings.Join(append([]string{stashURIprefix}, elements...), "/")
}

// newKeysURI builds stash keys URI
func newKeysURI(elements ...string) string {
	return strings.Join(append([]string{stashURIkeys}, elements...), "/")
}

// downloadFile will download a url to a local file.
func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

var licenseURLs = map[gitprovider.LicenseTemplate]string{
	gitprovider.LicenseTemplate("apache-2.0"): "https://www.apache.org/licenses/LICENSE-2.0.txt",
	gitprovider.LicenseTemplate("gpl-3.0"):    "https://www.gnu.org/licenses/gpl-3.0-standalone.html",
}

func getLicense(repoDir string, license gitprovider.LicenseTemplate) error {

	licenseURL, ok := licenseURLs[license]
	if !ok {
		return errors.New(fmt.Sprintf("license: %s, not supported", license))
	}
	return downloadFile(fmt.Sprintf("%s/LICENSE.md", repoDir), licenseURL)
}

func getSelfref(selves []Self) string {
	if len(selves) == 0 {
		return "no http ref found"
	}
	return selves[0].Href
}
