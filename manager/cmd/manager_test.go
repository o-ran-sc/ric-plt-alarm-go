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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go.git/alarm"
	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"github.com/gorilla/mux"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/assert"
)

var alarmManager *AlarmManager
var alarmer *alarm.RICAlarm
var eventChan chan string

// Test cases
func TestMain(M *testing.M) {
	alarmManager = NewAlarmManager("localhost:9093", 500, false)
	alarmManager.alertInterval = 20000
	go alarmManager.Run(false, 5)
	time.Sleep(time.Duration(10) * time.Second)

	// Wait until RMR is up-and-running
	for !xapp.Rmr.IsReady() {
		time.Sleep(time.Duration(1) * time.Second)
	}

	alarmer, _ = alarm.InitAlarm("my-pod", "my-app")
	alarmManager.alarmClient = alarmer
	time.Sleep(time.Duration(5) * time.Second)
	eventChan = make(chan string)

	os.Exit(M.Run())
}

func TestGetPreDefinedAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestGetPreDefinedAlarmDefinitions")
	var alarmDefinition alarm.AlarmDefinition
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.NewDecoder(response.Body).Decode(&alarmDefinition)
	xapp.Logger.Info("alarm definition = %v", alarmDefinition)
	if alarmDefinition.AlarmId != alarm.E2_CONNECTION_PROBLEM || alarmDefinition.AlarmText != "E2 CONNECTION PROBLEM" {
		t.Errorf("Incorrect alarm definition")
	}
}

func TestSetAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmDefinitions")

	var alarm72004Definition alarm.AlarmDefinition
	alarm72004Definition.AlarmId = alarm.E2_CONNECTION_PROBLEM
	alarm72004Definition.AlarmText = "E2 CONNECTIVITY LOST TO E-NODEB/G-NODEB"
	alarm72004Definition.EventType = "Communication error"
	alarm72004Definition.OperationInstructions = "Not defined"
	alarm72004Definition.RaiseDelay = 0
	alarm72004Definition.ClearDelay = 0

	var alarm72008Definition alarm.AlarmDefinition
	alarm72008Definition.AlarmId = alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD
	alarm72008Definition.AlarmText = "ACTIVE ALARM EXCEED MAX THRESHOLD"
	alarm72008Definition.EventType = "storage warning"
	alarm72008Definition.OperationInstructions = "Clear alarms or raise threshold"
	alarm72008Definition.RaiseDelay = 0
	alarm72008Definition.ClearDelay = 0

	var alarm72009Definition alarm.AlarmDefinition
	alarm72009Definition.AlarmId = alarm.ALARM_HISTORY_EXCEED_MAX_THRESHOLD
	alarm72009Definition.AlarmText = "ALARM HISTORY EXCEED MAX THRESHOLD"
	alarm72009Definition.EventType = "storage warning"
	alarm72009Definition.OperationInstructions = "Clear alarms or raise threshold"
	alarm72009Definition.RaiseDelay = 0
	alarm72009Definition.ClearDelay = 0

	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm72004Definition, &alarm72008Definition, &alarm72009Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusOK, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestSetAlarmConfigDecodeError(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmConfigDecodeError")

	var jsonStr = []byte(`{"test":"Invalid Alarm Data", "test1" 123}`)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms", bytes.NewBuffer(jsonStr))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmConfig)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusOK, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestSetAlarmDefinitionDecodeError(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmDefinitionDecodeError")

	var jsonStr = []byte(`{"test":"Invalid Alarm Data", "test1" 123}`)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(jsonStr))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusBadRequest, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestRaiseAlarmEmptyBody(t *testing.T) {
	xapp.Logger.Info("TestRaiseAlarmEmptyBody")
	req, _ := http.NewRequest("POST", "/ric/v1/alarms", nil)
	handleFunc := http.HandlerFunc(alarmManager.RaiseAlarm)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusOK, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestSetAlarmDefinitionsEmptyBody(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmDefinitionsEmptyBody")
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/define", nil)
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusBadRequest, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestClearAlarmEmptyBody(t *testing.T) {
	xapp.Logger.Info("TestClearAlarmEmptyBody")
	req, _ := http.NewRequest("DELETE", "/ric/v1/alarms", nil)
	handleFunc := http.HandlerFunc(alarmManager.ClearAlarm)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusOK, response.Code)
	xapp.Logger.Info("status = %v", status)
}

func TestGetAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestGetAlarmDefinitions")
	var alarmDefinition alarm.AlarmDefinition
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.NewDecoder(response.Body).Decode(&alarmDefinition)
	xapp.Logger.Info("alarm definition = %v", alarmDefinition)
	if alarmDefinition.AlarmId != alarm.E2_CONNECTION_PROBLEM || alarmDefinition.AlarmText != "E2 CONNECTION PROBLEM" {
		t.Errorf("Incorrect alarm definition")
	}
}

func TestDeleteAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestDeleteAlarmDefinitions")
	//Get all
	var ricAlarmDefinitions RicAlarmDefinitions
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.NewDecoder(response.Body).Decode(&ricAlarmDefinitions)
	for _, alarmDefinition := range ricAlarmDefinitions.AlarmDefinitions {
		xapp.Logger.Info("alarm definition = %v", *alarmDefinition)
	}

	//Delete 72004
	req, _ = http.NewRequest("DELETE", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	//Get 72004 fail
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusBadRequest, response.Code)

	//Delete Alarm which doesn't present
	//Set 72004 success
	var alarm72004Definition alarm.AlarmDefinition
	alarm72004Definition.AlarmId = alarm.E2_CONNECTION_PROBLEM
	alarm72004Definition.AlarmText = "E2 CONNECTION PROBLEM"
	alarm72004Definition.EventType = "Processing error"
	alarm72004Definition.OperationInstructions = "Not defined"
	alarm72004Definition.RaiseDelay = 0
	alarm72004Definition.ClearDelay = 0
	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm72004Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ = http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc = http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	//Get 72004 success
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestNewAlarmStoredAndPostedSucess(t *testing.T) {
	xapp.Logger.Info("TestNewAlarmStoredAndPostedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)

	var activeAlarms []AlarmNotification
	activeAlarms = make([]AlarmNotification, 1)
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/active", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc := http.HandlerFunc(alarmManager.GetActiveAlarms)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Decode the json output from handler
	json.NewDecoder(response.Body).Decode(activeAlarms)
	if len(activeAlarms) != 1 {
		t.Errorf("Incorrect alarm alarm count")
	}

	var alarmHistory []AlarmNotification
	alarmHistory = make([]AlarmNotification, 1)
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/history", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmHistory)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Decode the json output from handler
	json.NewDecoder(response.Body).Decode(alarmHistory)
	if len(alarmHistory) != 1 {
		t.Errorf("Incorrect alarm history count")
	}
}

func TestAlarmClearedSucess(t *testing.T) {
	xapp.Logger.Info("TestAlarmClearedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise the alarm
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)

	// Now Clear the alarm and check alarm is removed
	a = alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	//assert.Equal(t, len(alarmManager.activeAlarms), 0)
}

func TestMultipleAlarmsRaisedSucess(t *testing.T) {
	xapp.Logger.Info("TestMultipleAlarmsRaisedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	b := alarmer.NewAlarm(alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD, alarm.SeverityMinor, "Hello", "abcd 11")
	assert.Nil(t, alarmer.Raise(b), "raise failed")

	time.Sleep(time.Duration(5) * time.Second)

	xapp.Logger.Info("VerifyAlarm: %d %+v", len(alarmManager.activeAlarms), alarmManager.activeAlarms)
	VerifyAlarm(t, a, 1)
	xapp.Logger.Info("VerifyAlarm: %d %+v", len(alarmManager.activeAlarms), alarmManager.activeAlarms)
	VerifyAlarm(t, b, 2)
}

func TestMultipleAlarmsClearedSucess(t *testing.T) {
	xapp.Logger.Info("TestMultipleAlarmsClearedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	b := alarmer.NewAlarm(alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD, alarm.SeverityMinor, "Hello", "abcd 11")
	assert.Nil(t, alarmer.Clear(b), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), 0)
}

func TestAlarmsSuppresedSucess(t *testing.T) {
	xapp.Logger.Info("TestAlarmsSuppresedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two similar/matching alarms ... the second one suppresed
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)
	assert.Nil(t, alarmer.Clear(a), "clear failed")
}

func TestInvalidAlarms(t *testing.T) {
	xapp.Logger.Info("TestInvalidAlarms")
	a := alarmer.NewAlarm(1111, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	time.Sleep(time.Duration(2) * time.Second)
}

func TestAlarmHandlingErrorCases(t *testing.T) {
	xapp.Logger.Info("TestAlarmHandlingErrorCases")
	ok, err := alarmManager.HandleAlarms(&xapp.RMRParams{})
	assert.Equal(t, err.Error(), "unexpected end of JSON input")
	assert.Nil(t, ok, "raise failed")
}

func TestConsumeUnknownMessage(t *testing.T) {
	xapp.Logger.Info("TestConsumeUnknownMessage")
	err := alarmManager.Consume(&xapp.RMRParams{})
	assert.Nil(t, err, "raise failed")
}

func TestStatusCallback(t *testing.T) {
	xapp.Logger.Info("TestStatusCallback")
	assert.Equal(t, true, alarmManager.StatusCB())
}

func TestActiveAlarmMaxThresholds(t *testing.T) {
	xapp.Logger.Info("TestActiveAlarmMaxThresholds")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	alarmManager.maxActiveAlarms = 0
	alarmManager.maxAlarmHistory = 10

	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityCritical, "Some Application data", "eth 0 2")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	var alarmConfigParams alarm.AlarmConfigParams
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/config", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmConfig)
	response := executeRequest(req, handleFunc)

	// Check HTTP Status Code
	checkResponseCode(t, http.StatusOK, response.Code)

	// Decode the json output from handler
	json.NewDecoder(response.Body).Decode(&alarmConfigParams)
	if alarmConfigParams.MaxActiveAlarms != 0 || alarmConfigParams.MaxAlarmHistory != 10 {
		t.Errorf("Incorrect alarm thresholds")
	}

	time.Sleep(time.Duration(1) * time.Second)
	alarmManager.maxActiveAlarms = 5000
	alarmManager.maxAlarmHistory = 20000
	VerifyAlarm(t, a, 2)
	VerifyAlarm(t, a, 2)
	ts.Close()
}

func TestGetPrometheusAlerts(t *testing.T) {
	time.Sleep(1 * time.Second)
	xapp.Logger.Info("TestGetPrometheusAlerts")
	ts := CreatePromAlertSimulator2(t, "GET", "/alerts?active=true&inhibited=true&silenced=true&unprocessed=true")

	commandReady := make(chan bool, 1)
	command := "cli/alarm-cli"
	args := []string{"alerts", "--active", "true", "--inhibited", "true", "--silenced", "--unprocessed", "true", "true", "--host", "localhost", "--port", "9093", "flushall"}
	ExecCLICommand(commandReady, command, args...)
	<-commandReady

	ts.Close()
}

func TestDelayedAlarmRaiseAndClear(t *testing.T) {
	xapp.Logger.Info("TestDelayedAlarmRaiseAndClear")

	activeAlarmsBeforeTest := len(alarmManager.activeAlarms)
	alarmHistoryBeforeTest := len(alarmManager.alarmHistory)

	// Add new alarm definition
	var alarm9999Definition alarm.AlarmDefinition
	alarm9999Definition.AlarmId = 9999
	alarm9999Definition.AlarmText = "DELAYED TEST ALARM"
	alarm9999Definition.EventType = "Test type"
	alarm9999Definition.OperationInstructions = "Not defined"
	alarm9999Definition.RaiseDelay = 1
	alarm9999Definition.ClearDelay = 1
	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm9999Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Verify 9999 alarm definition
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise alarm. Posting alert and updating alarm history should be delayed
	a := alarmer.NewAlarm(9999, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	VerifyAlarm(t, a, activeAlarmsBeforeTest+1)

	// Clear the alarm and check the alarm is removed. Posting alert clear and updating alarm history should be delayed
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), activeAlarmsBeforeTest)
	assert.Equal(t, len(alarmManager.alarmHistory), alarmHistoryBeforeTest+2)
}

func TestDelayedAlarmRaiseAndClear2(t *testing.T) {
	xapp.Logger.Info("TestDelayedAlarmRaiseAndClear2")

	activeAlarmsBeforeTest := len(alarmManager.activeAlarms)
	alarmHistoryBeforeTest := len(alarmManager.alarmHistory)

	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms. The first should be delayed
	a := alarmer.NewAlarm(9999, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	VerifyAlarm(t, a, activeAlarmsBeforeTest+1)

	b := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(b), "raise failed")
	VerifyAlarm(t, b, activeAlarmsBeforeTest+2)

	// Clear two alarms. The first should be delayed. Check the alarms are removed
	assert.Nil(t, alarmer.Clear(a), "clear failed")
	assert.Nil(t, alarmer.Clear(b), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), activeAlarmsBeforeTest)
	assert.Equal(t, len(alarmManager.alarmHistory), alarmHistoryBeforeTest+4)
}

