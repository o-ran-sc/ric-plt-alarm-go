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
)

func (a *AlarmAdapter) GetActiveAlarms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response, _ := json.Marshal(a.activeAlarms)
	w.Write(response)
}

func (a *AlarmAdapter) GenerateAlarm(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		return
	}
	defer r.Body.Close()

	var alarmData alarm.Alarm
	if err := json.NewDecoder(r.Body).Decode(&alarmData); err == nil {
		a.UpdateActiveAlarms(alarmData)
		a.PostAlert(a.GenerateAlertLabels(alarmData))
	}
}
