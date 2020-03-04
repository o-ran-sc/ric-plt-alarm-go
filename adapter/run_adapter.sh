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
#	Mnemonic:	run_adapter.sh
#	Abstract:	Starts the alarm adapter service
#	Date:		10 March 2020
#
export RMR_SEED_RT=/opt/nokia/ric/ueec/uta_rtg.rt
export RMR_SRC_ID="service-ricplt-alarmadapter-rmr.ricplt"

exec ./alarm-adapter -f config-file.json
