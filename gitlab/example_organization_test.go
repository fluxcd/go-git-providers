//go:build e2e

package gitlab_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
	gogitlab "gitlab.com/gitlab-org/api/client-go"
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
	c, err := gitlab.NewClient(os.Getenv("GITLAB_ACCESS_TOKEN"), "")
	checkErr(err)

	// Get public information about the fluxcd organization
	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       gitlab.DefaultDomain,
		Organization: "fluxcd-testing-public",
	})
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	orgInfo := org.Get()
	// Cast the internal object to a *gogitlab.Group to access custom data
	internalOrg := org.APIObject().(*gogitlab.Group)

	fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.Path)
	// Output: Name: fluxcd-testing-public. Location: fluxcd-testing-public.
}
