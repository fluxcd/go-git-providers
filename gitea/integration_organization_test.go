/*
Copyright 2020 The Flux CD contributors.

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

package gitea

import (
	"context"
	"errors"

	"code.gitea.io/sdk/gitea"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

var _ = Describe("Gitea Provider", func() {
	var (
		ctx context.Context = context.Background()
	)

	It("should list the available organizations the user has access to", func() {
		// Get a list of all organizations the user is part of
		orgs, err := c.Organizations().List(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Make sure we find the expected one given as testOrgName
		var listedOrg, getOrg gitprovider.Organization
		for _, org := range orgs {
			if org.Organization().Organization == testOrgName {
				listedOrg = org
				break
			}
		}
		Expect(listedOrg).ToNot(BeNil())

		hits := customTransportImpl.countCacheHitsForFunc(func() {
			// Do a GET call for that organization
			getOrg, err = c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
		})
		// don't expect any cache hit, as we didn't request this before
		Expect(hits).To(Equal(0))

		// Expect that the organization's info is the same regardless of method
		Expect(getOrg.Organization()).To(Equal(listedOrg.Organization()))
		// We expect the name from LIST calls, but we do expect
		// the description, see: https://gitea.com/api/swagger#/organization/orgGetAll
		Expect(*listedOrg.Get().Name).To(Equal(testOrgName))
		Expect(listedOrg.Get().Description).ToNot(BeNil())
		// We expect the name and description to be populated
		// in the GET call. Note: This requires the user to set up
		// the given organization with a name and description in the UI :)
		// https://gitea.com/api/swagger#/organization/orgGet
		Expect(getOrg.Get().Name).ToNot(BeNil())
		Expect(getOrg.Get().Description).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.APIObject().(*gitea.Organization)
		Expect(*getOrg.Get().Name).To(Equal(internal.UserName))
		Expect(*getOrg.Get().Description).To(Equal(internal.Description))

		// Expect gitea not to hit cache even when we do the same request a second time
		hits = customTransportImpl.countCacheHitsForFunc(func() {
			getOrg2, err := c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
			Expect(getOrg2).ToNot(BeNil())
		})
		Expect(hits).To(Equal(0))
	})

	It("should fail when .Children is called", func() {
		// Expect .Children to return gitprovider.ErrProviderNoSupport
		_, err := c.Organizations().Children(ctx, gitprovider.OrganizationRef{})
		Expect(errors.Is(err, gitprovider.ErrNoProviderSupport)).To(BeTrue())
	})
})
