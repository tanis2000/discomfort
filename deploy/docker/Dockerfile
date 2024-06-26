# Build the application from source
FROM golang:latest AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV BUILD_VERSION=dev
ARG BUILD_VERSION=${BUILD_VERSION}
RUN CGO_ENABLED=1 go build -v -ldflags="-X 'discomfort/internal/build.Version=${BUILD_VERSION}' -X 'discomfort/internal/build.User=$(id -u -n)' -X 'discomfort/internal/build.Time=$(date)'" -o /discomfort

# Run the tests in the container
FROM build-stage AS run-test-stage
RUN go test -v ./...

# Deploy the application binary into a lean image
FROM debian:stable-slim AS build-release-stage
RUN apt-get update -y
RUN apt-get install -y ca-certificates

WORKDIR /

COPY --from=build-stage /discomfort /discomfort
COPY --chmod=777 --from=build-stage /app/deploy/docker/run.sh /run.sh
COPY --from=build-stage /app/workflows /workflows
COPY --from=build-stage /app/docs /docs
RUN mkdir /data

ENV TOKEN=""
ENV COMFY_ADDRESS="localhost"
ENV COMFY_PORT=8188

ENTRYPOINT ["./run.sh"]