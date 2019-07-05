#!/bin/bash

echo "Building Go binary"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xlsx2json-api ..

echo "Building Docker image"
docker build -q -t xlsx2json-api .
