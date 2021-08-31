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

package github

import (
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
			name: "github.com domain",
			opts: gitprovider.WithDomain("github.com"),
			want: "github.com",
		},
		{
			name: "custom domain without protocol",
			opts: gitprovider.WithDomain("my-github.dev.com"),
			want: "my-github.dev.com",
		},
		{
			name: "custom domain with https protocol",
			opts: gitprovider.WithDomain("https://my-github.dev.com"),
			want: "https://my-github.dev.com",
		},
		{
			name: "custom domain with http protocol",
			opts: gitprovider.WithDomain("http://my-github.dev.com"),
			want: "http://my-github.dev.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1, _ := NewClient(tt.opts)
			assertEqual(t, tt.want, c1.SupportedDomain())

			c2, _ := NewClient(tt.opts)
			assertEqual(t, tt.want, c2.SupportedDomain())
		})
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
