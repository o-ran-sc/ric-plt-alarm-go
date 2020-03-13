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

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	"gerrit.o-ran-sc.org/r/ric-plt/xapp-frame/pkg/xapp"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var alarmAdapter *AlarmAdapter
var alarmer *alarm.RICAlarm
var eventChan chan string

// Test cases
func TestMain(M *testing.M) {
	alarmAdapter = NewAlarmAdapter("localhost:9093", 500)
	go alarmAdapter.Run(false)
	time.Sleep(time.Duration(2) * time.Second)

	// Wait until RMR is up-and-running
	for !xapp.Rmr.IsReady() {
		time.Sleep(time.Duration(1) * time.Second)
	}

	alarmer, _ = alarm.InitAlarm("my-pod", "my-app")
	eventChan = make(chan string)

	os.Exit(M.Run())
}

func TestNewAlarmStoredAndPostedSucess(t *testing.T) {
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)
}

func TestAlarmClearedSucess(t *testing.T) {
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise the alarm
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)

	// Now Clear the alarm and check alarm is removed
	a = alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityCleared, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmAdapter.activeAlarms), 0)
}

func TestMultipleAlarmsRaisedSucess(t *testing.T) {
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
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two alarms
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Clear(a), "clear failed")

	b := alarmer.NewAlarm(alarm.TCP_CONNECTIVITY_LOST_TO_DBAAS, alarm.SeverityMinor, "Hello", "abcd 11")
	assert.Nil(t, alarmer.Clear(b), "clear failed")

	time.Sleep(time.Duration(2) * time.Second)
	assert.Equal(t, len(alarmAdapter.activeAlarms), 0)
}

func TestAlarmsSuppresedSucess(t *testing.T) {
	ts := CreatePromAlertSimulator(t, "POST", "/api/v2/alerts", http.StatusOK, models.LabelSet{})
	defer ts.Close()

	// Raise two similar/matching alarms ... the second one suppresed
	a := alarmer.NewAlarm(alarm.RIC_RT_DISTRIBUTION_FAILED, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	assert.Nil(t, alarmer.Raise(a), "raise failed")

	VerifyAlarm(t, a, 1)
}

func TestInvalidAlarms(t *testing.T) {
	a := alarmer.NewAlarm(1111, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Nil(t, alarmer.Raise(a), "raise failed")
	time.Sleep(time.Duration(2) * time.Second)
}

func TestAlarmHandlingErrorCases(t *testing.T) {
	ok, err := alarmAdapter.HandleAlarms(&xapp.RMRParams{})
	assert.Equal(t, err.Error(), "unexpected end of JSON input")
	assert.Nil(t, ok, "raise failed")
}

func TestConsumeUnknownMessage(t *testing.T) {
	err := alarmAdapter.Consume(&xapp.RMRParams{})
	assert.Nil(t, err, "raise failed")
}

func TestStatusCallback(t *testing.T) {
	assert.Equal(t, true, alarmAdapter.StatusCB())
}

func VerifyAlarm(t *testing.T, a alarm.Alarm, expectedCount int) string {
	receivedAlert := waitForEvent()

	assert.Equal(t, len(alarmAdapter.activeAlarms), expectedCount)
	_, ok := alarmAdapter.IsMatchFound(a)
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
