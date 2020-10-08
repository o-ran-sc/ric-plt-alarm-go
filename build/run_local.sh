#!/bin/sh -e
#
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
#
#
#	Mnemonic:	run.sh
#	Abstract:	Starts the alarm manager service
#	Date:		10 March 2020
#
export RMR_SEED_RT=$PWD/config/uta_rtg.rt
export RMR_SRC_ID="service-ricplt-alarmmanager-rmr.ricplt"
export DEF_FILE=$PWD/definitions/alarm-definition.json
export PERF_DEF_FILE=$PWD/cli/perf-alarm-definition.json
export PERF_OBJ_FILE=$PWD/cli/perf-alarm-object.json

$PWD/manager/alarm-manager -f $PWD/config/config-file.json
