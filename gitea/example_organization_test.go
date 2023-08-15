/*
Copyright 2023 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gitea_test

import (
	"context"
	"fmt"
	"log"
	"os"

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
	c, err := gitea.NewClient(os.Getenv("GITEA_ACCESS_TOKEN"), gitprovider.WithDomain(gitea.DefaultDomain))
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
	// Output: Name: gitea. Location: Git Universe.
}
