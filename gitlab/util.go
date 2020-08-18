package gitlab

import (
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/xanzy/go-gitlab"
	"github.com/xanzy/go-gitlab/groups"
)

// allPages runs fn for each page, expecting a HTTP request to be made and returned during that call.
// allPages expects that the data is saved in fn to an outer variable.
// allPages calls fn as many times as needed to get all pages, and modifies opts for each call.
// There is no need to wrap the resulting error in handleHTTPError(err), as that's already done.
func allPages(opts *groups.ListGroupsOptions, fn func() (*gitlab.Response, error)) error {
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
