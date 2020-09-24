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
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
        "github.com/gorilla/mux"
        "strconv"
	"bytes"
	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var alarmManager *AlarmManager
var alarmer *alarm.RICAlarm
var eventChan chan string

// Test cases
func TestMain(M *testing.M) {
	os.Setenv("ALARM_IF_RMR", "true")
	alarmManager = NewAlarmManager("localhost:9093", 500)
	go alarmManager.Run(false)
	time.Sleep(time.Duration(2) * time.Second)

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

func TestSetAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestSetAlarmDefinitions")
	var alarm8004Definition alarm.AlarmDefinition
	alarm8004Definition.AlarmId = alarm.RIC_RT_DISTRIBUTION_FAILED
	alarm8004Definition.AlarmText = "RIC ROUTING TABLE DISTRIBUTION FAILED"
	alarm8004Definition.EventType = "Processing error"
	alarm8004Definition.OperationInstructions = "Not defined"

	var alarm8005Definition alarm.AlarmDefinition
	alarm8005Definition.AlarmId = alarm.TCP_CONNECTIVITY_LOST_TO_DBAAS
	alarm8005Definition.AlarmText = "TCP CONNECTIVITY LOST TO DBAAS"
	alarm8005Definition.EventType = "Communication error"
	alarm8005Definition.OperationInstructions = "Not defined"

	var alarm8006Definition alarm.AlarmDefinition
	alarm8006Definition.AlarmId = alarm.E2_CONNECTIVITY_LOST_TO_GNODEB
	alarm8006Definition.AlarmText = "E2 CONNECTIVITY LOST TO G-NODEB"
	alarm8006Definition.EventType = "Communication error"
	alarm8006Definition.OperationInstructions = "Not defined"

	var alarm8007Definition alarm.AlarmDefinition
	alarm8007Definition.AlarmId = alarm.E2_CONNECTIVITY_LOST_TO_ENODEB
	alarm8007Definition.AlarmText = "E2 CONNECTIVITY LOST TO E-NODEB"
	alarm8007Definition.EventType = "Communication error"
	alarm8007Definition.OperationInstructions = "Not defined"

	var alarm8008Definition alarm.AlarmDefinition
	alarm8008Definition.AlarmId = alarm.ACTIVE_ALARM_EXCEED_MAX_THRESHOLD
	alarm8008Definition.AlarmText = "ACTIVE ALARM EXCEED MAX THRESHOLD"
	alarm8008Definition.EventType = "storage warning"
	alarm8008Definition.OperationInstructions = "clear alarms or raise threshold"

	var alarm8009Definition alarm.AlarmDefinition
	alarm8009Definition.AlarmId = alarm.ALARM_HISTORY_EXCEED_MAX_THRESHOLD
	alarm8009Definition.AlarmText = "ALARM HISTORY EXCEED MAX THRESHOLD"
	alarm8009Definition.EventType = "storage warning"
	alarm8009Definition.OperationInstructions = "clear alarms or raise threshold"

	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm8004Definition, &alarm8005Definition, &alarm8006Definition, &alarm8007Definition, &alarm8008Definition, &alarm8009Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ := http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc := http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	status := checkResponseCode(t, http.StatusOK, response.Code)
	xapp.Logger.Info("status = %v", status)

}

func TestGetAlarmDefinitions(t *testing.T) {
	xapp.Logger.Info("TestGetAlarmDefinitions")
	var alarmDefinition alarm.AlarmDefinition
	req, _ := http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars := map[string]string{"alarmId": strconv.FormatUint(8004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc := http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response := executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.NewDecoder(response.Body).Decode(&alarmDefinition)
	xapp.Logger.Info("alarm definition = %v", alarmDefinition)
	if alarmDefinition.AlarmId != alarm.RIC_RT_DISTRIBUTION_FAILED || alarmDefinition.AlarmText != "RIC ROUTING TABLE DISTRIBUTION FAILED" {
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

	//Delete 8004
	req, _ = http.NewRequest("DELETE", "/ric/v1/alarms/define", nil)
        vars := map[string]string{"alarmId": strconv.FormatUint(8004, 10)}
        req = mux.SetURLVars(req, vars)
        handleFunc = http.HandlerFunc(alarmManager.DeleteAlarmDefinition)
        response = executeRequest(req, handleFunc)
        checkResponseCode(t, http.StatusOK, response.Code)

	//Get 8004 fail
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(8004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusBadRequest, response.Code)

	//Set 8004 success
	var alarm8004Definition alarm.AlarmDefinition
	alarm8004Definition.AlarmId = alarm.RIC_RT_DISTRIBUTION_FAILED
	alarm8004Definition.AlarmText = "RIC ROUTING TABLE DISTRIBUTION FAILED"
	alarm8004Definition.EventType = "Processing error"
	alarm8004Definition.OperationInstructions = "Not defined"
	pbodyParams := RicAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarm8004Definition}}
	pbodyEn, _ := json.Marshal(pbodyParams)
	req, _ = http.NewRequest("POST", "/ric/v1/alarms/define", bytes.NewBuffer(pbodyEn))
	handleFunc = http.HandlerFunc(alarmManager.SetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)

	//Get 8004 success
	req, _ = http.NewRequest("GET", "/ric/v1/alarms/define", nil)
	vars = map[string]string{"alarmId": strconv.FormatUint(8004, 10)}
	req = mux.SetURLVars(req, vars)
	handleFunc = http.HandlerFunc(alarmManager.GetAlarmDefinition)
	response = executeRequest(req, handleFunc)
	checkResponseCode(t, http.StatusOK, response.Code)
}

func TestNewAlarmStoredAndPostedSucess(t *testing.T) {
	xapp.Logger.Info("TestNewAlarmStoredAndPostedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)
}

func TestAlarmClearedSucess(t *testing.T) {
	xapp.Logger.Info("TestAlarmClearedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise the alarm
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)

	// Now Clear the alarm and check alarm is removed
	a = alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityCritical, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), 0)
}

func TestMultipleAlarmsRaisedSucess(t *testing.T) {
	xapp.Logger.Info("TestMultipleAlarmsRaisedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	b := alarmer.NewAlarm(alarm.TCP_CONNECTIVITY_LOST_TO_DBAAS, alarm.SeverityMinor, "Hello", "abcd 11")
	assert.Nil(t, alarmer.Raise(b), "raise failed")

	VerifyAlarm(t, a, 2)
	VerifyAlarm(t, b, 2)
}

func TestMultipleAlarmsClearedSucess(t *testing.T) {
	xapp.Logger.Info("TestMultipleAlarmsClearedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	b := alarmer.NewAlarm(alarm.TCP_CONNECTIVITY_LOST_TO_DBAAS, alarm.SeverityMinor, "Hello", "abcd 11")
	assert.Nil(t, alarmer.Clear(b), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmManager.activeAlarms), 0)
}

func TestAlarmsSuppresedSucess(t *testing.T) {
	xapp.Logger.Info("TestAlarmsSuppresedSucess")
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two similar/matching alarms ... the second one suppresed
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
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

	a := alarmer.NewAlarm(alarm.E2_CONNECTIVITY_LOST_TO_GNODEB, alarm.SeverityCritical, "Some Application data", "eth 0 2")
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

func VerifyAlarm(t *testing.T, a alarm.Alarm, expectedCount int) string {
	receivedAlert := waitForEvent()

	assert.Equal(t, len(alarmManager.activeAlarms), expectedCount)
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
