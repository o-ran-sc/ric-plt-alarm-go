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

package alarm_test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go.git/alarm"
)

var alarmer *alarm.RICAlarm

// Test cases
func TestAlarmInitSuccess(t *testing.T) {
	a, err := alarm.InitAlarm("my-pod", "my-app")
	assert.Nil(t, err, "init failed")
	assert.Equal(t, false, a == nil)

	alarmer = a
	time.Sleep(time.Duration(5 * time.Second))
}

func TestAlarmRaiseSuccess(t *testing.T) {
	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	err := alarmer.Raise(a)
	assert.Nil(t, err, "raise failed")
}

func TestAlarmClearSuccess(t *testing.T) {
	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	err := alarmer.Clear(a)
	assert.Nil(t, err, "clear failed")
}

func TestAlarmReraiseSuccess(t *testing.T) {
	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	err := alarmer.Reraise(a)
	assert.Nil(t, err, "re-raise failed")
}

func TestAlarmClearAllSuccess(t *testing.T) {
	err := alarmer.ClearAll()
	assert.Nil(t, err, "clearAll failed")
}

func TestAlarmSendSuccess(t *testing.T) {
	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	consumer := func(m alarm.AlarmMessage) {
		assert.Equal(t, m.ManagedObjectId, a.ManagedObjectId)
		assert.Equal(t, m.ApplicationId, a.ApplicationId)
		assert.Equal(t, m.SpecificProblem, a.SpecificProblem)
		assert.Equal(t, m.PerceivedSeverity, a.PerceivedSeverity)
		assert.Equal(t, m.AdditionalInfo, a.AdditionalInfo)
		assert.Equal(t, m.IdentifyingInfo, a.IdentifyingInfo)
		assert.Equal(t, m.AlarmAction, alarm.AlarmActionRaise)
	}

	go alarmer.ReceiveMessage(consumer)
	time.Sleep(time.Duration(1 * time.Second))

	err := alarmer.Raise(a)
	assert.Nil(t, err, "send failed")
}

func TestSetManagedObjectIdSuccess(t *testing.T) {
	alarmer.SetManagedObjectId("new-pod")

	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Equal(t, a.ManagedObjectId, "new-pod")
}

func TestSetApplicationIdSuccess(t *testing.T) {
	alarmer.SetApplicationId("new-app")

	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")
	assert.Equal(t, a.ApplicationId, "new-app")
}
