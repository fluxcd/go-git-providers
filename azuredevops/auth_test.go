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
package azuredevops

import (
	"github.com/google/go-cmp/cmp"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_DomainVariations(t *testing.T) {
	tests := []struct {
		name         string
		opts         gitprovider.ClientOption
		want         string
		expectedErrs []error
	}{
		{
			name: "custom domain without protocol",
			opts: gitprovider.WithDomain("dev.azure.com/test-demo"),
			want: "https://dev.azure.com/test-demo",
		},
		{
			name: "custom domain with https protocol",
			opts: gitprovider.WithDomain("https://dev.azure.com/test-demo"),
			want: "https://dev.azure.com/test-demo",
		},
		{
			name: "custom domain with http protocol",
			opts: gitprovider.WithDomain("http://dev.azure.com/test-demo"),
			want: "https://dev.azure.com/test-demo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c2, _ := NewClient("personalAccessToken", tt.opts)
			if diff := cmp.Diff(tt.want, c2.SupportedDomain()); diff != "" {
				t.Errorf("New Stash client returned domain (want -> got): %s", diff)
			}
		})
	}
}
