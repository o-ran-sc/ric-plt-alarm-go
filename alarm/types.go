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
	"fmt"
	"os"
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
	IdentifyingInfo   string   `json:"identifyingInfo"`
	AdditionalInfo    string   `json:"additionalInfo"`
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
	E2_CONNECTION_PROBLEM              int = 72004
	ACTIVE_ALARM_EXCEED_MAX_THRESHOLD  int = 72007
	ALARM_HISTORY_EXCEED_MAX_THRESHOLD int = 72008
)

type AlarmDefinition struct {
	AlarmId               int    `json:"alarmId"`
	AlarmText             string `json:"alarmText"`
	EventType             string `json:"eventType"`
	OperationInstructions string `json:"operationInstructions"`
	RaiseDelay            int    `json:"raiseDelay"`
	ClearDelay            int    `json:"clearDelay"`
	TimeToLive            int    `json:"timeToLive"`
}

var RICAlarmDefinitions map[int]*AlarmDefinition
var RICPerfAlarmObjects map[int]*Alarm

var (
	namespace                     = os.Getenv("PLT_NAMESPACE")
	ALARM_MANAGER_HTTP_URL string = fmt.Sprintf("http://service-%s-alarmmanager-http.%s:8080", namespace, namespace)
	ALARM_MANAGER_RMR_URL  string = fmt.Sprintf("service-%s-alarmmanager-rmr.%s:4560", namespace, namespace)
)
