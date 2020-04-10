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

set -eux

echo "--> build_adapter_ubuntu.sh starts"

# Install RMR from deb packages at packagecloud.io
rmr=rmr_3.7.2_amd64.deb
wget --content-disposition  https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/$rmr/download.deb
sudo dpkg -i $rmr
rm $rmr
rmrdev=rmr-dev_3.7.2_amd64.deb
wget --content-disposition https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/$rmrdev/download.deb
sudo dpkg -i $rmrdev
rm $rmrdev

# Required to find nng and rmr libs
export LD_LIBRARY_PATH=/usr/local/lib

# Go install, build, etc
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH

# xApp-framework stuff
export CFG_FILE=../config/config-file.json
export RMR_SEED_RT=../config/uta_rtg.rt

GO111MODULE=on GO_ENABLED=0 GOOS=linux

# setup version tag
if [ -f container-tag.yaml ]
then
    tag=$(grep "tag:" container-tag.yaml | awk '{print $2}')
else
    tag="no-tag-found"
fi

hash=$(git rev-parse --short HEAD || true)

# Build
go build -a -installsuffix cgo -ldflags "-X main.Version=$tag -X main.Hash=$hash" -o alarm-adapter ./cmd/*.go

# Execute UT and measure coverage for the Alarm Library
cd ../alarm && go test . -v -coverprofile cover.out

# And for the Alarm Adapter
cd ../adapter && go test -v -p 1 -coverprofile cover.out ./cmd/ -c -o ./adapter_test && ./adapter_test

echo "--> build_adapter_ubuntu.sh ends"
