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
	"strconv"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go.git/alarm"
	app "gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"github.com/gorilla/mux"
)

func (a *AlarmManager) InjectRoutes() {
	app.Resource.InjectRoute("/ric/v1/alarms", a.RaiseAlarm, "POST")
	app.Resource.InjectRoute("/ric/v1/alarms", a.ClearAlarm, "DELETE")
	app.Resource.InjectRoute("/ric/v1/alarms/active", a.GetActiveAlarms, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms/history", a.GetAlarmHistory, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms/config", a.SetAlarmConfig, "POST")
	app.Resource.InjectRoute("/ric/v1/alarms/config", a.GetAlarmConfig, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms/define", a.SetAlarmDefinition, "POST")
	app.Resource.InjectRoute("/ric/v1/alarms/define/{alarmId}", a.DeleteAlarmDefinition, "DELETE")
	app.Resource.InjectRoute("/ric/v1/alarms/define", a.GetAlarmDefinition, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms/define/{alarmId}", a.GetAlarmDefinition, "GET")

	app.Resource.InjectRoute("/ric/v1/symptomdata", a.SymptomDataHandler, "GET")
}

func (a *AlarmManager) respondWithError(w http.ResponseWriter, code int, message string) {
	a.respondWithJSON(w, code, map[string]string{"error": message})
}

func (a *AlarmManager) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		response, _ := json.Marshal(payload)
		w.Write(response)
	}
}

func (a *AlarmManager) GetActiveAlarms(w http.ResponseWriter, r *http.Request) {
	app.Logger.Info("GetActiveAlarms: %+v", a.activeAlarms)
	a.respondWithJSON(w, http.StatusOK, a.activeAlarms)
}

func (a *AlarmManager) GetAlarmHistory(w http.ResponseWriter, r *http.Request) {
	app.Logger.Info("GetAlarmHistory: %+v", a.alarmHistory)
	a.respondWithJSON(w, http.StatusOK, a.alarmHistory)
}

func (a *AlarmManager) RaiseAlarm(w http.ResponseWriter, r *http.Request) {
	if err := a.doAction(w, r, true); err != nil {
		a.respondWithJSON(w, http.StatusOK, err)
	}
}

func (a *AlarmManager) ClearAlarm(w http.ResponseWriter, r *http.Request) {
	if err := a.doAction(w, r, false); err != nil {
		a.respondWithJSON(w, http.StatusOK, err)
	}
}

func (a *AlarmManager) SetAlarmDefinition(w http.ResponseWriter, r *http.Request) {

	app.Logger.Debug("POST arrived for creating alarm definition ")
	/* If body is nil then return error */
	if r.Body == nil {
		app.Logger.Error("POST - body is empty")
		a.respondWithError(w, http.StatusBadRequest, "No data in request body.")
		return
	}
	defer r.Body.Close()

	/* Parameters are available. Check if they are valid */
	var alarmDefinitions RicAlarmDefinitions
	err := json.NewDecoder(r.Body).Decode(&alarmDefinitions)
	if err != nil {
		app.Logger.Error("POST - received alarm definition  parameters are invalid - " + err.Error())
		a.respondWithError(w, http.StatusBadRequest, "Invalid data in request body.")
		return
	}

	for _, alarmDefinition := range alarmDefinitions.AlarmDefinitions {
		_, exists := alarm.RICAlarmDefinitions[alarmDefinition.AlarmId]
		if exists {
			app.Logger.Error("POST - alarm definition already exists for %v", alarmDefinition.AlarmId)
		} else {
			ricAlarmDefintion := new(alarm.AlarmDefinition)
			ricAlarmDefintion.AlarmId = alarmDefinition.AlarmId
			ricAlarmDefintion.AlarmText = alarmDefinition.AlarmText
			ricAlarmDefintion.EventType = alarmDefinition.EventType
			ricAlarmDefintion.OperationInstructions = alarmDefinition.OperationInstructions
			ricAlarmDefintion.RaiseDelay = alarmDefinition.RaiseDelay
			ricAlarmDefintion.ClearDelay = alarmDefinition.ClearDelay
			ricAlarmDefintion.TimeToLive = alarmDefinition.TimeToLive
			alarm.RICAlarmDefinitions[alarmDefinition.AlarmId] = ricAlarmDefintion
			app.Logger.Debug("POST - alarm definition added for alarm id %v", alarmDefinition.AlarmId)
		}
	}

	a.respondWithJSON(w, http.StatusOK, nil)
	return
}

