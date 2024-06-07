export BUILD_VERSION="0.1.0"

docker buildx build --platform linux/amd64,linux/arm64 -t tanis2000/discomfort:${BUILD_VERSION} -t tanis2000/discomfort --build-arg BUILD_VERSION=${BUILD_VERSION} --push -f ./deploy/docker/Dockerfile .
