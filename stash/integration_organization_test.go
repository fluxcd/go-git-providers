//go:build e2e

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

	"github.com/fluxcd/go-git-providers/gitprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stash Provider", func() {
	var (
		ctx context.Context = context.Background()
	)
	It("should list the available organizations the user has access to", func() {
		// Get a list of all organizations the user is part of
		orgs, err := client.Organizations().List(ctx)
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
			getOrg, err = client.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
		})
		// don't expect any cache hit, as we didn't request this before
		Expect(hits).To(Equal(0))

		// Expect that the organization's info is the same regardless of method
		Expect(getOrg.Organization()).To(Equal(listedOrg.Organization()))

		Expect(listedOrg.Get().Name).ToNot(BeNil())
		Expect(listedOrg.Get().Description).ToNot(BeNil())
		Expect(listedOrg.Organization().Key()).ToNot(BeNil())

		// We expect the name and description to be populated
		Expect(getOrg.Get().Name).ToNot(BeNil())
		Expect(getOrg.Get().Description).ToNot(BeNil())
		Expect(getOrg.Organization().Key()).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.APIObject().(*Project)
		derefOrgName := *getOrg.Get().Name
		Expect(derefOrgName).To(Equal(internal.Name))
		// Expect that when we do the same request a second time, it will hit the cache
		// Unfortunately our stash server doesn't support caching, so we can't test this
		// It sets header response header to "Cache-Control:[private, no-cache no-cache, no-transform]""
		hits = customTransportImpl.countCacheHitsForFunc(func() {
			getOrg2, err := client.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
			Expect(getOrg2).ToNot(BeNil())
		})
		Expect(hits).To(Equal(0))

	})

	It("should fail when .Children is called", func() {
		_, err := client.Organizations().Children(ctx, gitprovider.OrganizationRef{
			Domain:       stashDomain,
			Organization: testOrgName,
		})
		Expect(err).To(Equal(gitprovider.ErrNoProviderSupport))
	})

})