func (a *AlarmManager) DeleteAlarmDefinition(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	alarmId, alarmIdok := pathParams["alarmId"]
	if alarmIdok {
		if ialarmId, err := strconv.Atoi(alarmId); err == nil {
			delete(alarm.RICAlarmDefinitions, ialarmId)
			app.Logger.Debug("DELETE - alarm definition deleted for alarmId %v", ialarmId)
		} else {
			app.Logger.Error("DELETE - alarmId string to int conversion failed %v", alarmId)
			a.respondWithError(w, http.StatusBadRequest, "Invalid path parameter")
			return
		}
	} else {
		app.Logger.Error("DELETE - alarmId does not exist %v", alarmId)
		a.respondWithError(w, http.StatusBadRequest, "Invalid path parameter")
		return

	}
}

func (a *AlarmManager) GetAlarmDefinition(w http.ResponseWriter, r *http.Request) {
	var ricAlarmDefinitions RicAlarmDefinitions
	pathParams := mux.Vars(r)
	alarmId, alarmIdok := pathParams["alarmId"]
	if alarmIdok {
		if ialarmId, err := strconv.Atoi(alarmId); err == nil {
			alarmDefinition, ok := alarm.RICAlarmDefinitions[ialarmId]
			if ok {
				app.Logger.Debug("Successfully returned alarm defintion for alarm id %v", ialarmId)
				a.respondWithJSON(w, http.StatusOK, alarmDefinition)
				return

			} else {
				app.Logger.Error("Requested alarm id not found %v", ialarmId)
				a.respondWithError(w, http.StatusBadRequest, "Non existent alarmId")
				return
			}
		} else {
			app.Logger.Error("alarmId string to int conversion failed %v", alarmId)
			a.respondWithError(w, http.StatusBadRequest, "Invalid alarmId")
			return
		}
	} else {
		app.Logger.Debug("GET arrived for all alarm definitions ")
		for _, alarmDefinition := range alarm.RICAlarmDefinitions {
			ricAlarmDefinitions.AlarmDefinitions = append(ricAlarmDefinitions.AlarmDefinitions, alarmDefinition)
		}
		app.Logger.Debug("Successfully returned all alarm definitions")
		a.respondWithJSON(w, http.StatusOK, ricAlarmDefinitions)
	}
}

func (a *AlarmManager) doAction(w http.ResponseWriter, r *http.Request, isRaiseAlarm bool) error {
	app.Logger.Info("doAction: request received = %t", isRaiseAlarm)

	if r.Body == nil {
		app.Logger.Error("Error: Invalid message body!")
		return nil
	}
	defer r.Body.Close()

	var m alarm.AlarmMessage
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		app.Logger.Error("json.NewDecoder failed: %v", err)
		return err
	}

	if m.Alarm.ManagedObjectId == "" || m.Alarm.ApplicationId == "" || m.AlarmAction == "" {
		app.Logger.Error("Error: Mandatory parameters missing!")
		return nil
	}

	if m.AlarmTime == 0 {
		m.AlarmTime = time.Now().UnixNano()
	}

	_, err := a.ProcessAlarm(&AlarmNotification{m, alarm.AlarmDefinition{}})
	return err
}

