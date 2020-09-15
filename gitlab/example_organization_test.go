package gitlab_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
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
	c, err := gitlab.NewClientFromPAT(os.Getenv("GITLAB_TOKEN"))
	fmt.Println("Client: ", c)
	checkErr(err)

	// Get public information about the fluxcd organization
	org, err := c.Organizations().Get(ctx, gitprovider.OrganizationRef{
		Domain:       gitlab.DefaultDomain,
		Organization: "GGPGroup",
	})
	checkErr(err)
	fmt.Println("org:", org)

	// // Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	// orgInfo := org.Get()
	// // Cast the internal object to a *gogithub.Organization to access custom data
	// internalOrg := org.APIObject().(*gogitlab.Group)

	// fmt.Printf("Name: %s. Location: %s.", *orgInfo.Name, internalOrg.Path)
	// // Output: Name: Flux project. Location: CNCF sandbox.
}
