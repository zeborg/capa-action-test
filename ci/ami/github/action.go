package github

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v42/github"
)

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Action(blobBytes []byte, AMIBuildConfigFilename string) bool {
	// create a github api client and context using our action's auto-generated github token
	client, ctx := GetGithubClientCtx(os.Getenv("GITHUB_TOKEN"))

	// define references
	baseRef := "refs/heads/" + os.Getenv("CAPA_ACTION_BASE_BRANCH")
	headRef := "refs/heads/" + os.Getenv("CAPA_ACTION_HEAD_BRANCH")
	prHeadRef := OWNER + ":" + headRef
	prBaseRef := baseRef

	// check if the required head branch already exists
	ref, _, err := client.Git.GetRef(ctx, OWNER, REPO, headRef)
	if err == nil {
		prListOpts := github.PullRequestListOptions{
			Head: prHeadRef,
			Base: prBaseRef,
		}
		prList, _, err := client.PullRequests.List(ctx, OWNER, REPO, &prListOpts)
		if err != nil {
			if len(prList) != 0 {
				log.Fatal(err)
			}
		}

		if len(prList) == 0 {
			_, err := client.Git.DeleteRef(ctx, OWNER, REPO, headRef)
			checkError(err)

			_, err = CreateRef(client, ctx, baseRef, headRef)
			checkError(err)

			log.Printf("Info: Recreated existing head reference: %s", headRef)
		} else {
			log.Printf("Info: PR #%d corresponding to the specified base branch \"%s\" and head branch \"%s\" is still open. Exiting.\n", *prList[0].Number, baseRef, headRef)
			return false
		}
	} else {
		if ref == nil {
			_, err = CreateRef(client, ctx, baseRef, headRef)
			checkError(err)
		} else {
			log.Fatal(err)
		}
	}

	// get the reference to the head branch
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
	ref, _, err = client.Git.GetRef(ctx, OWNER, REPO, headRef)
	checkError(err)

	refObjType := "commit"
	ref.Object.SHA = commit.SHA
	ref.Object.URL = commit.URL
	ref.Object.Type = &refObjType

	_, _, err = client.Git.UpdateRef(ctx, OWNER, REPO, ref, true)
	checkError(err)

	// create pr to update the amibuildconfig
	prTitle := fmt.Sprintf("[CAPA-Action] ⚓️ Updating `%s`", AMIBuildConfigFilename)
	prBody := fmt.Sprintf("Updated config:\n```json\n%s\n```", string(blobBytes))
	prModify := false
	newPR := github.NewPullRequest{
		Title:               &prTitle,
		Head:                &prHeadRef,
		Base:                &prBaseRef,
		Body:                &prBody,
		MaintainerCanModify: &prModify,
	}

	prCreated, _, err := client.PullRequests.Create(ctx, OWNER, REPO, &newPR)
	checkError(err)

	reqReviewers := github.ReviewersRequest{
		Reviewers: []string{"zeborg"},
	}

	_, _, err = client.PullRequests.RequestReviewers(ctx, OWNER, REPO, *prCreated.Number, reqReviewers)
	checkError(err)

	_, _, err = client.Issues.AddLabelsToIssue(ctx, OWNER, REPO, *prCreated.Number, []string{"ami-build-action"})
	checkError(err)

	return true
}
