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

type AlarmAdapter struct {
	amHost        string
	amBaseUrl     string
	amSchemes     []string
	alertInterval int
	activeAlarms  []alarm.Alarm
	rmrReady      bool
}

// Temp alarm constants & definitions
const (
	RIC_RT_DISTRIBUTION_FAILED     int = 8004
	CONNECTIVITY_LOST_TO_DBAAS     int = 8005
	E2_CONNECTIVITY_LOST_TO_GNODEB int = 8006
	E2_CONNECTIVITY_LOST_TO_ENODEB int = 8007
)

var alarmDefinitions = map[int]string{
	RIC_RT_DISTRIBUTION_FAILED:     "RIC ROUTING TABLE DISTRIBUTION FAILED",
	CONNECTIVITY_LOST_TO_DBAAS:     "CONNECTIVITY LOST TO DBAAS",
	E2_CONNECTIVITY_LOST_TO_GNODEB: "E2 CONNECTIVITY LOST TO G-NODEB",
	E2_CONNECTIVITY_LOST_TO_ENODEB: "E2 CONNECTIVITY LOST TO E-NODEB",
}

var Version string
var Hash string

// Main function
func main() {
	NewAlarmAdapter(0).Run(true)
}

func NewAlarmAdapter(alertInterval int) *AlarmAdapter {
	if alertInterval == 0 {
		alertInterval = viper.GetInt("promAlertManager.alertInterval")
	}

	return &AlarmAdapter{
		rmrReady:      false,
		amHost:        viper.GetString("promAlertManager.address"),
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

	// Start background timer for re-raising alerts
	go a.StartAlertTimer()

	app.RunWithParams(a, sdlcheck)
}

func (a *AlarmAdapter) StartAlertTimer() {
	tick := time.Tick(time.Duration(a.alertInterval) * time.Millisecond)
	for range tick {
		for _, m := range a.activeAlarms {
			app.Logger.Info("Re-raising alarm: %v", m)
			a.PostAlert(a.GenerateAlertLabels(m))
		}
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

	if _, ok := alarmDefinitions[m.Alarm.SpecificProblem]; !ok {
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
		} else {
			app.Logger.Info("No matching alarm found, ignoring!")
		}
		return nil, nil
	}

	// New alarm -> update active alarms and post to Alert Manager
	if m.AlarmAction == alarm.AlarmActionRaise {
		a.UpdateActiveAlarms(m.Alarm)
		return a.PostAlert(a.GenerateAlertLabels(m.Alarm))
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
	copy(alarms[i:], alarms[i+1:])
	return alarms[:len(alarms)-1]
}

func (a *AlarmAdapter) UpdateActiveAlarms(newAlarm alarm.Alarm) {
	// For now just keep the active alarms in-memory. Use SDL later
	a.activeAlarms = append(a.activeAlarms, newAlarm)
}

func (a *AlarmAdapter) GenerateAlertLabels(newAlarm alarm.Alarm) (models.LabelSet, models.LabelSet) {
	amLabels := models.LabelSet{
		"alertname":   alarmDefinitions[newAlarm.SpecificProblem],
		"severity":    string(newAlarm.PerceivedSeverity),
		"service":     fmt.Sprintf("%s:%s", newAlarm.ManagedObjectId, newAlarm.ApplicationId),
		"system_name": "RIC",
	}
	amAnnotations := models.LabelSet{
		"description":     newAlarm.IdentifyingInfo,
		"additional_info": newAlarm.AdditionalInfo,
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
	return a.NewAlertmanagerClient().Alert.PostAlerts(alertParams)
}

func (a *AlarmAdapter) StatusCB() bool {
	if !a.rmrReady {
		app.Logger.Info("RMR not ready yet!")
	}

	return a.rmrReady
}
