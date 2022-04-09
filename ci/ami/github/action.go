package github

import (
	"encoding/base64"
	"log"
	"os"

	"github.com/google/go-github/v42/github"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Action(blobBytes []byte, AMIBuildConfigFilename string) {
	// create a github api client and context using our action's auto-generated github token
	client, ctx := GetGithubClientCtx(os.Getenv("GITHUB_TOKEN"))

	baseRef, headRef := "refs/heads/"+os.Getenv("CAPA_ACTION_BASE_BRANCH"), "refs/heads/"+os.Getenv("CAPA_ACTION_HEAD_BRANCH")

	// check if the required head branch already exists
	ref, _, err := client.Git.GetRef(ctx, OWNER, REPO, headRef)
	if err != nil {
		if ref == nil {
			_, err = CreateRef(client, ctx, baseRef, headRef)
			checkError(err)
		} else {
			log.Fatal(err)
		}
	}

	// get reference to the head branch
	ref, _, err = client.Git.GetRef(ctx, OWNER, REPO, headRef)
	checkError(err)

	// get the commit pointed by the head branch
	parentCommit, _, err := client.Git.GetCommit(ctx, OWNER, REPO, *ref.Object.SHA)
	checkError(err)

	// upload the base64 encoded blob for updated ami config to github server
	encoding := "base64"
	blobContent := base64.RawStdEncoding.EncodeToString(blobBytes)
	newBlob := github.Blob{
		Content:  &blobContent,
		Encoding: &encoding,
	}
	blob, _, err := client.Git.CreateBlob(
		ctx,
		OWNER,
		REPO,
		&newBlob,
	)
	checkError(err)

	// get the tree pointed by the head branch
	baseTree, _, err := client.Git.GetTree(ctx, OWNER, REPO, *parentCommit.Tree.SHA, true)
	checkError(err)

	// create a new tree with the updated amibuildconfig
	treePath := "ci/ami/" + AMIBuildConfigFilename
	treeMode := "100644"
	testTreeEntry := github.TreeEntry{
		Path: &treePath,
		Mode: &treeMode,
		SHA:  blob.SHA,
	}

	newTree, _, err := client.Git.CreateTree(ctx, OWNER, REPO, *baseTree.SHA, []*github.TreeEntry{&testTreeEntry})
	checkError(err)

	// create a new commit with our newly created tree
	commitMsg := "Test commit"
	newCommit := github.Commit{
		Message: &commitMsg,
		Tree:    newTree,
		Parents: []*github.Commit{parentCommit},
	}

	commit, _, err := client.Git.CreateCommit(ctx, OWNER, REPO, &newCommit)
	checkError(err)

	// update the head to point to our newly created commit
	ref, _, err = client.Git.GetRef(ctx, OWNER, REPO, "refs/heads/test-ref")
	checkError(err)

	commitType := "string"
	ref.Object.SHA = commit.SHA
	ref.Object.URL = commit.URL
	ref.Object.Type = &commitType

	_, _, err = client.Git.UpdateRef(ctx, OWNER, REPO, ref, true)
	checkError(err)
}
