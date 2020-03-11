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
	"encoding/json"
	"net/http"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	app "gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
)

var alarmClient *alarm.RICAlarm

func (a *AlarmAdapter) GetActiveAlarms(w http.ResponseWriter, r *http.Request) {
	app.Logger.Info("GetActiveAlarms: request received!")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response, _ := json.Marshal(a.activeAlarms)
	w.Write(response)
}

func (a *AlarmAdapter) RaiseAlarm(w http.ResponseWriter, r *http.Request) {
	a.doAction(w, r, true)
}

func (a *AlarmAdapter) ClearAlarm(w http.ResponseWriter, r *http.Request) {
	a.doAction(w, r, false)
}

func (a *AlarmAdapter) doAction(w http.ResponseWriter, r *http.Request, raiseAlarm bool) {
	app.Logger.Info("doAction: request received!")

	if r.Body == nil {
		return
	}
	defer r.Body.Close()

	var d alarm.Alarm
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		app.Logger.Error("json.NewDecoder failed: %v", err)
		return
	}

	if alarmClient == nil {
		alarmClient, err = alarm.InitAlarm("RIC", "UEEC")
		if err != nil {
			app.Logger.Error("json.NewDecoder failed: %v", err)
			return
		}
	}

	alarmData := alarmClient.NewAlarm(d.SpecificProblem, d.PerceivedSeverity, d.AdditionalInfo, d.IdentifyingInfo)
	if raiseAlarm {
		alarmClient.Raise(alarmData)
	} else {
		alarmClient.Clear(alarmData)
	}
}
