package stash_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fluxcd/go-git-providers/stash"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// checkErr is used for examples in this repository.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func TestExampleOrganizationsClient_Get(t *testing.T) {
	stashDomain := "dummy"
	if stashDomainVar := os.Getenv("STASH_DOMAIN"); len(stashDomainVar) != 0 {
		stashDomain = stashDomainVar
	}

	testOrgName := "fluxcd-testing-public"
	if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
		testOrgName = orgName
	}

	// Create a new client
	ctx := context.Background()
	c, err := stash.NewClient(os.Getenv("STASH_TOKEN"), "", stash.WithDomain(stashDomain))
	checkErr(err)

	// Get public information about the fluxcd organization
	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       stashDomain,
		Organization: testOrgName,
	})
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	orgInfo := org.Get()
	// Cast the internal object to a *gogithub.Organization to access custom data
	internalOrg := org.APIObject().(*stash.Project)

	fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.Links.Self)
}
