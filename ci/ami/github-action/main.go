package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/google/go-cmp/cmp"
	"github.com/zeborg/capa-action-test/custom"
)

func main() {
	var m2, m3 string
	url := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
	k8sReleaseResponse, err := http.Get(url)
	custom.CheckError(err)

	min1, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	custom.CheckError(err)

	min1Release := custom.BuildReleaseVersion(string(min1))
	log.Print("Info: min1Release: Major ", min1Release.Major, ", Minor ", min1Release.Minor, ", Patch ", min1Release.Patch)

	if min1Release.Minor >= 2 {
		m2 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-1)
		m3 = strconv.Itoa(min1Release.Major) + "." + strconv.Itoa(min1Release.Minor-2)
	}

	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m2)
	k8sReleaseResponse, err = http.Get(url)
	custom.CheckError(err)

	min2, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	custom.CheckError(err)

	min2Release := custom.BuildReleaseVersion(string(min2))
	log.Print("Info: min2Release: Major ", min2Release.Major, ", Minor ", min2Release.Minor, ", Patch ", min2Release.Patch)

	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m3)
	k8sReleaseResponse, err = http.Get(url)
	custom.CheckError(err)

	min3, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	custom.CheckError(err)

	min3Release := custom.BuildReleaseVersion(string(min3))
	log.Print("Info: min3Release: Major ", min3Release.Major, ", Minor ", min3Release.Minor, ", Patch ", min3Release.Patch)

	latestAMIBuildConfig := &custom.AMIBuildConfig{
		K8sReleases: map[string]string{
			"min1": string(min1),
			"min2": string(min2),
			"min3": string(min3),
		},
	}

	latestAMIBuildConfigFileBytes, err := json.MarshalIndent(latestAMIBuildConfig, "", "  ")
	custom.CheckError(err)

	AMIBuildConfigFilename := os.Getenv("AMI_BUILD_CONFIG_FILENAME")
	dat, err := os.ReadFile(AMIBuildConfigFilename)
	if err != nil {
		if os.IsNotExist(err) {
			Action(latestAMIBuildConfigFileBytes, AMIBuildConfigFilename)
			log.Printf("Info: Created \"AMIBuildConfig.json\" K8s versions \"%s\"", latestAMIBuildConfig.K8sReleases)
			return
		} else {
			log.Fatal(err)
		}
	}

	currentAMIBuildConfig := new(custom.AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	custom.CheckError(err)
	if !cmp.Equal(currentAMIBuildConfig, latestAMIBuildConfig) {
		prCreated := Action(latestAMIBuildConfigFileBytes, AMIBuildConfigFilename)
		if prCreated {
			log.Printf("Info: Updated \"%s\" with K8s versions from \"%s\" to \"%s\"", AMIBuildConfigFilename, currentAMIBuildConfig.K8sReleases, latestAMIBuildConfig.K8sReleases)
		}
	} else {
		log.Printf("Info: \"%s\" is up-to-date.", AMIBuildConfigFilename)
	}
}
