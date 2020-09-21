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
	"fmt"
	"time"

	clientruntime "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/spf13/viper"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	app "gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
)

func (a *AlarmManager) StartAlertTimer() {
	tick := time.Tick(time.Duration(a.alertInterval) * time.Millisecond)
	for range tick {
		a.mutex.Lock()
		for _, m := range a.activeAlarms {
			app.Logger.Info("Re-raising alarm: %v", m)
			a.PostAlert(a.GenerateAlertLabels(m.Alarm, AlertStatusActive))
		}
		a.mutex.Unlock()
	}
}

func (a *AlarmManager) Consume(rp *app.RMRParams) (err error) {
	app.Logger.Info("Message received!")

	defer app.Rmr.Free(rp.Mbuf)
	switch rp.Mtype {
	case alarm.RIC_ALARM_UPDATE:
		a.HandleAlarms(rp)
	default:
		app.Logger.Info("Unknown Message Type '%d', discarding", rp.Mtype)
	}

	return nil
}

func (a *AlarmManager) HandleAlarms(rp *app.RMRParams) (*alert.PostAlertsOK, error) {
	var m alarm.AlarmMessage
	app.Logger.Info("Received JSON: %s", rp.Payload)
	if err := json.Unmarshal(rp.Payload, &m); err != nil {
		app.Logger.Error("json.Unmarshal failed: %v", err)
		return nil, err
	}
	app.Logger.Info("newAlarm: %v", m)

	return a.ProcessAlarm(&m)
}

func (a *AlarmManager) ProcessAlarm(m *alarm.AlarmMessage) (*alert.PostAlertsOK, error) {
	if _, ok := alarm.RICAlarmDefinitions[m.Alarm.SpecificProblem]; !ok {
		app.Logger.Warn("Alarm (SP='%d') not recognized, suppressing ...", m.Alarm.SpecificProblem)
		return nil, nil
	}

	// Suppress duplicate alarms
	idx, found := a.IsMatchFound(m.Alarm)
	if found && m.AlarmAction != alarm.AlarmActionClear {
		app.Logger.Info("Duplicate alarm found, suppressing ...")
		return nil, nil
	}

	// Clear alarm if found from active alarm list
	if m.AlarmAction == alarm.AlarmActionClear {
		if found {
			a.alarmHistory = append(a.alarmHistory, *m)
			a.activeAlarms = a.RemoveAlarm(a.activeAlarms, idx, "active")

			if a.postClear {
				return a.PostAlert(a.GenerateAlertLabels(m.Alarm, AlertStatusResolved))
			}
		}
		app.Logger.Info("No matching active alarm found, suppressing ...")
		return nil, nil
	}

	// New alarm -> update active alarms and post to Alert Manager
	if m.AlarmAction == alarm.AlarmActionRaise {
		a.UpdateAlarmLists(m)
		return a.PostAlert(a.GenerateAlertLabels(m.Alarm, AlertStatusActive))
	}

	return nil, nil
}

func (a *AlarmManager) IsMatchFound(newAlarm alarm.Alarm) (int, bool) {
	for i, m := range a.activeAlarms {
		if m.ManagedObjectId == newAlarm.ManagedObjectId && m.ApplicationId == newAlarm.ApplicationId &&
			m.SpecificProblem == newAlarm.SpecificProblem && m.IdentifyingInfo == newAlarm.IdentifyingInfo {
			return i, true
		}
	}
	return -1, false
}

func (a *AlarmManager) RemoveAlarm(alarms []alarm.AlarmMessage, i int, listName string) []alarm.AlarmMessage {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	app.Logger.Info("Alarm '%+v' deleted from the '%s' list", alarms[i], listName)
	copy(alarms[i:], alarms[i+1:])
	return alarms[:len(alarms)-1]
}

func (a *AlarmManager) UpdateAlarmLists(newAlarm *alarm.AlarmMessage) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// If maximum number of active alarms is reached, purge the oldest alarm
	if len(a.activeAlarms) >= viper.GetInt("controls.maxActiveAlarms") {
		a.activeAlarms = a.RemoveAlarm(a.activeAlarms, 0, "active")
	}

	if len(a.alarmHistory) >= viper.GetInt("controls.maxAlarmHistory") {
		a.alarmHistory = a.RemoveAlarm(a.alarmHistory, 0, "history")
	}

	// @todo: For now just keep the alarms (both active and history) in-memory. Use SDL later for persistence
	a.activeAlarms = append(a.activeAlarms, *newAlarm)
	a.alarmHistory = append(a.alarmHistory, *newAlarm)
}

