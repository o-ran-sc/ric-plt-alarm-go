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
	"sync"
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

type AlertStatus string

const (
	AlertStatusActive   = "active"
	AlertStatusResolved = "resolved"
)

type AlarmAdapter struct {
	amHost        string
	amBaseUrl     string
	amSchemes     []string
	alertInterval int
	activeAlarms  []alarm.Alarm
	mutex         sync.Mutex
	rmrReady      bool
	postClear     bool
}

var Version string
var Hash string

// Main function
func main() {
	NewAlarmAdapter("", 0).Run(true)
}

func NewAlarmAdapter(amHost string, alertInterval int) *AlarmAdapter {
	if alertInterval == 0 {
		alertInterval = viper.GetInt("promAlertManager.alertInterval")
	}

	if amHost == "" {
		amHost = viper.GetString("promAlertManager.address")
	}

	return &AlarmAdapter{
		rmrReady:      false,
		amHost:        amHost,
		amBaseUrl:     viper.GetString("promAlertManager.baseUrl"),
		amSchemes:     []string{viper.GetString("promAlertManager.schemes")},
		alertInterval: alertInterval,
		activeAlarms:  make([]alarm.Alarm, 0),
	}
}

func (a *AlarmAdapter) Run(sdlcheck bool) {
	app.Logger.SetMdc("alarmAdapter", fmt.Sprintf("%s:%s", Version, Hash))
	app.SetReadyCB(func(d interface{}) { a.rmrReady = true }, true)
	app.Resource.InjectStatusCb(a.StatusCB)

	app.Resource.InjectRoute("/ric/v1/alarms", a.GetActiveAlarms, "GET")
	app.Resource.InjectRoute("/ric/v1/alarms", a.RaiseAlarm, "POST")
	app.Resource.InjectRoute("/ric/v1/alarms", a.ClearAlarm, "DELETE")

	// Start background timer for re-raising alerts
	a.postClear = sdlcheck
	go a.StartAlertTimer()

	app.RunWithParams(a, sdlcheck)
}

func (a *AlarmAdapter) StartAlertTimer() {
	tick := time.Tick(time.Duration(a.alertInterval) * time.Millisecond)
	for range tick {
		a.mutex.Lock()
		for _, m := range a.activeAlarms {
			app.Logger.Info("Re-raising alarm: %v", m)
			a.PostAlert(a.GenerateAlertLabels(m, AlertStatusActive))
		}
		a.mutex.Unlock()
	}
}

func (a *AlarmAdapter) Consume(rp *app.RMRParams) (err error) {
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

func (a *AlarmAdapter) HandleAlarms(rp *app.RMRParams) (*alert.PostAlertsOK, error) {
	var m alarm.AlarmMessage
	if err := json.Unmarshal(rp.Payload, &m); err != nil {
		app.Logger.Error("json.Unmarshal failed: %v", err)
		return nil, err
	}
	app.Logger.Info("newAlarm: %v", m)

	if _, ok := alarm.RICAlarmDefinitions[m.Alarm.SpecificProblem]; !ok {
		app.Logger.Warn("Alarm (SP='%d') not recognized, ignoring ...", m.Alarm.SpecificProblem)
		return nil, nil
	}

	// Suppress duplicate alarms
	idx, found := a.IsMatchFound(m.Alarm)
	if found && m.AlarmAction != alarm.AlarmActionClear {
		app.Logger.Info("Duplicate alarm ... suppressing!")
		return nil, nil
	}

	// Clear alarm if found from active alarm list
	if m.AlarmAction == alarm.AlarmActionClear {
		if found {
			a.activeAlarms = a.RemoveAlarm(a.activeAlarms, idx)
			app.Logger.Info("Active alarm cleared!")

			if a.postClear {
				return a.PostAlert(a.GenerateAlertLabels(m.Alarm, AlertStatusResolved))
			}
		}
		app.Logger.Info("No matching alarm found, ignoring!")
		return nil, nil
	}

	// New alarm -> update active alarms and post to Alert Manager
	if m.AlarmAction == alarm.AlarmActionRaise {
		a.UpdateActiveAlarms(m.Alarm)
		return a.PostAlert(a.GenerateAlertLabels(m.Alarm, AlertStatusActive))
	}

	return nil, nil
}

func (a *AlarmAdapter) IsMatchFound(newAlarm alarm.Alarm) (int, bool) {
	for i, m := range a.activeAlarms {
		if m.ManagedObjectId == newAlarm.ManagedObjectId && m.ApplicationId == newAlarm.ApplicationId &&
			m.SpecificProblem == newAlarm.SpecificProblem && m.IdentifyingInfo == newAlarm.IdentifyingInfo {
			return i, true
		}
	}
	return -1, false
}

func (a *AlarmAdapter) RemoveAlarm(alarms []alarm.Alarm, i int) []alarm.Alarm {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	copy(alarms[i:], alarms[i+1:])
	return alarms[:len(alarms)-1]
}

func (a *AlarmAdapter) UpdateActiveAlarms(newAlarm alarm.Alarm) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// For now just keep the active alarms in-memory. Use SDL later
	a.activeAlarms = append(a.activeAlarms, newAlarm)
}

func (a *AlarmAdapter) GenerateAlertLabels(newAlarm alarm.Alarm, status AlertStatus) (models.LabelSet, models.LabelSet) {
	alarmDef := alarm.RICAlarmDefinitions[newAlarm.SpecificProblem]
	amLabels := models.LabelSet{
		"status":      string(status),
		"alertname":   alarmDef.AlarmText,
		"severity":    string(newAlarm.PerceivedSeverity),
		"service":     fmt.Sprintf("%s:%s", newAlarm.ManagedObjectId, newAlarm.ApplicationId),
		"system_name": "RIC",
	}
	amAnnotations := models.LabelSet{
		"alarm_id":        string(alarmDef.AlarmId),
		"description":     newAlarm.IdentifyingInfo,
		"additional_info": newAlarm.AdditionalInfo,
		"summary":         alarmDef.EventType,
		"instructions":    alarmDef.OperationInstructions,
	}

	return amLabels, amAnnotations
}

func (a *AlarmAdapter) NewAlertmanagerClient() *client.Alertmanager {
	cr := clientruntime.New(a.amHost, a.amBaseUrl, a.amSchemes)
	return client.New(cr, strfmt.Default)
}

func (a *AlarmAdapter) PostAlert(amLabels, amAnnotations models.LabelSet) (*alert.PostAlertsOK, error) {
	pa := &models.PostableAlert{
		Alert: models.Alert{
			GeneratorURL: strfmt.URI(""),
			Labels:       amLabels,
		},
		Annotations: amAnnotations,
	}
	alertParams := alert.NewPostAlertsParams().WithAlerts(models.PostableAlerts{pa})

	app.Logger.Info("Posting alerts: labels: %v, annotations: %v", amLabels, amAnnotations)
	ok, err := a.NewAlertmanagerClient().Alert.PostAlerts(alertParams)
	if err != nil {
		app.Logger.Error("Posting alerts to '%s/%s' failed with error: %v", a.amHost, a.amBaseUrl, err)
	}
	return ok, err
}

func (a *AlarmAdapter) StatusCB() bool {
	if !a.rmrReady {
		app.Logger.Info("RMR not ready yet!")
	}

	return a.rmrReady
}
