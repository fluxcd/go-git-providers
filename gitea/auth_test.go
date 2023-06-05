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
			name: "try.gitea.io domain",
			opts: gitprovider.WithDomain("try.gitea.io"),
			want: "try.gitea.io",
		},
		{
			name: "custom domain without protocol",
			opts: gitprovider.WithDomain("try.gitea.io"),
			want: "try.gitea.io",
		},
		{
			name: "custom domain with https protocol",
			opts: gitprovider.WithDomain("https://try.gitea.io"),
			want: "https://try.gitea.io",
		},
		{
			name: "custom domain with http protocol",
			opts: gitprovider.WithDomain("http://try.gitea.io"),
			want: "http://try.gitea.io",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1, err := NewClient("token", tt.opts)
			if err != nil {
				t.Fatal(err)
			}
			assertEqual(t, tt.want, c1.SupportedDomain())

			c2, _ := NewClient("token", tt.opts)
			assertEqual(t, tt.want, c2.SupportedDomain())
		})
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
