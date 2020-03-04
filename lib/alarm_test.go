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

package alarm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAlarmInitSuccess(t *testing.T) {
	_, err := InitAlarm("ric", "dummy-xapp")
	assert.Nil(t, err, "init failed")
}

func TestAlarmRaiseSuccess(t *testing.T) {
	a := Alarm{
		SpecificProblem:   1234,
		PerceivedSeverity: SeverityMajor,
		AdditionalInfo:    "Some application specific data",
		IdentifyingInfo:   "eth 0 1",
	}

	alarmer, err := InitAlarm("ric", "dummy-xapp")
	assert.Nil(t, err, "init failed")

	err = alarmer.RaiseAlarm(a)
	assert.Nil(t, err, "raise failed")
}

func TestAlarmClearSuccess(t *testing.T) {
	a := Alarm{
		SpecificProblem:   1234,
		PerceivedSeverity: SeverityMajor,
		AdditionalInfo:    "Some application specific data",
		IdentifyingInfo:   "eth 0 1",
	}

	alarmer, err := InitAlarm("ric", "dummy-xapp")
	assert.Nil(t, err, "init failed")

	err = alarmer.ClearAlarm(a)
	assert.Nil(t, err, "clear failed")
}

func TestAlarmReraiseSuccess(t *testing.T) {
	a := Alarm{
		SpecificProblem:   1234,
		PerceivedSeverity: SeverityMajor,
		AdditionalInfo:    "Some application specific data",
		IdentifyingInfo:   "eth 0 1",
	}

	alarmer, err := InitAlarm("ric", "dummy-xapp")
	assert.Nil(t, err, "init failed")

	err = alarmer.ReraiseAlarm(a)
	assert.Nil(t, err, "re-raise failed")
}
