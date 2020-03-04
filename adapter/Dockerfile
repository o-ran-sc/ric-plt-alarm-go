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

# The CI system creates and publishes the alarm-adapter Docker image
# from the last step in this multi-stage build and applies
# a Docker tag from the string in file container-tag.yaml

#FROM golang:1.12.1 as alarmadapter
FROM nexus3.o-ran-sc.org:10004/bldr-ubuntu18-c-go:3-u18.04-nng as alarmadapter

RUN apt update && apt install -y iputils-ping net-tools curl

# Install RMr shared library & development header files
ARG RMRVERSION=3.2.4
RUN wget --content-disposition https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/rmr_${RMRVERSION}_amd64.deb/download.deb && dpkg -i rmr_${RMRVERSION}_amd64.deb && rm -rf rmr_${RMRVERSION}_amd64.deb
RUN wget --content-disposition https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/rmr-dev_${RMRVERSION}_amd64.deb/download.deb && dpkg -i rmr-dev_${RMRVERSION}_amd64.deb && rm -rf rmr-dev_${RMRVERSION}_amd64.deb

ENV GOLANG_VERSION 1.12.1
RUN wget --quiet https://dl.google.com/go/go$GOLANG_VERSION.linux-amd64.tar.gz \
     && tar xvzf go$GOLANG_VERSION.linux-amd64.tar.gz -C /usr/local
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH /go

RUN mkdir -p /go/bin
RUN cd /go/bin \
    && wget --quiet https://github.com/go-swagger/go-swagger/releases/download/v0.19.0/swagger_linux_amd64 \
    && mv swagger_linux_amd64 swagger \
    && chmod +x swagger

RUN mkdir -p /go/src/alarm-adapter
WORKDIR "/go/src/alarm-adapter"
COPY go.sum .
COPY go.mod .

RUN go mod download

COPY . /go/src/alarm-adapter
WORKDIR "/go/src/alarm-adapter/adapter"

# build and run UT
RUN ldconfig
RUN ./build_adapter.sh

# Final, executable container
FROM ubuntu:16.04

COPY --from=alarmadapter /go/src/alarm-adapter/adapter/run_adapter.sh /
COPY --from=alarmadapter /go/src/alarm-adapter/adapter/alarm-adapter /
COPY --from=alarmadapter /go/src/alarm-adapter/config/* /
COPY --from=alarmadapter /usr/local/include /usr/local/include
COPY --from=alarmadapter /usr/local/lib /usr/local/lib

RUN ldconfig

RUN chmod 755 /run_adapter.sh
CMD /run_adapter.sh
