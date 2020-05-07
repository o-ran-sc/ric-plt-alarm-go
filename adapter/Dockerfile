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

FROM nexus3.o-ran-sc.org:10004/bldr-ubuntu18-c-go:5-u18.04-nng as ubuntu-alarmadapter

# Install utilities
RUN apt update && apt install -y iputils-ping net-tools curl sudo

# Install dependencies, compile and test the module
RUN mkdir -p /go/src/alarm-adapter
COPY . /go/src/alarm-adapter

WORKDIR "/go/src/alarm-adapter/adapter"
RUN ./build_adapter_ubuntu.sh

# Final, executable and deployable container
FROM ubuntu:18.04

COPY --from=ubuntu-alarmadapter /go/src/alarm-adapter/adapter/run_adapter.sh /
COPY --from=ubuntu-alarmadapter /go/src/alarm-adapter/adapter/alarm-adapter /
COPY --from=ubuntu-alarmadapter /go/src/alarm-adapter/config/* /
COPY --from=ubuntu-alarmadapter /usr/local/lib /usr/local/lib

RUN ldconfig

RUN chmod 755 /run_adapter.sh
CMD /run_adapter.sh
