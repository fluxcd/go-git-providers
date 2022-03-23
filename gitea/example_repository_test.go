package gitea_test

import (
	"context"
	"fmt"

	gogitea "code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

func ExampleOrgRepositoriesClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := gitea.NewClient(gitprovider.WithDomain(gitea.DefaultDomain))
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL("https://gitea.com/gitea/go-sdk")
	checkErr(err)

	// Get public information about the flux repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogitea.Repository to access custom data
	internalRepo := repo.APIObject().(*gogitea.Repository)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.HTMLURL)
	// Output: Description: Gitea: Golang SDK. Homepage: https://gitea.com/gitea/go-sdk
}
