package github_test

import (
	"context"
	"fmt"
	"log"

	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitprovider"
	gogithub "github.com/google/go-github/v32/github"
)

// checkErr is used for examples in this repository.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleOrganizationsClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := github.NewClient()
	checkErr(err)

	// Get public information about the fluxcd organization
	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       github.DefaultDomain,
		Organization: "fluxcd",
	})
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	orgInfo := org.Get()
	// Cast the internal object to a *gogithub.Organization to access custom data
	internalOrg := org.APIObject().(*gogithub.Organization)

	fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.GetLocation())
	// Output: Name: Flux project. Location: CNCF incubation.
}
