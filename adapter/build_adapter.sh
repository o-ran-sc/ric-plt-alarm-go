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

set -e
set -x

# setup version tag
if [ -f container-tag.yaml ]
then
    tag=$(grep "tag:" container-tag.yaml | awk '{print $2}')
else
    tag="-"
fi

hash=$(git rev-parse --short HEAD || true)

export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export CFG_FILE=../config/config-file.json
export RMR_SEED_RT=../config/uta_rtg.rt

GO111MODULE=on GO_ENABLED=0 GOOS=linux

# Build
go build -a -installsuffix cgo -ldflags "-X main.Version=$tag -X main.Hash=$hash" -o alarm-adapter ./cmd/*.go

# Run UT
cd ../alarm && RMR_SEED_RT=../config/uta_rtg_lib.r go-acc ./
#go test -v -p 1 -coverprofile cover.out ./cmd/ -c -o ./adapter_test && ./adapter_test
#cd ../alarm && RMR_SEED_RT=../config/uta_rtg_lib.rt go test . -v -coverprofile cover.out
