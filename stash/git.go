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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

var licenseURLs = map[gitprovider.LicenseTemplate]string{
	gitprovider.LicenseTemplate("apache-2.0"): "https://www.apache.org/licenses/LICENSE-2.0.txt",
	gitprovider.LicenseTemplate("gpl-3.0"):    "https://www.gnu.org/licenses/gpl-3.0-standalone.html",
}

// Git interface defines the methods that can be used to
// communicate with the git protocol.
type Git interface {
	CleanCloner
	CleanIniter
	Committer
	Brancher
	Pusher
}

// CleanCloner interface defines the methods that can be used to Clone a repository
// and clean it up afterwards.
type CleanCloner interface {
	CloneRepository(ctx context.Context, URL string) (r *git.Repository, dir string, err error)
	Cleaner
}

// CleanIniter interface defines the methods that can be used to initialize a repository
// and clean it up afterwards.
type CleanIniter interface {
	InitRepository(c *CreateCommit, createRemote bool) (r *git.Repository, dir string, err error)
	Cleaner
}

// Cleaner interface defines the methods that can be used to clean up a directory
type Cleaner interface {
	Cleanup(dir string) error
}

// Committer interface defines the methods that can be used to commit to a repository
type Committer interface {
	CreateCommit(rPath string, r *git.Repository, branchName string, c *CreateCommit) (*Commit, error)
}

// Brancher interface defines the methods that can be used to create a new branch
type Brancher interface {
	CreateBranch(branchName string, r *git.Repository, commitID string) error
}

// Pusher interface defines the methods that can be used to push to a repository
type Pusher interface {
	Push(ctx context.Context, r *git.Repository) error
}

// GitService is a client for communicating with stash users endpoint
type GitService service

// Commit is a version of the repository
type Commit struct {
	// SHA of the commit.
	SHA string `json:"sha,omitempty"`
	// Author is the author of the commit.
	Author *CommitAuthor `json:"author,omitempty"`
	// Committer is the committer of the commit.
	Committer *CommitAuthor `json:"committer,omitempty"`
	// Message is the commit message.
	Message string `json:"message,omitempty"`
	// Tree is the three of git objects that this commit points to.
	Tree string `json:"tree,omitempty"`
	// Parents is the list of parents commit.
	Parents []string `json:"parents,omitempty"`
	// PGPSignature is the PGP signature of the commit.
	PGPSignature string `json:"pgp_signature,omitempty"`
}

// CommitAuthor represents the author or committer of a commit. The commit
// author may not correspond to a GitHub User.
type CommitAuthor struct {
	// Date is the date of the commit
	Date int64 `json:"date,omitempty"`
	// Name is the name of the author
	Name string `json:"name,omitempty"`
	// Email is the email of the author
	Email string `json:"email,omitempty"`
}

// CreateCommit creates a new commit in the repository.
type CreateCommit struct {
	// Author is the author of the commit.
	Author *CommitAuthor `json:"author,omitempty"`
	// Committer is the committer of the commit.
	Committer *CommitAuthor `json:"committer,omitempty"`
	// Message is the commit message.
	Message string `json:"message,omitempty"`
	// Parents is the list of parents commit.
	Parents []string `json:"parents,omitempty"`
	// URL is the URL of the commit.
	URL string `json:"url,omitempty"`
	// Files is the list of files to commit.
	Files []CommitFile `json:"files,omitempty"`
	// SigningKey denotes a key to sign the commit with. If not nil this key will
	// be used to sign the commit. The private key must be present and already
	// decrypted.
	SignKey *openpgp.Entity `json:"-"`
}

// CommitFile is a file to commit
type CommitFile struct {
	// The path of the file relative to the repository root.
	Path *string `json:"path"`
	// The contents of the file.
	Content *string `json:"content"`
}

// GitCommitOptionsFunc is a function that returns an error if the commit options are invalid
type GitCommitOptionsFunc func(c *CreateCommit) error

// WithURL is a currying function for the URL field
func WithURL(url string) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if url == "" {
			return errors.New("url is required")
		}
		c.URL = url
		return nil
	}
}

// WithAuthor is a currying function for the Author field
func WithAuthor(author *CommitAuthor) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if author == nil {
			return errors.New("Author is required")
		}
		c.Author = author
		return nil
	}
}

// WithCommitter is a currying function for the Committer field
func WithCommitter(committer *CommitAuthor) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if committer == nil {
			return errors.New("Committer is required")
		}
		c.Committer = committer
		return nil
	}
}

