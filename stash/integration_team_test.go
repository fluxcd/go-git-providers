/*
Copyright 2021 The Flux authors

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

package stash

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stash Provider", func() {
	var (
		ctx          context.Context = context.Background()
		testTeamName string          = "fluxcd-testing-2"
	)

	It("should list teams and members with access to an organization", func() {

		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the org
		teams, err := testOrg.Teams().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(teams)).To(Equal(1), "The 1 team wasn't there...")

		// Get a specific team
		team, err := testOrg.Teams().Get(ctx, testTeamName)
		Expect(err).ToNot(HaveOccurred())
		Expect(team.Get().Name).To(Equal(testTeamName))
		Expect(team.Get().Members).ToNot(BeNil())
	})
})
