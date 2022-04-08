package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v42/github"
	gh "github.com/zeborg/capa-action-test/github"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type updateRefRequest struct {
	SHA   *string `json:"sha"`
	Force *bool   `json:"force"`
}
type AMIBuildConfig struct {
	K8sReleases map[string]string `json:"k8s_releases"`
}

type ReleaseVersion struct {
	Major int
	Minor int
	Patch int
}

func (r *ReleaseVersion) toString() string {
	return "v" + strconv.Itoa(r.Major) + "." + strconv.Itoa(r.Minor) + "." + strconv.Itoa(r.Patch)
}

func BuildReleaseVersion(ver string) ReleaseVersion {
	verSplit := strings.Split(string(ver), ".")
	major, err := strconv.Atoi(strings.ReplaceAll(verSplit[0], "v", ""))
	check(err)
	minor, err := strconv.Atoi(verSplit[1])
	check(err)
	patch, err := strconv.Atoi(verSplit[2])
	check(err)

	return ReleaseVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// func main() {
// 	var m2, m3 string
// 	var updated bool
// 	url := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
// 	k8sReleaseResponse, err := http.Get(url)
// 	check(err)

// 	min1, err := ioutil.ReadAll(k8sReleaseResponse.Body)
// 	check(err)

// 	min1Release := BuildReleaseVersion(string(min1))

// 	log.Print("Info: min1Release: ReleaseVersion ", min1Release.toString())
// 	log.Print("Info: min1Release: Major ", min1Release.Major, ", Minor ", min1Release.Minor, ", Patch ", min1Release.Patch)
// 	fmt.Println()

// 	if min1Release.Minor >= 2 {
// 		m2 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-1)
// 		m3 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-2)
// 	}

// 	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m2)
// 	k8sReleaseResponse, err = http.Get(url)
// 	check(err)

// 	min2, err := ioutil.ReadAll(k8sReleaseResponse.Body)
// 	check(err)

// 	min2Release := BuildReleaseVersion(string(min2))

// 	log.Print("Info: min2Release: ReleaseVersion ", min2Release.toString())
// 	log.Print("Info: min2Release: Major ", min2Release.Major, ", Minor ", min2Release.Minor, ", Patch ", min2Release.Patch)
// 	fmt.Println()

// 	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m3)
// 	k8sReleaseResponse, err = http.Get(url)
// 	check(err)

// 	min3, err := ioutil.ReadAll(k8sReleaseResponse.Body)
// 	check(err)

// 	min3Release := BuildReleaseVersion(string(min3))

// 	log.Print("Info: min2Release: ReleaseVersion ", min3Release.toString())
// 	log.Print("Info: min2Release: Major ", min3Release.Major, ", Minor ", min3Release.Minor, ", Patch ", min3Release.Patch)
// 	fmt.Println()

// 	latestAMIBuildConfig := &AMIBuildConfig{
// 		K8sReleases: map[string]string{
// 			"min1": string(min1),
// 			"min2": string(min2),
// 			"min3": string(min3),
// 		},
// 	}

// 	fmt.Println(*latestAMIBuildConfig)

// 	latestAMIBuildConfigFileBytes, err := json.MarshalIndent(latestAMIBuildConfig, "", "  ")
// 	check(err)

// 	AMIBuildConfigFilepath := "AMIBuildConfig.json"

// 	echoExecPath, err := exec.LookPath("echo")
// 	check(err)

// 	echoCmdTrue := &exec.Cmd{
// 		Path:   echoExecPath,
// 		Args:   []string{echoExecPath, "::set-output", "name=updated::true"},
// 		Stdout: os.Stdout,
// 		Stderr: os.Stdout,
// 	}

// 	echoCmdFalse := &exec.Cmd{
// 		Path:   echoExecPath,
// 		Args:   []string{echoExecPath, "::set-output", "name=updated::false"},
// 		Stdout: os.Stdout,
// 		Stderr: os.Stdout,
// 	}

// 	dat, err := os.ReadFile(AMIBuildConfigFilepath)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			err = os.WriteFile("AMIBuildConfig.json", latestAMIBuildConfigFileBytes, 0666)
// 			check(err)
// 			log.Printf("Info: Created \"AMIBuildConfig.json\" K8s versions \"%s\"", latestAMIBuildConfig.K8sReleases)

// 			err = echoCmdTrue.Run()
// 			check(err)

// 			return
// 		} else {
// 			err = echoCmdFalse.Run()
// 			check(err)

// 			log.Fatal(err)
// 		}
// 	}

// 	currentAMIBuildConfig := new(AMIBuildConfig)
// 	err = json.Unmarshal(dat, currentAMIBuildConfig)
// 	check(err)
// 	if !cmp.Equal(currentAMIBuildConfig, latestAMIBuildConfig) {
// 		err = os.WriteFile(AMIBuildConfigFilepath, latestAMIBuildConfigFileBytes, 0666)
// 		check(err)

// 		log.Printf("Info: Updated \"%s\" with K8s versions from \"%s\" to \"%s\"", AMIBuildConfigFilepath, currentAMIBuildConfig.K8sReleases, latestAMIBuildConfig.K8sReleases)
// 		updated = true
// 	} else {
// 		log.Printf("Info: \"%s\" is up-to-date.", AMIBuildConfigFilepath)
// 		updated = false
// 	}

// 	if updated {
// 		err = echoCmdTrue.Run()
// 		check(err)
// 	} else {
// 		err = echoCmdFalse.Run()
// 		check(err)
// 	}
// }

func main() {
	fmt.Println(os.Getenv("GITHUB_REPOSITORY"))
	client, ctx := gh.GetGithubClientCtx(os.Getenv("GITHUB_TOKEN"))

	// repos, err := github.ListRepos(client, ctx)
	// if err == nil {
	// 	fmt.Println(repos)
	// } else {
	// 	panic(err)
	// }

	// issue, err := github.CreateIssue(client, ctx)
	// if err == nil {
	// 	log.Println(issue)
	// } else {
	// 	log.Fatal(err)
	// }

	baseRef, headRef := "refs/heads/test-ghapi", "refs/heads/test-ref"

	ref, _, err := client.Git.GetRef(ctx, gh.OWNER, gh.REPO, headRef)
	if err != nil {
		if ref == nil {
			ref, err = gh.CreateRef(client, ctx, baseRef, headRef)
			if err == nil {
				log.Println(ref)
			} else {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	// 1. GET A REFERENCE TO HEAD
	ref, _, err = client.Git.GetRef(ctx, gh.OWNER, gh.REPO, headRef)
	if err == nil {
		fmt.Println("REFERENCE TO HEAD: ", ref)
	} else {
		log.Fatal(err)
	}

	// 2. GRAB THE COMMIT THAT HEAD POINTS TO
	parentCommit, _, err := client.Git.GetCommit(ctx, gh.OWNER, gh.REPO, *ref.Object.SHA)
	if err == nil {
		fmt.Println("HEAD Commit: ", parentCommit)
	} else {
		log.Fatal(err)
	}

	// 3. POST YOUR NEW FILE TO THE SERVER
	blobContent := base64.RawStdEncoding.EncodeToString([]byte(`Test blob content`))
	fmt.Println("Base64 encoded blobContent: ", blobContent)
	encoding := "base64"
	newBlob := github.Blob{
		Content:  &blobContent,
		Encoding: &encoding,
	}
	blob, _, err := client.Git.CreateBlob(
		ctx,
		gh.OWNER,
		gh.REPO,
		&newBlob,
	)
	if err == nil {
		fmt.Println("BLOBL_SHA: ", *blob.SHA, "\tBLOB_URL:", *blob.URL)
	} else {
		log.Fatal(err)
	}

	// 4. GET A HOLD OF THE TREE THAT THE COMMIT IN STEP 2. POINTS TO
	baseTree, _, err := client.Git.GetTree(ctx, gh.OWNER, gh.REPO, *parentCommit.Tree.SHA, true)
	if err == nil {
		fmt.Println(baseTree)
	} else {
		log.Fatal(err)
	}

	// 5. CREATE A TREE CONTAINING YOUR NEW / UPDATED FILE
	treePath := "ci/ami/AMIBuildConfig.json"
	treeMode := "100644"
	testTreeEntry := github.TreeEntry{
		Path: &treePath,
		Mode: &treeMode,
		SHA:  blob.SHA,
	}

	newTree, _, err := client.Git.CreateTree(ctx, gh.OWNER, gh.REPO, *baseTree.SHA, []*github.TreeEntry{&testTreeEntry})
	if err == nil {
		fmt.Println(newTree)
	} else {
		log.Fatal(err)
	}

	// 6. CREATE A NEW COMMIT
	commitMsg := "Test commit"
	newCommit := github.Commit{
		Message: &commitMsg,
		Tree:    newTree,
		Parents: []*github.Commit{parentCommit},
	}

	commit, _, err := client.Git.CreateCommit(ctx, gh.OWNER, gh.REPO, &newCommit)
	if err == nil {
		fmt.Println(commit)
	} else {
		log.Fatal(err)
	}

	// 7. UPDATE HEAD
	ref, _, err = client.Git.GetRef(ctx, gh.OWNER, gh.REPO, "refs/heads/test-ref")
	if err == nil {
		fmt.Println("REFERENCE TO HEAD: ", ref)
	} else {
		log.Fatal(err)
	}

	commitType := "string"
	ref.Object.SHA = commit.SHA
	ref.Object.URL = commit.URL
	ref.Object.Type = &commitType

	updateRef, _, err := client.Git.UpdateRef(ctx, gh.OWNER, gh.REPO, ref, true)
	if err == nil {
		fmt.Println(updateRef)
	} else {
		log.Fatal(err)
	}

}