// HandleViaRmr is defined but not used as of now
/*
func (a *AlarmManager) HandleViaRmr(d alarm.Alarm, isRaiseAlarm bool) error {
	alarmClient, err := alarm.InitAlarm(d.ManagedObjectId, d.ApplicationId)
	if err != nil {
		app.Logger.Error("json.NewDecoder failed: %v", err)
		return err
	}

	alarmData := alarmClient.NewAlarm(d.SpecificProblem, d.PerceivedSeverity, d.AdditionalInfo, d.IdentifyingInfo)
	if isRaiseAlarm {
		alarmClient.Raise(alarmData)
	} else {
		alarmClient.Clear(alarmData)
	}

	return nil
}*/

func (a *AlarmManager) SetAlarmConfig(w http.ResponseWriter, r *http.Request) {
	var m alarm.AlarmConfigParams
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		app.Logger.Error("json.NewDecoder failed: %v", err)
	} else {
		a.maxActiveAlarms = m.MaxActiveAlarms
		a.maxAlarmHistory = m.MaxAlarmHistory
		app.Logger.Debug("new maxActiveAlarms = %v", a.maxActiveAlarms)
		app.Logger.Debug("new maxAlarmHistory = %v", a.maxAlarmHistory)
		a.respondWithJSON(w, http.StatusOK, err)
	}
}

func (a *AlarmManager) GetAlarmConfig(w http.ResponseWriter, r *http.Request) {
	var m alarm.AlarmConfigParams

	m.MaxActiveAlarms = a.maxActiveAlarms
	m.MaxAlarmHistory = a.maxAlarmHistory

	a.respondWithJSON(w, http.StatusOK, m)
	return
}

func (a *AlarmManager) SymptomDataHandler(w http.ResponseWriter, r *http.Request) {
	baseDir := "/tmp/symptomdata/"
	if err := app.Util.CreateDir(baseDir); err != nil {
		app.Resource.SendSymptomDataError(w, r, "CreateDir failed: "+err.Error())
		return
	}

	if b, err := json.MarshalIndent(a.activeAlarms, "", "    "); err == nil {
		if err := app.Util.WriteToFile(baseDir+"active_alarms.json", string(b)); err != nil {
			app.Resource.SendSymptomDataError(w, r, "writeToFile failed: "+err.Error())
			return
		}
	}

	if b, err := json.MarshalIndent(a.alarmHistory, "", "    "); err == nil {
		if err := app.Util.WriteToFile(baseDir+"alarm_history.json", string(b)); err != nil {
			app.Resource.SendSymptomDataError(w, r, "writeToFile failed: "+err.Error())
			return
		}
	}

	if alerts, err := a.GetAlerts(); err == nil {
		if b, err := json.MarshalIndent(alerts, "", "    "); err == nil {
			if err := app.Util.WriteToFile(baseDir+"alerts.json", string(b)); err != nil {
				app.Resource.SendSymptomDataError(w, r, "writeToFile failed: "+err.Error())
				return
			}
		}
	}

	var ac alarm.AlarmConfigParams
	ac.MaxActiveAlarms = a.maxActiveAlarms
	ac.MaxAlarmHistory = a.maxAlarmHistory

	if b, err := json.MarshalIndent(ac, "", "    "); err == nil {
		if err := app.Util.WriteToFile(baseDir+"alarm_config.json", string(b)); err != nil {
			app.Resource.SendSymptomDataError(w, r, "writeToFile failed: "+err.Error())
			return
		}
	}

	var ad RicAlarmDefinitions
	for _, alarmDefinition := range alarm.RICAlarmDefinitions {
		ad.AlarmDefinitions = append(ad.AlarmDefinitions, alarmDefinition)
	}
	if b, err := json.MarshalIndent(ad, "", "    "); err == nil {
		if err := app.Util.WriteToFile(baseDir+"alarm_defs.json", string(b)); err != nil {
			app.Resource.SendSymptomDataError(w, r, "writeToFile failed: "+err.Error())
			return
		}
	}

	app.Resource.SendSymptomDataFile(w, r, baseDir, "symptomdata.zip")
}
