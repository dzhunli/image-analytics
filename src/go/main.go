package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Release struct {
	TagName string `json:"tag_name"`
}

func main() {
	if len(os.Args) < 5 {
		log.Fatalf("Usage: %s <image_name> <ci_config> <allow_large_image> <continue_on_fail>", os.Args[0])
	}

	imageName := os.Args[1]
	ciConfig := os.Args[2]
	allowLargeImage := os.Args[3] == "true"
	continueOnFail := os.Args[4] == "true"

	fmt.Println("Checking Docker image size...")
	if !checkImageSize(imageName, allowLargeImage, continueOnFail) {
		return
	}

	getDive()

	fmt.Printf("Running Dive analysis on image: %s with CI config: %s\n", imageName, ciConfig)
	checkImage(imageName, ciConfig, continueOnFail)
}

func checkImageSize(imageName string, allowLargeImage, continueOnFail bool) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName, "--format={{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to inspect Docker image: %v", err)
	}

	sizeBytes, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		log.Fatalf("Failed to parse image size: %v", err)
	}

	sizeGB := sizeBytes / (1024 * 1024 * 1024)
	fmt.Printf("Image size: \033[1;33m%.2f GB\033[0m\n", sizeGB)

	if sizeGB > 1 {
		if !allowLargeImage {
			fmt.Println("\033[1;31mError: The image size exceeds 1 GB.\033[0m")
			fmt.Println("\n\n#\tPass 'allow_large_image=true' to proceed.")
			if continueOnFail {
				fmt.Println("#\tPass 'continue_on_fail=false' to fail actions that don't pass the test.")
				fmt.Println("\033[1;33mCONTINUE POLICY ENABLED...\033[0m")
				return false
			}
			os.Exit(1)
		} else {
			fmt.Println("\033[1;32mLarge image allowed. Continuing...\033[0m")
		}
	}
	return true
}

func getDive() {
	fmt.Println("::group::Fetching the latest Dive version...")
	resp, err := http.Get("https://api.github.com/repos/wagoodman/dive/releases/latest")
	if err != nil {
		log.Fatalf("Failed to fetch Dive version: %v", err)
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Fatalf("Failed to parse Dive version: %v", err)
	}

	fmt.Printf("Latest Dive version: %s\n", release.TagName)
	fmt.Println("::endgroup::")

	fmt.Println("::group::Downloading and installing Dive...")
	debianPackageURL := fmt.Sprintf("https://github.com/wagoodman/dive/releases/download/%s/dive_%s_linux_amd64.deb", release.TagName, release.TagName[1:])
	cmd := exec.Command("curl", "-OL", debianPackageURL)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to download Dive: %v", err)
	}

	cmd = exec.Command("sudo", "apt", "install", "-qqq", fmt.Sprintf("./dive_%s_linux_amd64.deb", release.TagName[1:]))
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to install Dive: %v", err)
	}
	fmt.Println("::endgroup::")
}

func checkImage(imageName, ciConfig string, continueOnFail bool) {
	cmd := exec.Command("dive", "--ci-config", ciConfig, imageName)
	if continueOnFail {
		cmd.Env = append(os.Environ(), "CI=true")
		if err := cmd.Run(); err != nil {
			fmt.Println("\033[1;33mCONTINUE POLICY ENABLED...\033[0m")
			fmt.Println("\n\n#\tPass 'continue_on_fail=false' to fail actions that don't pass the test.")
		}
	} else {
		cmd.Env = append(os.Environ(), "CI=true")
		if err := cmd.Run(); err != nil {
			log.Fatalf("Dive analysis failed: %v", err)
		}
	}
}
