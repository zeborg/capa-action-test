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

type AMIBuildConfigDefaults struct {
	Amazon2    map[string]string `json:"amazon-2"`
	Centos7    map[string]string `json:"centos-7"`
	Flatcar    map[string]string `json:"flatcar"`
	Ubuntu1804 map[string]string `json:"ubuntu-1804"`
	Ubuntu2004 map[string]string `json:"ubuntu-2004"`
	Default    map[string]string `json:"default"`
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
	url := "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
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

	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m2)
	k8sReleaseResponse, err = http.Get(url)
	checkError(err)

	min2, err := ioutil.ReadAll(k8sReleaseResponse.Body)
	checkError(err)

	min2Release := BuildReleaseVersion(string(min2))
	log.Print("Info: min2Release: Major ", min2Release.Major, ", Minor ", min2Release.Minor, ", Patch ", min2Release.Patch)

	url = fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%s.txt", m3)
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
	AMIBuildConfigDefaultsFilename := os.Getenv("AMI_BUILD_CONFIG_DEFAULTS")

	dat, err := os.ReadFile(AMIBuildConfigFilename)
	checkError(err)
	currentAMIBuildConfig := new(AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	checkError(err)

	dat, err = os.ReadFile(AMIBuildConfigDefaultsFilename)
	checkError(err)
	defaultAMIBuildConfig := new(AMIBuildConfigDefaults)
	err = json.Unmarshal(dat, defaultAMIBuildConfig)
	checkError(err)

	for _, v := range currentAMIBuildConfig.K8sReleases {
		err, out, _ := Shell(fmt.Sprintf("./clusterawsadm ami list --kubernetes-version %s", strings.TrimPrefix(v, "v")))
		checkError(err)

		if out == "" {
			log.Printf("Info: Building AMI for Kubernetes %s.", v)
			kubernetes_semver := v
			kubernetes_rpm_version := strings.TrimPrefix(v, "v") + "-0"
			kubernetes_deb_version := strings.TrimPrefix(v, "v") + "-00"
			kubernetes_series := strings.Split(v, ".")[0] + strings.Split(v, ".")[1]

			flagsK8s := fmt.Sprintf("-var=kubernetes_series=%s -var=kubernetes_semver=%s -var=kubernetes_rpm_version=%s -var=kubernetes_deb_version=%s ", kubernetes_series, kubernetes_semver, kubernetes_rpm_version, kubernetes_deb_version)
			for k, v := range defaultAMIBuildConfig.Default {
				flagsK8s += fmt.Sprintf("-var=%s=%s ", k, v)
			}

			for _, os := range []string{"amazon-2", "centos-7", "flatcar-stable", "ubuntu-18.04", "ubuntu-20.04"} {
				switch os {
				case "amazon-2":
					flags := ""
					for k, v := range defaultAMIBuildConfig.Amazon2 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}
					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					err, out, errout := Shell(fmt.Sprintf("PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					checkError(err)
					if errout != "" {
						log.Fatalf("Error: %s", errout)
					} else {
						log.Println(out)
					}
				case "centos-7":
					flags := ""
					for k, v := range defaultAMIBuildConfig.Centos7 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}
					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					err, out, errout := Shell(fmt.Sprintf("PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					checkError(err)
					if errout != "" {
						log.Fatalf("Error: %s", errout)
					} else {
						log.Println(out)
					}
				case "flatcar":
					flags := ""
					for k, v := range defaultAMIBuildConfig.Flatcar {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}
					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					err, out, errout := Shell(fmt.Sprintf("PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					checkError(err)
					if errout != "" {
						log.Fatalf("Error: %s", errout)
					} else {
						log.Println(out)
					}
				case "ubuntu-1804":
					flags := ""
					for k, v := range defaultAMIBuildConfig.Ubuntu1804 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}
					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					err, out, errout := Shell(fmt.Sprintf("PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					checkError(err)
					if errout != "" {
						log.Fatalf("Error: %s", errout)
					} else {
						log.Println(out)
					}
				case "ubuntu-2004":
					flags := ""
					for k, v := range defaultAMIBuildConfig.Ubuntu2004 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}
					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					err, out, errout := Shell(fmt.Sprintf("PACKER_FLAGS=\"%s\" make build-ami-%s", flags, os))
					checkError(err)
					if errout != "" {
						log.Fatalf("Error: %s", errout)
					} else {
						log.Println(out)
					}
				default:
					log.Println(fmt.Sprintf("Warning: Invalid OS %s. Skipping image building.", os))
				}
			}
		} else {
			log.Printf("Info: AMI for Kubernetes %s already exists. Skipping image building.", v)
		}
	}
}

func postsubmit() {
	log.Println("postsubmit() ")
}
