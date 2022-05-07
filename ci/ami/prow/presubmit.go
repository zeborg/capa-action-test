package prow

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zeborg/capa-action-test/custom"
)

func Presubmit() {
	AMIBuildConfigFilename := os.Getenv("AMI_BUILD_CONFIG_FILENAME")
	AMIBuildConfigDefaultsFilename := os.Getenv("AMI_BUILD_CONFIG_DEFAULTS")

	dat, err := os.ReadFile(AMIBuildConfigFilename)
	log.Println("Debugging: Presubmit: L18")
	custom.CheckError(err)
	currentAMIBuildConfig := new(custom.AMIBuildConfig)
	err = json.Unmarshal(dat, currentAMIBuildConfig)
	log.Println("Debugging: Presubmit: L22")
	custom.CheckError(err)

	dat, err = os.ReadFile(AMIBuildConfigDefaultsFilename)
	log.Println("Debugging: Presubmit: L26")
	custom.CheckError(err)
	defaultAMIBuildConfig := new(custom.AMIBuildConfigDefaults)
	err = json.Unmarshal(dat, defaultAMIBuildConfig)
	log.Println("Debugging: Presubmit: L30")
	custom.CheckError(err)

	for _, v := range currentAMIBuildConfig.K8sReleases {
		stderr, stdout, err := custom.Shell(fmt.Sprintf("./clusterawsadm ami list --kubernetes-version %s --owner-id %s", strings.TrimPrefix(v, "v"), os.Getenv("AWS_AMI_OWNER_ID")))
		log.Println("Debugging: Presubmit: L35")
		custom.CheckError(err)

		if stderr != "" {
			log.Fatalf("Error: %s", stderr)
		} else if stdout == "" {
			log.Printf("Info: Building AMI for Kubernetes %s.", v)
			ami_regions := os.Getenv("AMI_BUILD_REGIONS")
			kubernetes_semver := v
			kubernetes_rpm_version := strings.TrimPrefix(v, "v") + "-0"
			kubernetes_deb_version := strings.TrimPrefix(v, "v") + "-00"
			kubernetes_series := strings.Split(v, ".")[0] + "." + strings.Split(v, ".")[1]

			flagsK8s := fmt.Sprintf("-var=ami_regions=%s -var=kubernetes_series=%s -var=kubernetes_semver=%s -var=kubernetes_rpm_version=%s -var=kubernetes_deb_version=%s ", ami_regions, kubernetes_series, kubernetes_semver, kubernetes_rpm_version, kubernetes_deb_version)
			for k, v := range defaultAMIBuildConfig.Default {
				flagsK8s += fmt.Sprintf("-var=%s=%s ", k, v)
			}

			supportedOS := strings.Split(os.Getenv("AMI_BUILD_SUPPORTED_OS"), ",")

			for _, os := range supportedOS {
				switch os {
				case "amazon-2":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Amazon2 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stderr, stdout, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s && cd ../../..", flags, os))
					custom.CheckError(err)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "centos-7":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Centos7 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stderr, stdout, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s && cd ../../..", flags, os))
					custom.CheckError(err)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "flatcar":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Flatcar {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stderr, stdout, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s && cd ../../..", flags, os))
					custom.CheckError(err)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "ubuntu-1804":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Ubuntu1804 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stderr, stdout, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s && cd ../../..", flags, os))
					custom.CheckError(err)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
					}
				case "ubuntu-2004":
					flags := flagsK8s
					for k, v := range defaultAMIBuildConfig.Ubuntu2004 {
						flags += fmt.Sprintf("-var=%s=%s ", k, v)
					}

					log.Println(fmt.Sprintf("Info: Building AMI for OS %s", os))
					log.Println(fmt.Sprintf("Info: flags:  \"%s\"", flags))

					stderr, stdout, err := custom.Shell(fmt.Sprintf("cd image-builder/images/capi && PACKER_FLAGS=\"%s\" make build-ami-%s && cd ../../..", flags, os))
					custom.CheckError(err)
					if stderr != "" {
						log.Fatalf("Error: %s", stderr)
					} else {
						log.Println(stdout)
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
