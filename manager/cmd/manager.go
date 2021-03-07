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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	app "gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	clientruntime "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/spf13/viper"
)

func (a *AlarmManager) ClearExpiredAlarms(m AlarmNotification, idx int, mLocked bool) bool {
	d, ok := alarm.RICAlarmDefinitions[m.Alarm.SpecificProblem]
	if !ok || d.TimeToLive == 0 {
		return false
	}

	elapsed := (time.Now().UnixNano() - m.AlarmTime) / 1e9
	if int(elapsed) >= d.TimeToLive {
		app.Logger.Info("Alarm (sp=%d id=%d) with TTL=%d expired, clearing ...", m.Alarm.SpecificProblem, m.AlarmId, d.TimeToLive)

		m.AlarmAction = alarm.AlarmActionClear
		m.AlarmTime = time.Now().UnixNano()

		if !mLocked { // For testing purpose
			a.mutex.Lock()
		}
		a.ProcessClearAlarm(&m, d, idx)
		return true
	}
	return false
}

func (a *AlarmManager) StartTTLTimer(interval int) {
	tick := time.Tick(time.Duration(interval) * time.Second)
	for range tick {
		a.mutex.Lock()
		for idx, m := range a.activeAlarms {
			if a.ClearExpiredAlarms(m, idx, true) {
				a.mutex.Lock() // ClearExpiredAlarms unlocks the mutex, so re-lock here
				continue
			}
		}
		a.mutex.Unlock()
	}
}

