package stash

import (
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	gostash "github.com/fluxcd/go-git-providers/go-stash"
	"github.com/fluxcd/go-git-providers/validation"
)

var (
	mainBranchName = "main"
)

func addTilde(userName string) string {
	if len(userName) > 0 && userName[0] == '~' {
		return userName
	}
	return fmt.Sprintf("~%s", userName)
}

// allPages runs fn for each page, expecting a HTTP request to be made and returned during that call.
// allPages expects that the data is saved in fn to an outer variable.
// allPages calls fn as many times as needed to get all pages, and modifies opts for each call.
// There is no need to wrap the resulting error in handleHTTPError(err), as that's already done.
func allPages(opts *gostash.PagingOptions, fn func() (*gostash.Paging, error)) error {
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

func validateUserAPI(apiObj *gostash.User) error {
	return validateAPIObject("Stash.User", func(validator validation.Validator) {
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

func validateProjectAPI(apiObj *Project) error {
	return validateAPIObject("Stash.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
	})
}

func validateProjectGroupPermissionAPI(apiObj *ProjectGroupPermission) error {
	return validateAPIObject("Stash.ProjectGroupPermission", func(validator validation.Validator) {
		if apiObj.Group.Name == "" {
			validator.Required("Name")
		}
	})
}
