package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/zeborg/capa-action-test/github"
)

func Shell(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
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
	verSplit := strings.Split(ver, ".")
	major, err := strconv.Atoi(strings.ReplaceAll(verSplit[0], "v", ""))
	checkError(err)
	minor, err := strconv.Atoi(verSplit[1])
	checkError(err)
	patch, err := strconv.Atoi(verSplit[2])
	checkError(err)

	return ReleaseVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

func main() {
	mode := flag.String("mode", "", "Acceptable values: 'github' (for CAPA GitHub Action) and 'prow' (for CAPA Prow Jobs)")
	flag.Parse()

	if *mode == "github" {
		gh()
	} else if *mode == "presubmit" {
		presubmit()
	} else if *mode == "postsubmit" {
		postsubmit()
	} else if *mode == "" {
		log.Fatal("Error: Value not provided for 'mode' flag\n")
	} else {
		log.Fatalf("Error: Invalid value '%s' found for 'mode' flag\n", *mode)
	}
}

func gh() {
	log.Println("gh()")
	var m2, m3 string
	url := "https://raw.githubusercontent.com/zeborg/capa-action-test/main/stable.txt"
	k8sReleaseResponse, err := http.Get(url)
	checkError(err)

	min1, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	checkError(err)

	min1Release := BuildReleaseVersion(string(min1))
	log.Print("Info: min1Release: Major ", min1Release.Major, ", Minor ", min1Release.Minor, ", Patch ", min1Release.Patch)

	if min1Release.Minor >= 2 {
		m2 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-1)
		m3 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-2)
	}

	url = fmt.Sprintf("https://raw.githubusercontent.com/zeborg/capa-action-test/main/stable-%s.txt", m2)
	k8sReleaseResponse, err = http.Get(url)
	checkError(err)

	min2, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	checkError(err)

	min2Release := BuildReleaseVersion(string(min2))
	log.Print("Info: min2Release: Major ", min2Release.Major, ", Minor ", min2Release.Minor, ", Patch ", min2Release.Patch)

	url = fmt.Sprintf("https://raw.githubusercontent.com/zeborg/capa-action-test/main/stable-%s.txt", m3)
	k8sReleaseResponse, err = http.Get(url)
	checkError(err)

	min3, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	checkError(err)

	min3Release := BuildReleaseVersion(string(min3))
	log.Print("Info: min3Release: Major ", min3Release.Major, ", Minor ", min3Release.Minor, ", Patch ", min3Release.Patch)

	latestAMIBuildConfig := &AMIBuildConfig{
		K8sReleases: map[string]string{
			"min1": string(min1),
			"min2": string(min2),
			"min3": string(min3),
		},
	}

	latestAMIBuildConfigFileBytes, err := json.MarshalIndent(latestAMIBuildConfig, "", "  ")
	checkError(err)

	AMIBuildConfigFilename := os.Getenv("AMI_BUILD_CONFIG_FILENAME")
	dat, err := os.ReadFile(AMIBuildConfigFilename)
	if err != nil {
		if os.IsNotExist(err) {
			github.Action(latestAMIBuildConfigFileBytes, AMIBuildConfigFilename)
			log.Printf("Info: Created \"AMIBuildConfig.json\" K8s versions \"%s\"", latestAMIBuildConfig.K8sReleases)
			return
		} else {
			log.Fatal(err)
		}
	}

	currentAMIBuildConfig := new(AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	checkError(err)
	if !cmp.Equal(currentAMIBuildConfig, latestAMIBuildConfig) {
		prCreated := github.Action(latestAMIBuildConfigFileBytes, AMIBuildConfigFilename)
		if prCreated {
			log.Printf("Info: Updated \"%s\" with K8s versions from \"%s\" to \"%s\"", AMIBuildConfigFilename, currentAMIBuildConfig.K8sReleases, latestAMIBuildConfig.K8sReleases)
		}
	} else {
		log.Printf("Info: \"%s\" is up-to-date.", AMIBuildConfigFilename)
	}
}

func presubmit() {
	AMIBuildConfigFilename := os.Getenv("AMI_BUILD_CONFIG_FILENAME")
	dat, err := os.ReadFile(AMIBuildConfigFilename)
	checkError(err)

	currentAMIBuildConfig := new(AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	checkError(err)

	for _, v := range currentAMIBuildConfig.K8sReleases {
		err, out, _ := Shell(fmt.Sprintf("./clusterawsadm ami list --kubernetes-version %s", strings.ReplaceAll(v, "v", "")))
		checkError(err)

		if out == "" {
			log.Printf("Info: Building AMI for Kubernetes %s.", v)
			err, out, errout := Shell("sh ./image-builder.sh")
			checkError(err)

			if errout != "" {
				log.Fatalf("Error: %s", errout)
			} else {
				log.Println("Info: image-builder log")
				log.Println(out)
			}
		} else {
			log.Printf("Info: AMI for Kubernetes %s already exists. Skipping image building.", v)
		}
	}
}

func postsubmit() {
	log.Println("postsubmit()")
}