func (a *AlarmManager) StartAlertTimer() {
	tick := time.Tick(time.Duration(a.alertInterval) * time.Millisecond)
	for range tick {
		a.mutex.Lock()
		for _, m := range a.activeAlarms {
			app.Logger.Info("Re-raising alarm: %v", m)
			a.PostAlert(a.GenerateAlertLabels(m.AlarmId, m.Alarm, AlertStatusActive, m.AlarmTime))
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

	return a.ProcessAlarm(&AlarmNotification{m, alarm.AlarmDefinition{}})
}

func (a *AlarmManager) ProcessAlarm(m *AlarmNotification) (*alert.PostAlertsOK, error) {
	a.mutex.Lock()
	alarmDef := &alarm.AlarmDefinition{}
	var ok bool
	if alarmDef, ok = alarm.RICAlarmDefinitions[m.Alarm.SpecificProblem]; !ok {
		app.Logger.Warn("Alarm (SP='%d') not recognized, suppressing ...", m.Alarm.SpecificProblem)
		a.mutex.Unlock()
		return nil, nil
	}

	idx, found := a.IsMatchFound(m.Alarm)
	// Suppress duplicate alarms
	if found && m.AlarmAction == alarm.AlarmActionRaise {
		app.Logger.Info("Duplicate alarm found, suppressing ...")
		if m.PerceivedSeverity == a.activeAlarms[idx].PerceivedSeverity {
			// Duplicate with same severity found
			a.mutex.Unlock()
			return nil, nil
		} else {
			// Remove duplicate with different severity
			a.activeAlarms = a.RemoveAlarm(a.activeAlarms, idx, "active")
		}
	}

	// Clear alarm if found from active alarm list
	if found && m.AlarmAction == alarm.AlarmActionClear {
		return a.ProcessClearAlarm(m, alarmDef, idx)
	}

	// New alarm -> update active alarms and post to Alert Manager
	if m.AlarmAction == alarm.AlarmActionRaise {
		return a.ProcessRaiseAlarm(m, alarmDef)
	}

	a.mutex.Unlock()
	return nil, nil
}

func (a *AlarmManager) ProcessRaiseAlarm(m *AlarmNotification, alarmDef *alarm.AlarmDefinition) (*alert.PostAlertsOK, error) {
	app.Logger.Debug("Raise alarmDef.RaiseDelay = %v, AlarmNotification = %v", alarmDef.RaiseDelay, *m)

	// RaiseDelay > 0 in an alarm object in active alarm table indicates that raise delay is still ongoing for the alarm
	m.AlarmDefinition.RaiseDelay = alarmDef.RaiseDelay
	a.UpdateAlarmFields(a.GenerateAlarmId(), m)
	a.UpdateActiveAlarmList(m)
	a.mutex.Unlock()

	if alarmDef.RaiseDelay > 0 {
		timerDelay(alarmDef.RaiseDelay)
		a.mutex.Lock()
		// Alarm may have been deleted from active alarms table during delay or table index may have changed
		idx, found := a.IsMatchFound(m.Alarm)
		if found {
			// Alarm is not showed in active alarms or alarm history via CLI before RaiseDelay has elapsed, i.e the value is 0
			a.activeAlarms[idx].AlarmDefinition.RaiseDelay = 0
			app.Logger.Debug("Raise after delay alarmDef.RaiseDelay = %v, AlarmNotification = %v", alarmDef.RaiseDelay, *m)
			a.mutex.Unlock()
		} else {
			app.Logger.Debug("Alarm deleted during raise delay. AlarmNotification = %v", *m)
			a.mutex.Unlock()
			return nil, nil
		}
	}

	m.AlarmDefinition.RaiseDelay = 0
	a.UpdateAlarmHistoryList(m)
	a.WriteAlarmInfoToPersistentVolume()

	// Send alarm notification to NOMA, if enabled
	if app.Config.GetBool("controls.noma.enabled") {
		return a.PostAlarm(m)
	}
	return a.PostAlert(a.GenerateAlertLabels(m.AlarmId, m.Alarm, AlertStatusActive, m.AlarmTime))
}

func (a *AlarmManager) ProcessClearAlarm(m *AlarmNotification, alarmDef *alarm.AlarmDefinition, idx int) (*alert.PostAlertsOK, error) {
	app.Logger.Debug("Clear alarmDef.ClearDelay = %v, AlarmNotification = %v", alarmDef.ClearDelay, *m)
	if alarmDef.ClearDelay > 0 {
		a.mutex.Unlock()
		timerDelay(alarmDef.ClearDelay)
		app.Logger.Debug("Clear after delay alarmDef.ClearDelay = %v, AlarmNotification = %v", alarmDef.ClearDelay, *m)
		a.mutex.Lock()
		// Another alarm clear may have happened during delay and active alarms table index changed
		var found bool
		idx, found = a.IsMatchFound(m.Alarm)
		if !found {
			a.mutex.Unlock()
			return nil, nil
		}
	}
	a.UpdateAlarmFields(a.activeAlarms[idx].AlarmId, m)
	a.alarmHistory = append(a.alarmHistory, *m)
	a.activeAlarms = a.RemoveAlarm(a.activeAlarms, idx, "active")
	if (len(a.alarmHistory) >= a.maxAlarmHistory) && (a.exceededAlarmHistoryOn == false) {
		app.Logger.Warn("alarm history count exceeded maxAlarmHistory threshold")
		a.GenerateThresholdAlarm(alarm.ALARM_HISTORY_EXCEED_MAX_THRESHOLD, "history")
	}

	if a.exceededActiveAlarmOn && m.Alarm.SpecificProblem == alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD {
		a.exceededActiveAlarmOn = false
	}

	if a.exceededAlarmHistoryOn && m.Alarm.SpecificProblem == alarm.ALARM_HISTORY_EXCEED_MAX_THRESHOLD {
		a.exceededAlarmHistoryOn = false
	}
	a.WriteAlarmInfoToPersistentVolume()

	a.mutex.Unlock()
	if a.postClear && app.Config.GetBool("controls.noma.enabled") {
		m.PerceivedSeverity = alarm.SeverityCleared
		return a.PostAlarm(m)
	}
	return nil, nil
}

func timerDelay(delay int) {
	timer := time.NewTimer(time.Duration(delay) * time.Second)
	<-timer.C
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

func (a *AlarmManager) RemoveAlarm(alarms []AlarmNotification, i int, listName string) []AlarmNotification {
	app.Logger.Info("Alarm '%+v' deleted from the '%s' list", alarms[i], listName)
	copy(alarms[i:], alarms[i+1:])
	return alarms[:len(alarms)-1]
}

func (a *AlarmManager) GenerateAlarmId() int {
	a.uniqueAlarmId++ // @todo: generate a unique ID
	return a.uniqueAlarmId
}

func (a *AlarmManager) UpdateAlarmFields(alarmId int, newAlarm *AlarmNotification) {
	alarmDef := alarm.RICAlarmDefinitions[newAlarm.SpecificProblem]
	newAlarm.AlarmId = alarmId
	newAlarm.AlarmText = alarmDef.AlarmText
	newAlarm.EventType = alarmDef.EventType
}

func (a *AlarmManager) GenerateThresholdAlarm(sp int, data string) bool {
	thresholdAlarm := a.alarmClient.NewAlarm(sp, alarm.SeverityWarning, "threshold", data)
	thresholdMessage := alarm.AlarmMessage{
		Alarm:       thresholdAlarm,
		AlarmAction: alarm.AlarmActionRaise,
		AlarmTime:   time.Now().UnixNano(),
	}
	alarmDef := alarm.RICAlarmDefinitions[sp]
	alarmId := a.GenerateAlarmId()
	alarmDef.AlarmId = alarmId
	a.activeAlarms = append(a.activeAlarms, AlarmNotification{thresholdMessage, *alarmDef})
	a.alarmHistory = append(a.alarmHistory, AlarmNotification{thresholdMessage, *alarmDef})

	return true
}

func (a *AlarmManager) UpdateActiveAlarmList(newAlarm *AlarmNotification) {
	/* If maximum number of active alarms is reached, an error log writing is made, and new alarm indicating the problem is raised.
	   The attempt to raise the alarm next time will be suppressed when found as duplicate. */
	if (len(a.activeAlarms) >= a.maxActiveAlarms) && (a.exceededActiveAlarmOn == false) {
		app.Logger.Warn("active alarm count exceeded maxActiveAlarms threshold")
		a.exceededActiveAlarmOn = a.GenerateThresholdAlarm(alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD, "active")
	}

	// @todo: For now just keep the  active alarms in-memory. Use SDL later for persistence
	a.activeAlarms = append(a.activeAlarms, *newAlarm)
}

func (a *AlarmManager) UpdateAlarmHistoryList(newAlarm *AlarmNotification) {
	/* If maximum number of events in alarm history is reached, an error log writing is made,
	   and new alarm indicating the problem is raised. The attempt to add new event time will
	   be suppressed */

	if (len(a.alarmHistory) >= a.maxAlarmHistory) && (a.exceededAlarmHistoryOn == false) {
		app.Logger.Warn("alarm history count exceeded maxAlarmHistory threshold")
		a.exceededAlarmHistoryOn = a.GenerateThresholdAlarm(alarm.ALARM_HISTORY_EXCEED_MAX_THRESHOLD, "history")
	}

	// @todo: For now just keep the alarms history in-memory. Use SDL later for persistence
	a.alarmHistory = append(a.alarmHistory, *newAlarm)
}

func (a *AlarmManager) PostAlarm(m *AlarmNotification) (*alert.PostAlertsOK, error) {
	result, err := json.Marshal(m)
	if err != nil {
		app.Logger.Info("json.Marshal failed: %v", err)
		return nil, err
	}

	fullUrl := fmt.Sprintf("%s/%s", app.Config.GetString("controls.noma.host"), app.Config.GetString("controls.noma.alarmUrl"))
	app.Logger.Info("Posting alarm to '%s'", fullUrl)

	resp, err := http.Post(fullUrl, "application/json", bytes.NewReader(result))
	if err != nil || resp == nil {
		app.Logger.Info("Unable to post alarm to '%s': %v", fullUrl, err)
	}

	return nil, err
}

func (a *AlarmManager) GenerateAlertLabels(alarmId int, newAlarm alarm.Alarm, status AlertStatus, alarmTime int64) (models.LabelSet, models.LabelSet) {
	alarmDef := alarm.RICAlarmDefinitions[newAlarm.SpecificProblem]
	amLabels := models.LabelSet{
		"status":      string(status),
		"alertname":   alarmDef.AlarmText,
		"severity":    string(newAlarm.PerceivedSeverity),
		"service":     fmt.Sprintf("%s/%s", newAlarm.ManagedObjectId, newAlarm.ApplicationId),
		"system_name": "RIC",
	}
	amAnnotations := models.LabelSet{
		"alarm_id":         fmt.Sprintf("%d", alarmId),
		"specific_problem": fmt.Sprintf("%d", newAlarm.SpecificProblem),
		"event_type":       alarmDef.EventType,
		"identifying_info": newAlarm.IdentifyingInfo,
		"additional_info":  newAlarm.AdditionalInfo,
		"description":      fmt.Sprintf("%s:%s", newAlarm.IdentifyingInfo, newAlarm.AdditionalInfo),
		"instructions":     alarmDef.OperationInstructions,
		"timestamp":        fmt.Sprintf("%s", time.Unix(0, alarmTime).Format("02/01/2006, 15:04:05")),
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

func (a *AlarmManager) ConfigChangeCB(configparam string) {

	a.maxActiveAlarms = app.Config.GetInt("controls.maxActiveAlarms")
	if a.maxActiveAlarms == 0 {
		a.maxActiveAlarms = 5000
	}

	a.maxAlarmHistory = app.Config.GetInt("controls.maxAlarmHistory")
	if a.maxAlarmHistory == 0 {
		a.maxAlarmHistory = 20000
	}

	a.alertInterval = viper.GetInt("controls.promAlertManager.alertInterval")
	a.amHost = viper.GetString("controls.promAlertManager.address")

	app.Logger.Debug("ConfigChangeCB: maxActiveAlarms %v", a.maxActiveAlarms)
	app.Logger.Debug("ConfigChangeCB: maxAlarmHistory = %v", a.maxAlarmHistory)
	app.Logger.Debug("ConfigChangeCB: alertInterval %v", a.alertInterval)
	app.Logger.Debug("ConfigChangeCB: amHost = %v", a.amHost)

	return
}

func (a *AlarmManager) ReadAlarmDefinitionFromJson() {

	filename := os.Getenv("DEF_FILE")
	file, err := ioutil.ReadFile(filename)
	if err == nil {
		data := RicAlarmDefinitions{}
		err = json.Unmarshal([]byte(file), &data)
		if err == nil {
			for _, alarmDefinition := range data.AlarmDefinitions {
				_, exists := alarm.RICAlarmDefinitions[alarmDefinition.AlarmId]
				if exists {
					app.Logger.Error("ReadAlarmDefinitionFromJson: alarm definition already exists for %v", alarmDefinition.AlarmId)
				} else {
					app.Logger.Debug("ReadAlarmDefinitionFromJson: alarm  %v", alarmDefinition.AlarmId)
					ricAlarmDefintion := new(alarm.AlarmDefinition)
					ricAlarmDefintion.AlarmId = alarmDefinition.AlarmId
					ricAlarmDefintion.AlarmText = alarmDefinition.AlarmText
					ricAlarmDefintion.EventType = alarmDefinition.EventType
					ricAlarmDefintion.OperationInstructions = alarmDefinition.OperationInstructions
					ricAlarmDefintion.RaiseDelay = alarmDefinition.RaiseDelay
					ricAlarmDefintion.ClearDelay = alarmDefinition.ClearDelay
					ricAlarmDefintion.TimeToLive = alarmDefinition.TimeToLive
					alarm.RICAlarmDefinitions[alarmDefinition.AlarmId] = ricAlarmDefintion
				}
			}
		} else {
			app.Logger.Error("ReadAlarmDefinitionFromJson: json.Unmarshal failed with error %v", err)
		}
	} else {
		app.Logger.Error("ReadAlarmDefinitionFromJson: ioutil.ReadFile failed with error %v", err)
	}
}

func (a *AlarmManager) ReadAlarmInfoFromPersistentVolume() {
	var alarmpersistentinfo AlarmPersistentInfo
	byteValue, rerr := ioutil.ReadFile(a.alarmInfoPvFile)
	if rerr != nil {
		app.Logger.Error("ararminfo.json file read error %v", rerr)
	} else {
		err := json.Unmarshal(byteValue, &alarmpersistentinfo)
		if err != nil {
			app.Logger.Error("alarmpersistentinfo json unmarshal error %v", err)
		} else {
			a.uniqueAlarmId = alarmpersistentinfo.UniqueAlarmId
			a.activeAlarms = make([]AlarmNotification, len(alarmpersistentinfo.ActiveAlarms))
			a.alarmHistory = make([]AlarmNotification, len(alarmpersistentinfo.AlarmHistory))
			copy(a.activeAlarms, alarmpersistentinfo.ActiveAlarms)
			copy(a.alarmHistory, alarmpersistentinfo.AlarmHistory)
		}
	}
}

func (a *AlarmManager) WriteAlarmInfoToPersistentVolume() {
	var alarmpersistentinfo AlarmPersistentInfo
	alarmpersistentinfo.UniqueAlarmId = a.uniqueAlarmId
	alarmpersistentinfo.ActiveAlarms = make([]AlarmNotification, len(a.activeAlarms))
	alarmpersistentinfo.AlarmHistory = make([]AlarmNotification, len(a.alarmHistory))

	copy(alarmpersistentinfo.ActiveAlarms, a.activeAlarms)
	copy(alarmpersistentinfo.AlarmHistory, a.alarmHistory)

	wdata, err := json.MarshalIndent(alarmpersistentinfo, "", " ")
	if err != nil {
		app.Logger.Error("alarmpersistentinfo json marshal error %v", err)
	} else {
		werr := ioutil.WriteFile(a.alarmInfoPvFile, wdata, 0777)
		if werr != nil {
			app.Logger.Error("alarminfo.json file write error %v", werr)
		}
	}
}

func (a *AlarmManager) Run(sdlcheck bool, ttlInterval int) {
	app.Logger.SetMdc("alarmManager", fmt.Sprintf("%s:%s", Version, Hash))
	app.SetReadyCB(func(d interface{}) { a.rmrReady = true }, true)
	app.Resource.InjectStatusCb(a.StatusCB)
	app.AddConfigChangeListener(a.ConfigChangeCB)

	alarm.RICAlarmDefinitions = make(map[int]*alarm.AlarmDefinition)
	a.ReadAlarmDefinitionFromJson()

	a.InjectRoutes()

	// Start background timer for re-raising alerts
	go a.StartAlertTimer()
	go a.StartTTLTimer(ttlInterval)

	a.alarmClient, _ = alarm.InitAlarm("SEP", "ALARMMANAGER")

	a.ReadAlarmInfoFromPersistentVolume()

	app.RunWithParams(a, sdlcheck)
}

func NewAlarmManager(amHost string, alertInterval int, clearAlarm bool) *AlarmManager {
	if alertInterval == 0 {
		alertInterval = viper.GetInt("controls.promAlertManager.alertInterval")
	}

	if amHost == "" {
		amHost = viper.GetString("controls.promAlertManager.address")
	}

	maxActiveAlarms := app.Config.GetInt("controls.maxActiveAlarms")
	if maxActiveAlarms == 0 {
		maxActiveAlarms = 5000
	}

	maxAlarmHistory := app.Config.GetInt("controls.maxAlarmHistory")
	if maxAlarmHistory == 0 {
		maxAlarmHistory = 20000
	}

	return &AlarmManager{
		rmrReady:               false,
		postClear:              clearAlarm,
		amHost:                 amHost,
		amBaseUrl:              app.Config.GetString("controls.promAlertManager.baseUrl"),
		amSchemes:              []string{app.Config.GetString("controls.promAlertManager.schemes")},
		alertInterval:          alertInterval,
		activeAlarms:           make([]AlarmNotification, 0),
		alarmHistory:           make([]AlarmNotification, 0),
		uniqueAlarmId:          0,
		maxActiveAlarms:        maxActiveAlarms,
		maxAlarmHistory:        maxAlarmHistory,
		exceededActiveAlarmOn:  false,
		exceededAlarmHistoryOn: false,
		alarmInfoPvFile:        app.Config.GetString("controls.alarmInfoPvFile"),
	}
}

// Main function
func main() {
	NewAlarmManager("", 0, true).Run(true, 10)
}
