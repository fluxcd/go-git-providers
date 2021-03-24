package gitlab_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
	gogitlab "github.com/xanzy/go-gitlab"
)

// checkErr is used for examples in this repository.
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func TestExampleOrganizationsClient_Get(t *testing.T) {
	// Create a new client
	ctx := context.Background()
	c, err := gitlab.NewClient(os.Getenv("GITLAB_TOKEN"), "")
	checkErr(err)

	// Get public information about the fluxcd organization
	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       gitlab.DefaultDomain,
		Organization: "fluxcd-testing-public",
	})
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	orgInfo := org.Get()
	// Cast the internal object to a *gogithub.Organization to access custom data
	internalOrg := org.APIObject().(*gogitlab.Group)

	fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.Path)
	// Output: Name: Flux project. Location: CNCF incubation.
}
