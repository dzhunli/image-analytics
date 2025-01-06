#!/bin/bash

set -e

IMAGE_NAME=$1
CI_CONFIG=$2
ALLOW_LARGE_IMAGE=$3

check_image_size() {
    IMAGE_SIZE=$(docker image inspect "$1" --format='{{.Size}}')
    IMAGE_SIZE_GB=$(bc <<< "scale=2; $IMAGE_SIZE / 1024 / 1024 / 1024")
    echo -e -n "Image size:"
    echo -e -n "\033[1;33m ${IMAGE_SIZE_GB} \033[0m"
    echo -e "GB"

    if (( $(echo "$IMAGE_SIZE_GB > 1" | bc -l) )); then
        if [[ "$ALLOW_LARGE_IMAGE" != "true" ]]; then
	        echo -e -n "\033[1;31m Error: The image size exceeds 1 GB. \033[0m"
		echo "Pass 'allow_large_image=true' to proceed."                                    
		exit 1
        else
	        echo -e "\033[1;32m Large image allowed. Continuing... \033[0m"
	fi
    fi
}

echo "Checking Docker image size..."
check_image_size "$IMAGE_NAME"

echo "Fetching the latest Dive version..."
DIVE_VERSION=$(curl -sL "https://api.github.com/repos/wagoodman/dive/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
echo "Latest Dive version: $DIVE_VERSION"

echo "Downloading and installing Dive..."
curl -OL https://github.com/wagoodman/dive/releases/download/v${DIVE_VERSION}/dive_${DIVE_VERSION}_linux_amd64.deb
sudo apt install -qqq ./dive_${DIVE_VERSION}_linux_amd64.deb

echo "Running Dive analysis on image: $IMAGE_NAME with CI config: $CI_CONFIG"
CI=true dive --ci-config "$CI_CONFIG" "$IMAGE_NAME"

