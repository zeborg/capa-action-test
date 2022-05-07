package github

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v42/github"
	"github.com/zeborg/capa-action-test/custom"
)

var (
	OWNER_REPO  = os.Getenv("GITHUB_REPOSITORY")
	OWNER, REPO = strings.Split(OWNER_REPO, "/")[0], strings.Split(OWNER_REPO, "/")[1]
)

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
			custom.CheckError(err)

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
	custom.CheckError(err)

	// get the commit pointed by the head branch
	parentCommit, _, err := client.Git.GetCommit(ctx, OWNER, REPO, *ref.Object.SHA)
	custom.CheckError(err)

	// upload the base64 encoded blob for updated amibuildconfig to github server
	blob, err := CreateBlob(client, ctx, "base64", blobBytes)
	custom.CheckError(err)

	// get the tree pointed by the head branch
	baseTree, _, err := client.Git.GetTree(ctx, OWNER, REPO, *parentCommit.Tree.SHA, true)
	custom.CheckError(err)

	// create a new tree with the updated amibuildconfig
	newTree, err := CreateTree(client, ctx, AMIBuildConfigFilename, "100644", *baseTree.SHA, *blob.SHA)
	custom.CheckError(err)

	// create a new commit with our newly created tree
	commitMsg := fmt.Sprintf("⚓️ Updating `%s`", AMIBuildConfigFilename)
	newCommit := github.Commit{
		Message: &commitMsg,
		Tree:    newTree,
		Parents: []*github.Commit{parentCommit},
	}
	commit, _, err := client.Git.CreateCommit(ctx, OWNER, REPO, &newCommit)
	custom.CheckError(err)

	// update the head to point to our newly created commit
	_, err = UpdateRef(client, ctx, ref, commit)
	custom.CheckError(err)

	// create pr to update the amibuildconfig
	prTitle := fmt.Sprintf("[CAPA-Action] ⚓️ Updating `%s`", AMIBuildConfigFilename)
	prBody := fmt.Sprintf("Updated config:\n```json\n%s\n```", string(blobBytes))
	prCreated, err := CreatePR(client, ctx, false, prTitle, prHeadRef, prBaseRef, prBody)
	custom.CheckError(err)

	// add labels to the newly created pr
	labels := []string{"ami-build-action"}
	_, _, err = client.Issues.AddLabelsToIssue(ctx, OWNER, REPO, *prCreated.Number, labels)
	custom.CheckError(err)

	// request reviewers for the newly created pr
	_, err = RequestReviewers(client, ctx, *prCreated.Number)
	custom.CheckError(err)

	// add assignees to the newly created pr
	_, _, err = client.Issues.AddAssignees(ctx, OWNER, REPO, *prCreated.Number, strings.Split(os.Getenv("CAPA_ACTION_PR_ASSIGNEES"), ","))
	custom.CheckError(err)

	return true
}
