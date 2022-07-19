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

package gitprovider

import (
	"reflect"
	"time"

	"github.com/fluxcd/go-git-providers/validation"
)

const (
	// the default repository visibility is private.
	defaultRepositoryVisibility = RepositoryVisibilityPrivate
	// the default repository permission is "pull" (or read).
	defaultRepoPermission = RepositoryPermissionPull
	// the default branch name.
	// TODO: When enough Git providers support setting this at both POST and PATCH-time
	// (including when auto-initing), change this to "main".
	defaultBranchName = "main"
	// by default, deploy keys are read-only.
	defaultDeployKeyReadOnly = true
)

// RepositoryInfo implements InfoRequest and DefaultedInfoRequest (with a pointer receiver).
var _ InfoRequest = RepositoryInfo{}
var _ DefaultedInfoRequest = &RepositoryInfo{}

// RepositoryInfo represents a Git repository provided by a Git provider.
type RepositoryInfo struct {
	// Description returns a description for the repository.
	// No default value at POST-time.
	// +optional
	Description *string `json:"description"`

	// DefaultBranch describes the default branch for the given repository. This has
	// historically been "master" (and is as of writing still the Git default), but is
	// expected to be changed to e.g. "main" shortly in the future.
	// Default value at POST-time: master (but this can and will change in future library versions!).
	// +optional
	DefaultBranch *string `json:"defaultBranch"`

	// Visibility returns the desired visibility for the repository.
	// Default value at POST-time: RepositoryVisibilityPrivate.
	// +optional
	Visibility *RepositoryVisibility `json:"visibility"`
}

// Default defaults the Repository, implementing the InfoRequest interface.
func (r *RepositoryInfo) Default() {
	if r.Visibility == nil {
		r.Visibility = RepositoryVisibilityVar(defaultRepositoryVisibility)
	}
	if r.DefaultBranch == nil {
		r.DefaultBranch = StringVar(defaultBranchName)
	}
}

// ValidateInfo validates the object at {Object}.Set() and POST-time.
func (r RepositoryInfo) ValidateInfo() error {
	validator := validation.New("Repository")
	// Validate the Visibility enum
	if r.Visibility != nil {
		validator.Append(ValidateRepositoryVisibility(*r.Visibility), *r.Visibility, "Visibility")
	}
	return validator.Error()
}

// Equals can be used to check if this *Info request (the desired state) matches the actual
// passed in as the argument.
func (r RepositoryInfo) Equals(actual InfoRequest) bool {
	return reflect.DeepEqual(r, actual)
}

// TeamAccessInfo implements InfoRequest and DefaultedInfoRequest (with a pointer receiver).
var _ InfoRequest = TeamAccessInfo{}
var _ DefaultedInfoRequest = &TeamAccessInfo{}

// TeamAccessInfo contains high-level information about a team's access to a repository.
type TeamAccessInfo struct {
	// Name describes the name of the team. The team name may contain slashes.
	// +required
	Name string `json:"name"`

	// Permission describes the permission level for which the team is allowed to operate.
	// Default: pull.
	// Available options: See the RepositoryPermission enum.
	// +optional
	Permission *RepositoryPermission `json:"permission,omitempty"`
}

// Default defaults the TeamAccess fields.
func (ta *TeamAccessInfo) Default() {
	if ta.Permission == nil {
		ta.Permission = RepositoryPermissionVar(defaultRepoPermission)
	}
}

// ValidateInfo validates the object at {Object}.Set() and POST-time.
func (ta TeamAccessInfo) ValidateInfo() error {
	validator := validation.New("TeamAccess")
	// Make sure we've set the name of the team
	if len(ta.Name) == 0 {
		validator.Required("Name")
	}
	// Validate the Permission enum
	if ta.Permission != nil {
		validator.Append(ValidateRepositoryPermission(*ta.Permission), *ta.Permission, "Permission")
	}
	return validator.Error()
}

// Equals can be used to check if this *Info request (the desired state) matches the actual
// passed in as the argument.
func (ta TeamAccessInfo) Equals(actual InfoRequest) bool {
	return reflect.DeepEqual(ta, actual)
}

