BUILD_VERSION=$(git describe --tags HEAD)
export BUILD_VERSION
IMAGE_VERSION=${BUILD_VERSION#"v"}
export IMAGE_VERSION

docker buildx build --platform linux/amd64,linux/arm64 -t tanis2000/discomfort:"${IMAGE_VERSION}" -t tanis2000/discomfort --build-arg BUILD_VERSION="${BUILD_VERSION}" --push -f ./deploy/docker/Dockerfile .
