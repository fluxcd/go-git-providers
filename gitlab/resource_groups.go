package gitlab

import (
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/xanzy/go-gitlab"
)

// validateGroupAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateGroupAPI(apiObj *gitlab.Group) error {
	return validateAPIObject("GitLab.Group", func(validator validation.Validator) {
	})
}
