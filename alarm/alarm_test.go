/*
 *  Copyright (c) 2019 AT&T Intellectual Property.
 *  Copyright (c) 2018-2019 Nokia.
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

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
)

var alarmer *alarm.RICAlarm

// Test cases
func TestAlarmInitSuccess(t *testing.T) {
	a, err := alarm.InitAlarm("dummy-pod", "dummy-xapp")
	assert.Nil(t, err, "init failed")
	assert.Equal(t, false, a == nil)

	alarmer = a
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

func TestAlarmSendSuccess(t *testing.T) {
	a := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	consumer := func(data alarm.Alarm) {
		assert.Equal(t, data, a)
	}

	go alarmer.ReceiveMessage(consumer)
	time.Sleep(time.Duration(1 * time.Second))

	err := alarmer.SendMessage(a)
	assert.Nil(t, err, "send failed")
}