func TestDelayedAlarmRaiseAndClear3(t *testing.T) {
	xapp.Logger.Info("TestDelayedAlarmRaiseAndClear3")

	// Delete exisitng 9999 alarm definition
	req, _ := http.NewRequest("DELETE", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(9999, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Add updated 9999 alarm definition
	var alarm9999Definition alarm.AlarmDefinition
	alarm9999Definition.AlarmId = 9999
	alarm9999Definition.AlarmText = "DELAYED TEST ALARM"
	alarm9999Definition.EventType = "Test type"
	alarm9999Definition.OperationInstructions = "Not defined"
	alarm9999Definition.RaiseDelay = 1
	alarm9999Definition.ClearDelay = 0
	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm9999Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ = http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc = http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Verify 9999 alarm definition
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(72004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	activeAlarmsBeforeTest := len(alarmManager.activeAlarms)
	alarmHistoryBeforeTest := len(alarmManager.alarmHistory)

	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms. The first should be delayed
	a := alarmer.NewAlarm(9999, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	VerifyAlarm(t, a, activeAlarmsBeforeTest+1)

	b := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(b), "raise failed")
	VerifyAlarm(t, b, activeAlarmsBeforeTest+2)

	// Clear two alarms. The first should be delayed. Check the alarms are removed
	assert.Nil(t, alarmer.Clear(a), "clear failed")
	assert.Nil(t, alarmer.Clear(b), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), activeAlarmsBeforeTest)
	assert.Equal(t, len(alarmManager.alarmHistory), alarmHistoryBeforeTest+4)
}

func TestClearExpiredAlarms(t *testing.T) {
	xapp.Logger.Info("TestClearExpiredAlarms")

	a := alarm.AlarmMessage{
		Alarm:       alarmer.NewAlarm(72004, alarm.SeverityWarning, "threshold", ""),
		AlarmAction: alarm.AlarmActionRaise,
		AlarmTime:   time.Now().UnixNano(),
	}
	d := alarm.RICAlarmDefinitions[72004]
	n := AlarmNotification{a, *d}
	alarmManager.activeAlarms = make([]AlarmNotification, 0)
	alarmManager.UpdateActiveAlarmList(&n)

	// Unknown SP
	a.Alarm.SpecificProblem = 1234
	assert.False(t, alarmManager.ClearExpiredAlarms(n, 0, false), "ClearExpiredAlarms failed")

	// TTL is 0
	d.TimeToLive = 0
	assert.False(t, alarmManager.ClearExpiredAlarms(n, 0, false), "ClearExpiredAlarms failed")

	// TTL not expired
	a.Alarm.SpecificProblem = 72004
	d.TimeToLive = 2
	assert.False(t, alarmManager.ClearExpiredAlarms(n, 0, false), "ClearExpiredAlarms failed")

	// TTL expired, alarm should be cleared
	time.Sleep(time.Duration(3) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), 1)
	assert.True(t, alarmManager.ClearExpiredAlarms(n, 0, false), "ClearExpiredAlarms failed")
	assert.Equal(t, len(alarmManager.activeAlarms), 0)
}

func TestSetAlarmConfig(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmConfig")

	var setAlarmConfig alarm.AlarmConfigParams
	setAlarmConfig.MaxActiveAlarms = 500
	setAlarmConfig.MaxAlarmHistory = 2000

	pbodyEn, _ := json.Marshal(setAlarmConfig)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/config", bytes.NewBuffer(pbodyEn))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmConfig)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	var getAlarmConfig alarm.AlarmConfigParams
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/config", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmConfig)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Decode the json output from handler
	json.NewDecoder(response.Body).Decode(&getAlarmConfig)
	if getAlarmConfig.MaxActiveAlarms != 500 || getAlarmConfig.MaxAlarmHistory != 2000 {
		t.Errorf("Incorrect alarm thresholds")
	}

	// Revert ot default
	setAlarmConfig.MaxActiveAlarms = 5000
	setAlarmConfig.MaxAlarmHistory = 20000

	pbodyEn, _ = json.Marshal(setAlarmConfig)
	req, _ = http.NewRequest("POST", "/ric/v1/alarms/config", bytes.NewBuffer(pbodyEn))
	handleFunc = http.HandlerFunc(alarmManager.SetAlarmConfig)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("GET", "/ric/v1/alarms/config", nil)
	req = mux.SetURLVars(req, nil)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmConfig)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	// Decode the json output from handler
	json.NewDecoder(response.Body).Decode(&getAlarmConfig)
	if getAlarmConfig.MaxActiveAlarms != 5000 || getAlarmConfig.MaxAlarmHistory != 20000 {
		t.Errorf("Incorrect alarm thresholds")
	}
}

