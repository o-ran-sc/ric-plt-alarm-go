#!/bin/bash

#==================================================================================
#   Copyright (c) 2020 AT&T Intellectual Property.
#   Copyright (c) 2020 Nokia
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#==================================================================================

#set -eux

echo "--> build_ubuntu.sh starts"

# Install RMR from deb packages at packagecloud.io
rmr=rmr_4.8.5_amd64.deb
wget --content-disposition  https://packagecloud.io/o-ran-sc/release/packages/debian/stretch/$rmr/download.deb
sudo dpkg -i $rmr
rm $rmr
rmrdev=rmr-dev_4.8.5_amd64.deb
wget --content-disposition https://packagecloud.io/o-ran-sc/release/packages/debian/stretch/$rmrdev/download.deb
sudo dpkg -i $rmrdev
rm $rmrdev

# Required to find nng and rmr libs
export LD_LIBRARY_PATH=/usr/local/lib

# Installing the go version
GOLANG_VERSION=1.16.1
wget --quiet https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz \
        && tar xvzf go$GOLANG_VERSION.linux-amd64.tar.gz -C /usr/local
PATH="/usr/local/go/bin:${PATH}"
GOPATH="/go"
rm go$GOLANG_VERSION.linux-amd64.tar.gz


# Go install, build, etc
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH

# xApp-framework stuff
export CFG_FILE=../config/config-file.json
export RMR_SEED_RT=../config/uta_rtg.rt

# xApp stuff
export DEF_FILE=../../definitions/alarm-definition.json
export PERF_DEF_FILE=../testresources/perf-alarm-definition.json
export PERF_OBJ_FILE=../testresources/perf-alarm-object.json

GO111MODULE=on GO_ENABLED=0 GOOS=linux

# setup version tag
if [ -f build/container-tag.yaml ]
then
    tag=$(grep "tag:" build/container-tag.yaml | awk '{print $2}')
else
    tag="no-tag-found"
fi

hash=$(git rev-parse --short HEAD || true)

ROOT_DIR=$PWD

# compile the CLI
cd ${ROOT_DIR}/cli && go build -a -installsuffix cgo alarm-cli.go

# Build
cd ${ROOT_DIR}/manager && go build -a -installsuffix cgo -ldflags "-X main.Version=$tag -X main.Hash=$hash" -o alarm-manager ./cmd/*.go

# Execute UT and measure coverage for the Alarm Library
cd ${ROOT_DIR}/alarm && go test . -v -coverprofile cover.out

# Copy alarm/cover.out to alarm-go/cover.out
cd ${ROOT_DIR} && cat alarm/cover.out > coverage.out

# And for the Alarm Manager
#cd ${ROOT_DIR}/manager && go test -v -p 1 -coverprofile cover.out ./cmd/ -c -o ./manager_test && ./manager_test
cd ${ROOT_DIR}/manager && GO111MODULE=on RMR_SEED_RT=../../config/uta_rtg.rt CFG_FILE=../../config/config-file.json go test -v -p 1 -cover -coverprofile=cover.out ./...

# Remove first line of the manager/cover.out and append to alarm-go/coverity.out
cd ${ROOT_DIR} && sed '1d' manager/cover.out >> coverage.out

cd ${ROOT_DIR} && GO111MODULE=on go tool cover -html=coverage.out -o coverage.html

echo "--> build_ubuntu.sh ends"
