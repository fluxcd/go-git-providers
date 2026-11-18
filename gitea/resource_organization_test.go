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

package gitea

import (
	"testing"

	"gitea.dev/sdk"
)

func Test_organizationName(t *testing.T) {
	tests := []struct {
		name   string
		apiObj *gitea.Organization
		want   string
	}{
		{
			name:   "name is set",
			apiObj: &gitea.Organization{Name: "fluxcd", UserName: "fluxcd"},
			want:   "fluxcd",
		},
		{
			name:   "fall back to username on Gitea < v1.18",
			apiObj: &gitea.Organization{UserName: "fluxcd"},
			want:   "fluxcd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := organizationName(tt.apiObj); got != tt.want {
				t.Errorf("organizationName() = %q, want %q", got, tt.want)
			}
		})
	}
}
