package gitlab

import (
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/xanzy/go-gitlab"
)

// validateProjectAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateProjectAPI(apiObj *gitlab.Project) error {
	return validateAPIObject("GitLab.Project", func(validator validation.Validator) {
	})
}
