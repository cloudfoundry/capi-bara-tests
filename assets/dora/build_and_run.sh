#!/bin/bash

docker build -t cfcapidocker/dora:stretch -f Dockerfile_stretch .
docker build -t cfcapidocker/dora:alpine -f Dockerfile_alpine .

docker rm -f dora_stretch_docker
docker rm -f dora_alpine_docker

docker run -d -p 8081:8080  --name dora_stretch_docker dora_stretch
docker run -d -p 8080:8080  --name dora_alpine_docker dora_alpine