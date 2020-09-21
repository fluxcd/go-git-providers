package gitlab_test

import (
	"context"
	"fmt"
	"os"

	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
	gogitlab "github.com/xanzy/go-gitlab"
)

func ExampleOrgRepositoriesClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), "")
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL("https://gitlab.com/gitlab-org/gitlab-foss")
	checkErr(err)

	// Get public information about the flux repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogithub.Repository to access custom data
	internalRepo := repo.APIObject().(*gogitlab.Project)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.HTTPURLToRepo)
	// Output: Description: GitLab FOSS is a read-only mirror of GitLab, with all proprietary code removed. This project was previously used to host GitLab Community Edition, but all development has now moved to https://gitlab.com/gitlab-org/gitlab.. Homepage: https://gitlab.com/gitlab-org/gitlab-foss.git
}