// WithMessage is a currying function for the message field
func WithMessage(message string) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if message == "" {
			return errors.New("message is required")
		}
		c.Message = message
		return nil
	}
}

// WithFiles is a currying function for the files field
func WithFiles(files []CommitFile) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if files != nil && len(files) != 0 {
			c.Files = files
			return nil
		}
		return errors.New("files are required")
	}
}

// WithSignature is a currying function for the signKey field
func WithSignature(signKey *openpgp.Entity) GitCommitOptionsFunc {
	return func(c *CreateCommit) error {
		if signKey != nil {
			c.SignKey = signKey
			return nil
		}
		return errors.New("SignKey required")
	}
}

// NewCommit is a helper function to create a CreateCommit object
// Use the currying functions provided to pass in the commit options
func NewCommit(opts ...GitCommitOptionsFunc) (*CreateCommit, error) {
	c := &CreateCommit{}
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	if c.Author == nil {
		return nil, errors.New("commit author: invalid parameters")
	}
	if c.Message == "" {
		return nil, errors.New("commit message: invalid parameters")
	}
	if c.URL == "" {
		return nil, errors.New("commit url: invalid parameters")
	}

	return c, nil
}

// CreateCommit creates a commit for the given CommitFiles. The commit is not pushed.
// The commit is signed with the given SignKey when provided.
// When committer is nil, author is used as the committer.
// An optional branch name can be provided to checkout the branch before committing.
func (s *GitService) CreateCommit(rPath string, r *git.Repository, branchName string, c *CreateCommit) (*Commit, error) {
	if c == nil {
		return nil, errors.New("commit must be provided")
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	if branchName != "" {
		err := s.CreateBranch(branchName, r, "")
		if err != nil {
			return nil, err
		}
	}

	err = s.addCommitFiles(w, rPath, c.Files)
	if err != nil {
		return nil, err
	}

	// Set the committer & author DATE
	now := time.Now().Unix()
	c.Author.Date = now
	if c.Committer != nil {
		c.Committer.Date = now
	}

	obj, err := s.commit(w, r, c)
	if err != nil {
		return nil, err
	}

	var parents []string
	if obj.ParentHashes == nil {
		for _, parent := range obj.ParentHashes {
			parents = append(parents, parent.String())
		}
	}

	commit := &Commit{
		SHA: obj.Hash.String(),
		Author: &CommitAuthor{
			Date:  obj.Author.When.Unix(),
			Name:  obj.Author.Name,
			Email: obj.Author.Email,
		},
		Committer: &CommitAuthor{
			Date:  obj.Committer.When.Unix(),
			Name:  obj.Committer.Name,
			Email: obj.Committer.Email,
		},
		Message:      obj.Message,
		Tree:         obj.TreeHash.String(),
		Parents:      parents,
		PGPSignature: obj.PGPSignature,
	}

	return commit, nil
}

// CloneRepository clones the repository at the given URL to the given path.
// The repository will be cloned into a temporary directory which shall be clean up by the caller.
func (s *GitService) CloneRepository(ctx context.Context, URL string) (r *git.Repository, dir string, err error) {
	dir, err = os.MkdirTemp("", "repo-*")
	if err != nil {
		return nil, "", err
	}

	r, err = git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:      URL,
		Auth:     &githttp.BasicAuth{Username: s.Client.username, Password: s.Client.token},
		CABundle: s.Client.caBundle,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repository: %v", err)
	}

	err = r.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		Auth:     &githttp.BasicAuth{Username: s.Client.username, Password: s.Client.token},
		CABundle: s.Client.caBundle,
	})

	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch repository: %v", err)
	}

	return r, dir, nil
}

