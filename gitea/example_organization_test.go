package gitea_test

import (
	"context"
	"fmt"
	"log"

	gogitea "code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
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
	c, err := gitea.NewClient(gitprovider.WithDomain(gitea.DefaultDomain))
	checkErr(err)

	// Get public information about the gitea organization

	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       gitea.DefaultDomain,
		Organization: "gitea",
	})
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	orgInfo := org.Get()
	// Cast the internal object to a *gogitea.Organization to access custom data
	internalOrg := org.APIObject().(*gogitea.Organization)

	fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.Location)
	// Output: Name: gitea. Location: Git Earth.
}
