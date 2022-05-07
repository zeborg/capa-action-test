package github

import (
	"context"
	"encoding/base64"
	"os"
	"strings"

	"github.com/google/go-github/v42/github"
	"github.com/zeborg/capa-action-test/custom"
	"golang.org/x/oauth2"
)

func GetGithubClientCtx(token string) (*github.Client, context.Context) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

func CreateIssue(client *github.Client, ctx context.Context) (*github.Issue, error) {
	MyIssue := struct {
		Title    string
		Body     string
		Labels   []string
		Assignee string
	}{
		Title:    "Test issue",
		Body:     "Test issue body",
		Labels:   []string{"test-issue-label"},
		Assignee: "zeborg",
	}
	testIssue := &github.IssueRequest{
		Title:    &MyIssue.Title,
		Body:     &MyIssue.Body,
		Labels:   &MyIssue.Labels,
		Assignee: &MyIssue.Assignee,
	}

	issue, _, err := client.Issues.Create(ctx, OWNER, REPO, testIssue)

	return issue, err
}

func CreateRef(client *github.Client, ctx context.Context, fromRef, toRef string) *github.Reference {
	ref, _, err := client.Git.GetRef(ctx, OWNER, REPO, fromRef)
	custom.CheckError(err)

	newRef := github.Reference{
		Ref:    &toRef,
		URL:    ref.URL,
		Object: ref.Object,
	}
	refNew, _, err := client.Git.CreateRef(ctx, OWNER, REPO, &newRef)
	custom.CheckError(err)

	return refNew
}

func CreateBlob(client *github.Client, ctx context.Context, encoding string, blobBytes []byte) (*github.Blob, error) {
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

	return blob, err
}

func CreateTree(client *github.Client, ctx context.Context, filename string, mode string, baseSHA, blobSHA string) (*github.Tree, error) {
	treePath := "ci/ami/" + filename
	treeMode := "100644"
	newTreeEntry := github.TreeEntry{
		Path: &treePath,
		Mode: &treeMode,
		SHA:  &blobSHA,
	}
	newTree, _, err := client.Git.CreateTree(ctx, OWNER, REPO, baseSHA, []*github.TreeEntry{&newTreeEntry})

	return newTree, err
}

func CreatePR(client *github.Client, ctx context.Context, prModify bool, prTitle, prHeadRef, prBaseRef, prBody string) (*github.PullRequest, error) {
	newPR := github.NewPullRequest{
		Title:               &prTitle,
		Head:                &prHeadRef,
		Base:                &prBaseRef,
		Body:                &prBody,
		MaintainerCanModify: &prModify,
	}
	prCreated, _, err := client.PullRequests.Create(ctx, OWNER, REPO, &newPR)

	return prCreated, err
}

func UpdateRef(client *github.Client, ctx context.Context, ref *github.Reference, commit *github.Commit) (*github.Reference, error) {
	refObjType := "commit"
	ref.Object.SHA = commit.SHA
	ref.Object.URL = commit.URL
	ref.Object.Type = &refObjType

	newRef, _, err := client.Git.UpdateRef(ctx, OWNER, REPO, ref, true)

	return newRef, err
}

func RequestReviewers(client *github.Client, ctx context.Context, prNum int) (*github.PullRequest, error) {
	reqReviewers := github.ReviewersRequest{
		Reviewers: strings.Split(os.Getenv("CAPA_ACTION_PR_REVIEWERS"), ","),
	}
	pr, _, err := client.PullRequests.RequestReviewers(ctx, OWNER, REPO, prNum, reqReviewers)

	return pr, err
}
