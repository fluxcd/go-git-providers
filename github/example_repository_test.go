package github_test

import (
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitprovider"
	gogithub "github.com/google/go-github/v32/github"
)

func ExampleOrgRepositoriesClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := github.NewClient()
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL("https://github.com/fluxcd/flux")
	checkErr(err)

	// Get public information about the flux repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogithub.Repository to access custom data
	internalRepo := repo.APIObject().(*gogithub.Repository)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.GetHomepage())
	// Output: Description: The GitOps Kubernetes operator. Homepage: https://fluxcd.io/docs
}