func TestConfigChangeCB(t *testing.T) {
	xapp.Logger.Info("TestConfigChangeCB")
	alarmManager.ConfigChangeCB("AlarmManager")
}

func TestPostAlarm(t *testing.T) {
	xapp.Logger.Info("TestPostAlarm")
	var activeAlarms []AlarmNotification
	activeAlarms = make([]AlarmNotification, 1)
	alarmManager.PostAlarm(&activeAlarms[0])
}

func TestPostAlarm1(t *testing.T) {
	xapp.Logger.Info("TestPostAlarm")
	var activeAlarms []AlarmNotification
	activeAlarms = make([]AlarmNotification, 2)
	alarmManager.PostAlarm(&activeAlarms[0])
}

func TestNewAlarmManagerOther(t *testing.T) {
	NewAlarmManager("", 0, true)
}

func TestStatusCallbackFailure(t *testing.T) {
	xapp.Logger.Info("TestStatusCallbackFailure")
	alarmManager.rmrReady = false
	assert.Equal(t, false, alarmManager.StatusCB())
}

func TestConfigChangeCBFailure(t *testing.T) {
	xapp.Logger.Info("TestConfigChangeCBFailure")
	alarmManager.maxActiveAlarms = 0
	alarmManager.maxAlarmHistory = 0
	alarmManager.ConfigChangeCB("AlarmManager")
}

