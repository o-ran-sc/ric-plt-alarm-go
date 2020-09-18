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
	RIC_RT_DISTRIBUTION_FAILED     int = 8004
	TCP_CONNECTIVITY_LOST_TO_DBAAS int = 8005
	E2_CONNECTIVITY_LOST_TO_GNODEB int = 8006
	E2_CONNECTIVITY_LOST_TO_ENODEB int = 8007
)

type AlarmDefinition struct {
	AlarmId               int
	AlarmText             string
	EventType             string
	OperationInstructions string
}

var RICAlarmDefinitions = map[int]AlarmDefinition{
	RIC_RT_DISTRIBUTION_FAILED: {
		AlarmId:               RIC_RT_DISTRIBUTION_FAILED,
		AlarmText:             "RIC ROUTING TABLE DISTRIBUTION FAILED",
		EventType:             "Processing error",
		OperationInstructions: "Not defined",
	},
	TCP_CONNECTIVITY_LOST_TO_DBAAS: {
		AlarmId:               TCP_CONNECTIVITY_LOST_TO_DBAAS,
		AlarmText:             "TCP CONNECTIVITY LOST TO DBAAS",
		EventType:             "Communication error",
		OperationInstructions: "Not defined",
	},
	E2_CONNECTIVITY_LOST_TO_GNODEB: {
		AlarmId:               E2_CONNECTIVITY_LOST_TO_GNODEB,
		AlarmText:             "E2 CONNECTIVITY LOST TO G-NODEB",
		EventType:             "Communication error",
		OperationInstructions: "Not defined",
	},
	E2_CONNECTIVITY_LOST_TO_ENODEB: {
		AlarmId:               E2_CONNECTIVITY_LOST_TO_ENODEB,
		AlarmText:             "E2 CONNECTIVITY LOST TO E-NODEB",
		EventType:             "Communication error",
		OperationInstructions: "Not defined",
	},
}

const (
	ALARM_MANAGER_HTTP_URL string = "http://service-ricplt-alarmmanager-http.ricplt:8080"
	ALARM_MANAGER_RMR_URL  string = "service-ricplt-alarmmanager-rmr.ricplt:4560"
)