func (a *AlarmManager) GenerateAlertLabels(newAlarm alarm.Alarm, status AlertStatus) (models.LabelSet, models.LabelSet) {
	alarmDef := alarm.RICAlarmDefinitions[newAlarm.SpecificProblem]
	amLabels := models.LabelSet{
		"status":      string(status),
		"alertname":   alarmDef.AlarmText,
		"severity":    string(newAlarm.PerceivedSeverity),
		"service":     fmt.Sprintf("%s:%s", newAlarm.ManagedObjectId, newAlarm.ApplicationId),
		"system_name": fmt.Sprintf("RIC:%s:%s", newAlarm.ManagedObjectId, newAlarm.ApplicationId),
	}
	amAnnotations := models.LabelSet{
		"alarm_id":        fmt.Sprintf("%d", alarmDef.AlarmId),
		"description":     fmt.Sprintf("%d:%s:%s", newAlarm.SpecificProblem, newAlarm.IdentifyingInfo, newAlarm.AdditionalInfo),
		"additional_info": newAlarm.AdditionalInfo,
		"summary":         alarmDef.EventType,
		"instructions":    alarmDef.OperationInstructions,
	}

	return amLabels, amAnnotations
}

func (a *AlarmManager) NewAlertmanagerClient() *client.Alertmanager {
	cr := clientruntime.New(a.amHost, a.amBaseUrl, a.amSchemes)
	return client.New(cr, strfmt.Default)
}

func (a *AlarmManager) PostAlert(amLabels, amAnnotations models.LabelSet) (*alert.PostAlertsOK, error) {
	pa := &models.PostableAlert{
		Alert: models.Alert{
			GeneratorURL: strfmt.URI(""),
			Labels:       amLabels,
		},
		Annotations: amAnnotations,
	}
	alertParams := alert.NewPostAlertsParams().WithAlerts(models.PostableAlerts{pa})

	app.Logger.Info("Posting alerts: labels: %+v, annotations: %+v", amLabels, amAnnotations)
	ok, err := a.NewAlertmanagerClient().Alert.PostAlerts(alertParams)
	if err != nil {
		app.Logger.Error("Posting alerts to '%s/%s' failed with error: %v", a.amHost, a.amBaseUrl, err)
	}
	return ok, err
}

func (a *AlarmManager) StatusCB() bool {
	if !a.rmrReady {
		app.Logger.Info("RMR not ready yet!")
	}

	return a.rmrReady
}

func (a *AlarmManager) Run(sdlcheck bool) {
	app.Logger.SetMdc("alarmManager", fmt.Sprintf("%s:%s", Version, Hash))
	app.SetReadyCB(func(d interface{}) { a.rmrReady = true }, true)
	app.Resource.InjectStatusCb(a.StatusCB)

	app.Resource.InjectRoute("/ric/v1/alarms", a.RaiseAlarm, "POST")
	app.Resource.InjectRoute("/ric/v1/alarms", a.ClearAlarm, "DELETE")
	app.Resource.InjectRoute("/ric/v1/alarms/active", a.GetActiveAlarms, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms/history", a.GetAlarmHistory, "GET")

	// Start background timer for re-raising alerts
	a.postClear = sdlcheck
	go a.StartAlertTimer()

	app.RunWithParams(a, sdlcheck)
}

func NewAlarmManager(amHost string, alertInterval int) *AlarmManager {
	if alertInterval == 0 {
		alertInterval = viper.GetInt("controls.promAlertManager.alertInterval")
	}

	if amHost == "" {
		amHost = viper.GetString("controls.promAlertManager.address")
	}

	return &AlarmManager{
		rmrReady:      false,
		amHost:        amHost,
		amBaseUrl:     viper.GetString("controls.promAlertManager.baseUrl"),
		amSchemes:     []string{viper.GetString("controls.promAlertManager.schemes")},
		alertInterval: alertInterval,
		activeAlarms:  make([]alarm.AlarmMessage, 0),
		alarmHistory:  make([]alarm.AlarmMessage, 0),
	}
}

// Main function
func main() {
	NewAlarmManager("", 0).Run(true)
}