func (s *GitService) addCommitFiles(w *git.Worktree, dir string, files []CommitFile) error {
	for _, file := range files {
		err := writeCommitFile(file, dir)
		if err != nil {
			return err
		}
		// Adds the new file to the staging area.
		_, err = w.Add(*file.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeCommitFile(file CommitFile, dir string) error {
	filename := filepath.Join(dir, *file.Path)
	filePath := strings.Split(*file.Path, "/")
	if len(filePath) > 1 {
		fullPath := append([]string{dir}, filePath[0:len(filePath)-1]...)
		err := os.MkdirAll(strings.Join(fullPath, "/"), 0777)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(filename, []byte(*file.Content), 0644)
	if err != nil {
		return err
	}

	return nil
}

// Cleanup removes the temporary directory created for the repository.
func (s *GitService) Cleanup(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return nil
}

// CreateBranch creates a new branch with the given name and checkout the branch.
// An optional commit id can be provided to checkout the branch at the given commit.
func (s *GitService) CreateBranch(branchName string, r *git.Repository, commitID string) error {
	w, err := r.Worktree()
	if err != nil {
		return err
	}

	branch := fmt.Sprintf("refs/heads/%s", branchName)
	b := plumbing.ReferenceName(branch)

	if commitID != "" {
		commitHash := plumbing.NewHash(commitID)
		err = w.Checkout(&git.CheckoutOptions{
			Hash: commitHash,
		})

		if err != nil {
			return fmt.Errorf("failed to checkout commit: %v", err)
		}

		// make we are on the correct Head
		head, err := r.Head()
		if err != nil {
			return fmt.Errorf("failed to get head: %v", err)
		}

		if head.Hash() != commitHash {
			return fmt.Errorf("commit %s not found", commitID)
		}

		ref := plumbing.NewHashReference(b, plumbing.NewHash(commitID))
		err = w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: plumbing.ReferenceName(ref.Name().String())})
		if err != nil {
			return err
		}

		return nil

	}

	// First try to checkout branch
	err = w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b})

	if err != nil {
		// got an error  - try to create it
		err := w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})
		if err != nil {
			return err
		}
	}

	return nil
}

// InitRepository is a function to create a new repository.
// The caller must clean up the directory after the function returns.
func (s *GitService) InitRepository(c *CreateCommit, createRemote bool) (r *git.Repository, dir string, err error) {
	dir, err = os.MkdirTemp("", "repo-*")
	if err != nil {
		return nil, "", err
	}

	gitDir := osfs.New(dir + "/.git")
	fs := osfs.New(dir)
	r, err = git.Init(filesystem.NewStorage(gitDir, cache.NewObjectLRUDefault()), fs)
	if err != nil {
		return nil, "", err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, "", err
	}

	err = s.addCommitFiles(w, dir, c.Files)
	if err != nil {
		return nil, "", err
	}

	if createRemote {
		rc := &config.RemoteConfig{Name: "origin", URLs: []string{c.URL}}
		_, err = r.CreateRemote(rc)
		if err != nil {
			return nil, "", err
		}
	}

	_, err = s.commit(w, r, c)
	if err != nil {
		return nil, "", err
	}

	return r, dir, nil
}

// commit creates a new commit with the given commit object.
// The commit is pushed to the remote repository.
func (s *GitService) commit(w *git.Worktree, r *git.Repository, c *CreateCommit) (*object.Commit, error) {
	// Commits the current staging area to the repository, with the new file
	// just created. We should provide the object.Signature of Author of the
	// gitClient Since version 5.0.1, we can omit the Author signature, being read
	// from the git config files.
	var p []plumbing.Hash
	if c.Parents != nil && len(c.Parents) > 0 {
		p = make([]plumbing.Hash, len(c.Parents))
	}
	if p != nil && len(p) > 0 {
		for i, parent := range c.Parents {
			copy(p[i][:], parent)
		}
	}

	// calculate time.Time from unix Time
	authorDate := time.Unix(c.Author.Date, 0)
	var committer *object.Signature
	if c.Committer != nil {
		committerDate := time.Unix(c.Committer.Date, 0)
		committer = &object.Signature{
			Name:  c.Committer.Name,
			Email: c.Committer.Email,
			When:  committerDate,
		}
	} else {
		committer = &object.Signature{
			Name:  c.Author.Name,
			Email: c.Author.Email,
			When:  authorDate,
		}
	}

	commitHash, err := w.Commit(c.Message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  c.Author.Name,
			Email: c.Author.Email,
			When:  authorDate,
		},
		Committer: committer,
		Parents:   p,
		SignKey:   c.SignKey,
		All:       true,
	})
	if err != nil {
		return nil, err
	}

	obj, err := r.CommitObject(commitHash)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// Push commits the current changes to the remote repository.
func (s *GitService) Push(ctx context.Context, r *git.Repository) error {

	options := &git.PushOptions{
		RemoteName: "origin",
		Auth:       &githttp.BasicAuth{Username: s.Client.username, Password: s.Client.token},
		CABundle:   s.Client.caBundle,
	}

	err := r.PushContext(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to push to remote: %w", err)
	}

	return nil
}

func getLicense(license gitprovider.LicenseTemplate) (string, error) {

	licenseURL, ok := licenseURLs[license]
	if !ok {
		return "", fmt.Errorf("license: %s, not supported", license)
	}
	return downloadFile(licenseURL)
}

// downloadFile will download a url to a string.
func downloadFile(url string) (string, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out := new(strings.Builder)

	// Write the body to the string builder
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
