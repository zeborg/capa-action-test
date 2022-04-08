package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

var (
	OWNER = os.Getenv("GITHUB_REPOSITORY_OWNER")
	REPO  = strings.Split(os.Getenv("GITHUB_REPOSITORY"), "/")[1]
)

func GetGithubClientCtx(token string) (*github.Client, context.Context) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

func ListRepos(client *github.Client, ctx context.Context) ([]*github.Repository, error) {
	repos, _, err := client.Repositories.List(ctx, os.Getenv("GITHUB_REPOSITORY_OWNER"), nil)
	return repos, err
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

func CreateRef(client *github.Client, ctx context.Context) (*github.Reference, error) {
	var strPtr = new(string)
	*strPtr = "test-ref"

	ref, _, err := client.Git.GetRef(ctx, OWNER, REPO, os.Getenv("GITHUB_BASE_REF"))
	time.Sleep(5 * time.Second)
	log.Println("Ref ", ref)

	newRef := github.Reference{
		Ref:    strPtr,
		URL:    ref.URL,
		Object: ref.Object,
		NodeID: ref.NodeID,
	}

	if err == nil {
		fmt.Println(ref)
	} else {
		log.Fatal(err)
	}

	refNew, _, err := client.Git.CreateRef(ctx, OWNER, REPO, &newRef)
	return refNew, err
}
