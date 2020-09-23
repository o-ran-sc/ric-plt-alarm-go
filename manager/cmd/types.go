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

package main

import (
	"sync"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
)

type AlarmManager struct {
	amHost        string
	amBaseUrl     string
	amSchemes     []string
	alertInterval int
	activeAlarms  []alarm.AlarmMessage
	alarmHistory  []alarm.AlarmMessage
	mutex         sync.Mutex
	rmrReady      bool
	postClear     bool
}

type AlertStatus string

const (
	AlertStatusActive   = "active"
	AlertStatusResolved = "resolved"
)

var Version string
var Hash string

type RicAlarmDefinitions struct {
	AlarmDefinitions []*alarm.AlarmDefinition `json:"alarmdefinitions"`
}
