/*
 *  Copyright (c) 2020 AT&T Intellectual Property.
 *  Copyright (c) 2020 Nokia.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 * This source code is part of the near-RT RIC (RAN Intelligent Controller)
 * platform project (RICP).
 */

package alarm

import (
	"sync"
	"unsafe"
)

import "C"

// Severity for alarms
type Severity string

// Possible values for Severity
const (
	SeverityUnspecified Severity = "UNSPECIFIED"
	SeverityCritical    Severity = "CRITICAL"
	SeverityMajor       Severity = "MAJOR"
	SeverityMinor       Severity = "MINOR"
	SeverityWarning     Severity = "WARNING"
	SeverityCleared     Severity = "CLEARED"
	SeverityDefault     Severity = "DEFAULT"
)

// Alarm object - see README for more information
type Alarm struct {
	ManagedObjectId   string   `json:"managedObjectId"`
	ApplicationId     string   `json:"applicationId"`
	SpecificProblem   int      `json:"specificProblem"`
	PerceivedSeverity Severity `json:"perceivedSeverity"`
	AdditionalInfo    string   `json:"additionalInfo"`
	IdentifyingInfo   string   `json:"identifyingInfo"`
}

// Alarm actions
type AlarmAction string

// Possible values for alarm actions
const (
	AlarmActionRaise    AlarmAction = "RAISE"
	AlarmActionClear    AlarmAction = "CLEAR"
	AlarmActionClearAll AlarmAction = "CLEARALL"
)

type AlarmMessage struct {
	Alarm
	AlarmAction
	AlarmTime int64
}

type AlarmConfigParams struct {
	MaxActiveAlarms int `json:"maxactivealarms"`
	MaxAlarmHistory int `json:"maxalarmhistory"`
}

// RICAlarm is an alarm instance
type RICAlarm struct {
	moId       string
	appId      string
	managerUrl string
	rmrCtx     unsafe.Pointer
	rmrReady   bool
	mutex      sync.Mutex
}

const (
	RIC_ALARM_UPDATE = 13111
	RIC_ALARM_QUERY  = 13112
)

// Temp alarm constants & definitions
const (
	PERFORMANCE_TEST_ALARM_1           int = 1001
	PERFORMANCE_TEST_ALARM_2           int = 1002
	PERFORMANCE_TEST_ALARM_3           int = 1003
	PERFORMANCE_TEST_ALARM_4           int = 1004
	PERFORMANCE_TEST_ALARM_5           int = 1005
	PERFORMANCE_TEST_ALARM_6           int = 1006
	PERFORMANCE_TEST_ALARM_7           int = 1007
	PERFORMANCE_TEST_ALARM_8           int = 1008
	PERFORMANCE_TEST_ALARM_9           int = 1009
	PERFORMANCE_TEST_ALARM_10          int = 1010
	PERFORMANCE_TEST_ALARM_11          int = 1011
	PERFORMANCE_TEST_ALARM_12          int = 1012
	PERFORMANCE_TEST_ALARM_13          int = 1013
	PERFORMANCE_TEST_ALARM_14          int = 1014
	PERFORMANCE_TEST_ALARM_15          int = 1015
	PERFORMANCE_TEST_ALARM_16          int = 1016
	PERFORMANCE_TEST_ALARM_17          int = 1017
	PERFORMANCE_TEST_ALARM_18          int = 1018
	PERFORMANCE_TEST_ALARM_19          int = 1019
	PERFORMANCE_TEST_ALARM_20          int = 1020
	PERFORMANCE_TEST_ALARM_21          int = 1021
	PERFORMANCE_TEST_ALARM_22          int = 1022
	PERFORMANCE_TEST_ALARM_23          int = 1023
	PERFORMANCE_TEST_ALARM_24          int = 1024
	PERFORMANCE_TEST_ALARM_25          int = 1025
	PERFORMANCE_TEST_ALARM_26          int = 1026
	PERFORMANCE_TEST_ALARM_27          int = 1027
	PERFORMANCE_TEST_ALARM_28          int = 1028
	PERFORMANCE_TEST_ALARM_29          int = 1029
	PERFORMANCE_TEST_ALARM_30          int = 1030
	PERFORMANCE_TEST_ALARM_31          int = 1031
	PERFORMANCE_TEST_ALARM_32          int = 1032
	PERFORMANCE_TEST_ALARM_33          int = 1033
	PERFORMANCE_TEST_ALARM_34          int = 1034
	PERFORMANCE_TEST_ALARM_35          int = 1035
	PERFORMANCE_TEST_ALARM_36          int = 1036
	PERFORMANCE_TEST_ALARM_37          int = 1037
	PERFORMANCE_TEST_ALARM_38          int = 1038
	PERFORMANCE_TEST_ALARM_39          int = 1039
	PERFORMANCE_TEST_ALARM_40          int = 1040
	PERFORMANCE_TEST_ALARM_41          int = 1041
	PERFORMANCE_TEST_ALARM_42          int = 1042
	PERFORMANCE_TEST_ALARM_43          int = 1043
	PERFORMANCE_TEST_ALARM_44          int = 1044
	PERFORMANCE_TEST_ALARM_45          int = 1045
	PERFORMANCE_TEST_ALARM_46          int = 1046
	PERFORMANCE_TEST_ALARM_47          int = 1047
	PERFORMANCE_TEST_ALARM_48          int = 1048
	PERFORMANCE_TEST_ALARM_49          int = 1049
	PERFORMANCE_TEST_ALARM_50          int = 1050
	RIC_RT_DISTRIBUTION_FAILED         int = 8004
	TCP_CONNECTIVITY_LOST_TO_DBAAS     int = 8005
	E2_CONNECTIVITY_LOST_TO_GNODEB     int = 8006
	E2_CONNECTIVITY_LOST_TO_ENODEB     int = 8007
	ACTIVE_ALARM_EXCEED_MAX_THRESHOLD  int = 8008
	ALARM_HISTORY_EXCEED_MAX_THRESHOLD int = 8009
)

type AlarmDefinition struct {
	AlarmId               int    `json:"alarmid"`
	AlarmText             string `json:"alarmtext"`
	EventType             string `json:"eventtype"`
	OperationInstructions string `json:"operationinstructions"`
}

var RICAlarmDefinitions map[int]*AlarmDefinition
var RICPerfAlarmObjects map[int]*Alarm

const (
	ALARM_MANAGER_HTTP_URL string = "http://service-ricplt-alarmmanager-http.ricplt:8080"
	ALARM_MANAGER_RMR_URL  string = "service-ricplt-alarmmanager-rmr.ricplt:4560"
)