// DeployKeyInfo implements InfoRequest and DefaultedInfoRequest (with a pointer receiver).
var _ InfoRequest = DeployKeyInfo{}
var _ DefaultedInfoRequest = &DeployKeyInfo{}

// DeployKeyInfo contains high-level information about a deploy key.
type DeployKeyInfo struct {
	// Name is the human-friendly interpretation of what the key is for (and does).
	// +required
	Name string `json:"name"`

	// Key specifies the public part of the deploy (e.g. SSH) key.
	// +required
	Key []byte `json:"key"`

	// ReadOnly specifies whether this DeployKey can write to the repository or not.
	// Default value at POST-time: true.
	// +optional
	ReadOnly *bool `json:"readOnly,omitempty"`
}

// Default defaults the DeployKey fields.
func (dk *DeployKeyInfo) Default() {
	if dk.ReadOnly == nil {
		dk.ReadOnly = BoolVar(defaultDeployKeyReadOnly)
	}
}

// ValidateInfo validates the object at {Object}.Set() and POST-time.
func (dk DeployKeyInfo) ValidateInfo() error {
	validator := validation.New("DeployKey")
	// Make sure we've set the name of the deploy key
	if len(dk.Name) == 0 {
		validator.Required("Name")
	}
	// Key is a required field
	if len(dk.Key) == 0 {
		validator.Required("Key")
	}
	// Don't care about the RepositoryRef, as that information is coming from
	// the RepositoryClient. In the client, we make sure that they equal.
	return validator.Error()
}

// Equals can be used to check if this *Info request (the desired state) matches the actual
// passed in as the argument.
func (dk DeployKeyInfo) Equals(actual InfoRequest) bool {
	return reflect.DeepEqual(dk, actual)
}

// CommitInfo contains high-level information about a deploy key.
type CommitInfo struct {
	// Sha is the git sha for this commit.
	// +required
	Sha string `json:"sha"`

	// TreeSha is the tree sha this commit belongs to.
	// +required
	TreeSha string `json:"tree_sha"`

	// Author is the author of the commit
	Author string `json:"author"`

	// Message is the commit message
	Message string `json:"message"`

	// CreatedAt is the time the commit was created
	CreatedAt time.Time `json:"created_at"`

	// URL is the link for the commit
	URL string `json:"url"`
}

// CommitFile contains high-level information about a file added to a commit.
type CommitFile struct {
	// Path is path where this file is located.
	// +required
	Path *string `json:"path"`

	// Content is the content of the file.
	// +required
	Content *string `json:"content"`
}

// PullRequestInfo contains high-level information about a pull request.
type PullRequestInfo struct {
	// Merged specifes whether or not this pull request has been merged
	Merged bool `json:"merged"`

	// Number is the number of the pull request that can be used to merge
	Number int `json:"number"`

	// WebURL is the URL of the pull request in the git provider web interface.
	// +required
	WebURL string `json:"web_url"`
}

// TreeEntry contains info about each tree object's structure in TreeInfo whether it is a file or tree
type TreeEntry struct {
	// Path is the path of the file/blob or sub tree in a tree
	Path string `json:"path"`
	// Mode of the file/tree.
	// (100644:file (blob), 100755:executable (blob), 040000:subdirectory(tree),160000:submodule(commit),120000:blob that specifies the path of a symlink)
	Mode string `json:"mode"`
	// Type is the item type, It is either blob, tree, or commit.
	Type string `json:"type"`
	// Size is the size of the file/blob if the type is a blob, it is not populated if the type is a tree
	Size int `json:"size"`
	// SHA is the SHA1 checksum ID of the object in the tree
	SHA string `json:"sha"`
	// URL is the url that can be used to retrieve the details of the blob, tree of commit
	URL string `json:"url"`
}

// TreeInfo contains high-level information about a git Tree representing the hierarchy between files in a Git repository
type TreeInfo struct {
	// SHA is the SHA1 checksum ID of the tree, or the branch name
	SHA string `json:"sha"`
	// Tree is the list of TreeEntry objects describing the structure of the tree
	Tree []*TreeEntry `json:"tree"`
	// Truncated represents whether a tree is truncated when fetching a tree
	// If truncated is true in the response when fetching a tree, then the number of items in the tree array exceeded the maximum limit
	Truncated bool `json:"truncated"`
}
