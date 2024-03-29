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
#
#   This source code is part of the near-RT RIC (RAN Intelligent Controller)
#   platform project (RICP).
#==================================================================================

FROM nexus3.o-ran-sc.org:10002/o-ran-sc/bldr-ubuntu20-c-go:1.0.0 as ubuntu-alarm-manager

RUN apt update && apt install --reinstall -y ca-certificates && update-ca-certificates

# Install utilities
RUN apt update && apt install -y iputils-ping net-tools curl sudo build-essential

# Installing the go version
ENV GOLANG_VERSION 1.20.4
RUN wget --quiet https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz \
        && tar xvzf go$GOLANG_VERSION.linux-amd64.tar.gz -C /usr/local
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# Install dependencies, compile and test the module
RUN mkdir -p /go/src/am
RUN mkdir -p /cfg
COPY . /go/src/am

WORKDIR "/go/src/am"
RUN ./build/build_ubuntu.sh

# Final, executable and deployable container
FROM ubuntu:20.04

COPY --from=ubuntu-alarm-manager /go/src/am/build/run.sh /
COPY --from=ubuntu-alarm-manager /go/src/am/manager/alarm-manager /
COPY --from=ubuntu-alarm-manager /go/src/am/cli/alarm-cli /
COPY --from=ubuntu-alarm-manager /go/src/am/testresources/perf-alarm-object.json /
COPY --from=ubuntu-alarm-manager /go/src/am/testresources/perf-alarm-definition.json /
COPY --from=ubuntu-alarm-manager /go/src/am/config/* /cfg/
COPY --from=ubuntu-alarm-manager /usr/local/lib /usr/local/lib
COPY --from=ubuntu-alarm-manager /go/src/am/definitions/* / 

RUN ldconfig
ENV DEF_FILE="./alarm-definition.json"
ENV PERF_DEF_FILE="./perf-alarm-definition.json"
ENV PERF_OBJ_FILE="./perf-alarm-object.json"

RUN chmod 755 /run.sh
CMD /run.sh
