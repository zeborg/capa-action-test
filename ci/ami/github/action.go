package github

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v42/github"
)

var (
	OWNER_REPO  = os.Getenv("GITHUB_REPOSITORY")
	OWNER, REPO = strings.Split(OWNER_REPO, "/")[0], strings.Split(OWNER_REPO, "/")[1]
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

			CreateRef(client, ctx, baseRef, headRef)
			log.Printf("Info: Recreated existing head reference: %s", headRef)
		} else {
			log.Printf("Info: PR #%d corresponding to the specified base branch \"%s\" and head branch \"%s\" is still open. Exiting.", *prList[0].Number, baseRef, headRef)
			return false
		}
	} else {
		if ref == nil {
			CreateRef(client, ctx, baseRef, headRef)
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
	blob, err := CreateBlob(client, ctx, "base64", blobBytes)
	checkError(err)

	// get the tree pointed by the head branch
	baseTree, _, err := client.Git.GetTree(ctx, OWNER, REPO, *parentCommit.Tree.SHA, true)
	checkError(err)

	// create a new tree with the updated amibuildconfig
	newTree, err := CreateTree(client, ctx, AMIBuildConfigFilename, "100644", *baseTree.SHA, *blob.SHA)
	checkError(err)

	// create a new commit with our newly created tree
	commitMsg := fmt.Sprintf("⚓️ Updating `%s`", AMIBuildConfigFilename)
	newCommit := github.Commit{
		Message: &commitMsg,
		Tree:    newTree,
		Parents: []*github.Commit{parentCommit},
	}
	commit, _, err := client.Git.CreateCommit(ctx, OWNER, REPO, &newCommit)
	checkError(err)

	// update the head to point to our newly created commit
	_, err = UpdateRef(client, ctx, ref, commit)
	checkError(err)

	// create pr to update the amibuildconfig
	prTitle := fmt.Sprintf("[CAPA-Action] ⚓️ Updating `%s`", AMIBuildConfigFilename)
	prBody := fmt.Sprintf("Updated config:\n```json\n%s\n```", string(blobBytes))
	prCreated, err := CreatePR(client, ctx, false, prTitle, prHeadRef, prBaseRef, prBody)
	checkError(err)

	// request reviewers for the newly created pr
	reviewers := []string{"zeborg"}
	reqReviewers := github.ReviewersRequest{
		Reviewers: reviewers,
	}
	_, _, err = client.PullRequests.RequestReviewers(ctx, OWNER, REPO, *prCreated.Number, reqReviewers)
	checkError(err)

	// add labels to the pr
	labels := []string{"ami-build-action"}
	_, _, err = client.Issues.AddLabelsToIssue(ctx, OWNER, REPO, *prCreated.Number, labels)
	checkError(err)

	return true
}