func TestReadAlarmDefinitionFromJsonWrongFilename(t *testing.T) {
	// use   to set wrong file name os.Setenv("SITE_TITLE", "Test Site")
	xapp.Logger.Info("TestReadAlarmDefinitionFromJsonWrongFilename")
	os.Setenv("DEF_FILE", "test.json")
	alarmManager.ReadAlarmDefinitionFromJson()
	// correct the filename
}

func TestReadAlarmDefinitionFromJsonInvalidFilename(t *testing.T) {
	// use   to set wrong file name os.Setenv("SITE_TITLE", "Test Site")
	xapp.Logger.Info("TestReadAlarmDefinitionFromJsonInvalidFilename")
	os.Setenv("DEF_FILE", "../../definitions/test.json")
	alarmManager.ReadAlarmDefinitionFromJson()
	// correct the filename
}

func TestPersistentStorage(t *testing.T) {
	xapp.Logger.Info("TestPersistentStorage")
	alarmManager.alarmInfoPvFile = "../../definitions/sample.json"
	alarmManager.ReadAlarmInfoFromPersistentVolume()
}

func TestDeleteAlarmDefinitions1(t *testing.T) {
	xapp.Logger.Info("TestDeleteAlarmDefinitions1")
	//Get all
	//Delete Alarm which doesn't present
	req, _ := http.NewRequest("DELETE", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(882004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	//Delete Alarm which is incorrect present
	req, _ = http.NewRequest("DELETE", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(898989, 8)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestGetPreDefinedAlarmInvalidAlarm(t *testing.T) {
	xapp.Logger.Info("TestGetPreDefinedAlarmInvalidAlarm")
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": "asdsc"}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	xapp.Logger.Info("response code = %v", response.Code)
	checkResponseCode(t, http.StatusBadRequest, response.Code)
}

func TestDeleteAlarmDefinitions2(t *testing.T) {
	xapp.Logger.Info("TestDeleteAlarmDefinitions2")
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	//Giving Wrong alarmId which can't convert into int
	vars := map[string]string{"alarmId": "asdsc"}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusBadRequest, response.Code)
}

func VerifyAlarm(t *testing.T, a alarm.Alarm, expectedCount int) string {
	receivedAlert := waitForEvent()

	assert.Equal(t, expectedCount, len(alarmManager.activeAlarms))
	_, ok := alarmManager.IsMatchFound(a)
	assert.True(t, ok)

	return receivedAlert
}

func VerifyAlert(t *testing.T, receivedAlert, expectedAlert string) {
	receivedAlert = strings.Replace(fmt.Sprintf("%v", receivedAlert), "\r\n", " ", -1)
	//assert.Equal(t, receivedAlert, e)
}

func CreatePromAlertSimulator(t *testing.T, method, url string, status int, respData interface{}) *httptest.Server {
	l, err := net.Listen("tcp", "localhost:9093")
	if err != nil {
		t.Error("Failed to create listener: " + err.Error())
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Contains(r.URL.String(), "active") {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(200)
			// Read alerts from file
			payload, err := readJSONFromFile("../../testresources/prometheus-alerts.json")
			if err != nil {
				t.Error("Failed to send response: ", err)
			}
			_, err = w.Write(payload)
			if err != nil {
				t.Error("Failed to send response: " + err.Error())
			}
			return
		}

		assert.Equal(t, r.Method, method)
		assert.Equal(t, r.URL.String(), url)

		fireEvent(t, r.Body)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		b, _ := json.Marshal(respData)
		w.Write(b)
	}))
	ts.Listener.Close()
	ts.Listener = l

	ts.Start()

	return ts
}

func CreatePromAlertSimulator2(t *testing.T, method, url string) *httptest.Server {
	l, err := net.Listen("tcp", "localhost:9093")
	if err != nil {
		t.Error("Failed to create listener: " + err.Error())
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, method)
		assert.Equal(t, r.URL.String(), url)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		// Read alerts from file
		payload, err := readJSONFromFile("../testresources/prometheus-alerts.json")
		if err != nil {
			t.Error("Failed to send response: ", err)
		}
		_, err = w.Write(payload)
		if err != nil {
			t.Error("Failed to send response: " + err.Error())
		}
	}))
	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	return ts
}

func waitForEvent() string {
	receivedAlert := <-eventChan
	return receivedAlert
}

func fireEvent(t *testing.T, body io.ReadCloser) {
	reqBody, err := ioutil.ReadAll(body)
	assert.Nil(t, err, "ioutil.ReadAll failed")
	assert.NotNil(t, reqBody, "ioutil.ReadAll failed")

	eventChan <- fmt.Sprintf("%s", reqBody)
}

func executeRequest(req *http.Request, handleR http.HandlerFunc) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()

	handleR.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) bool {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
		return false
	}
	return true
}

func ExecCLICommand(commandReady chan bool, command string, args ...string) {
	go func() {
		xapp.Logger.Info("Giving CLI command")
		cmd := exec.Command(command, args...)
		cmd.Dir = "../"
		output, err := cmd.CombinedOutput()
		if err != nil {
			xapp.Logger.Info("CLI command failed out: %s", err)
		}
		xapp.Logger.Info("CLI command output: %s", output)
		commandReady <- true
		xapp.Logger.Info("CLI command completed")
	}()
}

func readJSONFromFile(filename string) ([]byte, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		err := fmt.Errorf("readJSONFromFile() failed: Error: %v", err)
		return nil, err
	}
	return file, nil
}
