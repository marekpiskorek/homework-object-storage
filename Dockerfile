FROM golang:1.18
# dependency issues with moby packages forces us to bump the version from 1.15 to 1.18
WORKDIR /mnt/homework
COPY . .
RUN go build

# Docker is used as a base image so you can easily start playing around in the container using the Docker command line client.
FROM docker
COPY --from=0 /mnt/homework/homework-object-storage /usr/local/bin/homework-object-storage
RUN apk add bash curl libc6-compat
