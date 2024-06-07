# Build the application from source
FROM golang:latest AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /discomfort

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
RUN echo "{}" > /images.json

EXPOSE 8080

#USER nonroot:nonroot
ENV TOKEN=""
ENV COMFY_ADDRESS="localhost"
ENV COMFY_PORT=8188

ENTRYPOINT ["./run.sh"]