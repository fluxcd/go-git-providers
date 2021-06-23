package stash_test

import (
	"context"
	"fmt"
	"os"

	"github.com/fluxcd/go-git-providers/stash"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func ExampleOrgRepositoriesClient_Get() {
	stashDomain := stash.DefaultDomain
	if stashDomainVar := os.Getenv("STASH_DOMAIN"); len(stashDomainVar) != 0 {
		stashDomain = stashDomainVar
	}

	testOrgName := "fluxcd-testing-public"
	if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
		testOrgName = orgName
	}

	testRepoName := "test2"
	if repoName := os.Getenv("STASH_TEST_REPO_NAME"); len(repoName) != 0 {
		testRepoName = repoName
	}

	// Create a new client
	ctx := context.Background()
	c, err := stash.NewClient(os.Getenv("STASH_TOKEN"), "")
	checkErr(err)

	orgRef := gitprovider.OrganizationRef{
		Domain:       stashDomain,
		Organization: testOrgName,
	}
	targetRepo, err := c.Organizations().Get(ctx, orgRef)
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL(fmt.Sprintf("https://%s/scm/%s/%s.git", stashDomain, targetRepo.APIObject().(stash.Project).Key, testRepoName))
	checkErr(err)

	// Get public information about the repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogithub.Repository to access custom data
	internalRepo := repo.APIObject().(*stash.Repository)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.Links.Self)
}
