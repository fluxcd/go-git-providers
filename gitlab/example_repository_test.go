//go:build e2e

package gitlab_test

import (
	"context"
	"fmt"
	"os"

	gogitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

func ExampleOrgRepositoriesClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := gitlab.NewClient(os.Getenv("GITLAB_ACCESS_TOKEN"), "")
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL("https://gitlab.com/gitlab-org/gitlab-foss")
	checkErr(err)

	// Get public information about the flux repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogitlab.Project to access custom data
	internalRepo := repo.APIObject().(*gogitlab.Project)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.HTTPURLToRepo)
}
