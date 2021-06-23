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

package stash

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

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

type commitHashes struct {
	hash     string
	treeHash string
}

func createCommit(ctx context.Context, cx *clientContext, remoteURL, userName, userEmail, branchName, message string, files []gitprovider.CommitFile) (*commitHashes, error) {

	if len(files) == 0 {
		return nil, gitprovider.ErrNotFound
	}

	dir, err := ioutil.TempDir("", "repo-*")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(dir)

	auth := &githttp.BasicAuth{
		Username: userName,
		Password: cx.token,
	}

	// Clone the given repository to the given directory
	cx.log.V(1).Info("git clone", "url", remoteURL, "directory", dir)
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:  remoteURL,
		Auth: auth})
	if err != nil {
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	err = r.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
		Auth:     auth})
	if err != nil {
		return nil, err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)),
		Force:  true,
	})
	if err != nil {
		return nil, err
	}
	/*
		headRef, err := r.Head()
		if err != nil {
			return nil, err
		}
		ref := plumbing.NewHashReference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)), headRef.Hash())
		err = w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: plumbing.ReferenceName(ref.Name().String())})
		if err != nil {
			return nil,err
		}
	*/
	for _, file := range files {
		filename := filepath.Join(dir, *file.Path)
		filePath := strings.Split(*file.Path, "/")
		if len(filePath) > 1 {
			fullPath := append([]string{dir}, filePath[0:len(filePath)-1]...)
			err := os.MkdirAll(strings.Join(fullPath, "/"), 0777)
			if err != nil {
				return nil, err
			}
		}
		err = ioutil.WriteFile(filename, []byte(*file.Content), 0644)
		if err != nil {
			return nil, err
		}

		// Adds the new file to the staging area.
		_, err = w.Add(*file.Path)
		if err != nil {
			return nil, err
		}

	}

	commitObj, err := commitPush(ctx, cx, remoteURL, userName, userEmail, message, w, r)
	if err != nil {
		return nil, err
	}

	return &commitHashes{hash: commitObj.Hash.String(), treeHash: commitObj.TreeHash.String()}, nil
}

func createBranch(ctx context.Context, cx *clientContext, remoteURL, userName, userEmail, branchName, commitID string) error {

	dir, err := ioutil.TempDir("", "repo-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	auth := &githttp.BasicAuth{
		Username: userName,
		Password: cx.token,
	}

	// Clone the given repository to the given directory
	cx.log.V(1).Info("git clone", "url", remoteURL, "directory", dir)
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:  remoteURL,
		Auth: auth})
	if err != nil {
		return err
	}

	cx.log.V(1).Info("git branch creation", "branch", branchName)

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commitID),
	})
	if err != nil {
		return err
	}

	ref := plumbing.NewHashReference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchName)), plumbing.NewHash(commitID))

	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Force:  false,
		Branch: plumbing.ReferenceName(ref.Name().String())})
	if err != nil {
		return err
	}

	_, err = commitPush(ctx, cx, remoteURL, userName, userEmail, "create branch", w, r)
	return err
}

func createInitialCommit(ctx context.Context, c stashClient, cx *clientContext, remoteURL, userName, userEmail, readmeContent string, license gitprovider.LicenseTemplate) error {
	dir, err := ioutil.TempDir("", "repo-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	gitDir := osfs.New(dir + "/.git")
	fs := osfs.New(dir)
	r, err := git.Init(filesystem.NewStorage(gitDir, cache.NewObjectLRUDefault()), fs)
	if err != nil {
		return err
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, "/README.md")
	err = ioutil.WriteFile(filename, []byte(readmeContent), 0644)
	if err != nil {
		return err
	}

	// Adds the new file to the staging area.
	_, err = w.Add("README.md")
	if err != nil {
		return err
	}

	if err := getLicense(dir, license); err == nil {
		// Adds the new file to the staging area.
		_, err = w.Add("LICENSE.md")
		if err != nil {
			return err
		}
	}

	rc := &config.RemoteConfig{Name: "origin", URLs: []string{remoteURL}}
	remote, err := r.CreateRemote(rc)
	if err != nil {
		return err
	}
	cx.log.V(1).Info("remote", "url", remote.Config().URLs[0])

	_, err = commitPush(ctx, cx, remoteURL, userName, userEmail, "initial commit", w, r)
	return err
}

func commitPush(ctx context.Context, cx *clientContext, remoteURL, userName, userEmail, message string, worktree *git.Worktree, repository *git.Repository) (*object.Commit, error) {

	// Commits the current staging area to the repository, with the new file
	// just created. We should provide the object.Signature of Author of the
	// commit Since version 5.0.1, we can omit the Author signature, being read
	// from the git config files.
	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userName,
			Email: userEmail,
			When:  time.Now(),
		},
		All: true,
	})
	if err != nil {
		return nil, err
	}

	obj, err := repository.CommitObject(commit)
	if err != nil {
		return nil, err
	}

	auth := &githttp.BasicAuth{
		Username: userName,
		Password: cx.token,
	}

	options := &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth}

	err = repository.PushContext(ctx, options)
	if err != nil {
		return nil, err
	}

	return obj, nil
}
