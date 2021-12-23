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
	"net/http"
	"net/http/httptest"
	"testing"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go.git/alarm"
	"github.com/stretchr/testify/assert"
)

// Test cases
func TestGetActiveAlarmsRESTInterface(t *testing.T) {
	req, err := http.NewRequest("GET", "/ric/v1/alarms", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(alarmManager.GetActiveAlarms)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, true, rr != nil)
	assert.Equal(t, rr.Code, http.StatusOK)
}

func TestRaiseAlarmRESTInterface(t *testing.T) {
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	b, err := json.Marshal(&a)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	req, err := http.NewRequest("POST", "/ric/v1/alarms", bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(alarmManager.RaiseAlarm)
	handler.ServeHTTP(rr, req)

	assert.True(t, rr != nil)
	assert.Equal(t, rr.Code, http.StatusOK)
}

func TestClearAlarmRESTInterface(t *testing.T) {
	a := alarmer.NewAlarm(alarm.E2_CONNECTION_PROBLEM, alarm.SeverityMajor, "Some App data", "eth 0 1")
	b, err := json.Marshal(&a)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	req, err := http.NewRequest("POST", "/ric/v1/alarms", bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(alarmManager.ClearAlarm)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, true, rr != nil)
	assert.Equal(t, rr.Code, http.StatusOK)
}

func TestSymptomDataHandler(t *testing.T) {
	req, err := http.NewRequest("POST", "/ric/v1/symptomdata", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(alarmManager.SymptomDataHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, true, rr != nil)
	assert.Equal(t, rr.Code, http.StatusOK)
}
