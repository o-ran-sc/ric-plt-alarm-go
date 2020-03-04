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
	"fmt"
	"log"
	"sync"
)

// Severity for alarms
type Severity string

// Possible values for Severity
const (
	SeverityUnspecified Severity = "UNSPECIFIED"
	SeverityCritical    Severity = "CRITICAL"
	SeverityMajor       Severity = "MAJOR"
	SeverityMinor       Severity = "MINOR"
	SeverityWarning     Severity = "WARNING"
	SeverityNormal      Severity = "CLEARED"
	SeverityDefault     Severity = "DEFAULT"
)

// Alarm object - see README for more information
type Alarm struct {
	SpecificProblem   int      `json:"specificProblem"`
	PerceivedSeverity Severity `json:"perceivedSeverity"`
	ManagedObjectId   string   `json:"managedObjectId"`
	ApplicationId     string   `json:"applicationId"`
	AdditionalInfo    string   `json:"additionalInfo"`
	IdentifyingInfo   string   `json:"identifyingInfo"`
}

// RICAlarm is an alarm instance
type RICAlarm struct {
	moId  string
	appId string
	mutex sync.Mutex
}

// InitAlarm is the init routine which returns a new alarm instance.
// The MO and APP identities are given as a parameters.
// The identities are used when raising/clearing alarms, unless provided by the applications.
func InitAlarm(mo, id string) (*RICAlarm, error) {
	return &RICAlarm{moId: mo, appId: id}, nil
}

// Raise a RIC alarm
func (r *RICAlarm) RaiseAlarm(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Println("RaiseAlarm: alarm raised:", r.AlarmString(a))
	return nil
}

// Clear a RIC alarm
func (r *RICAlarm) ClearAlarm(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Println("ClearAlarm: alarm cleared:", r.AlarmString(a))
	return nil
}

// Re-raise a RIC alarm
func (r *RICAlarm) ReraiseAlarm(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Println("ReraiseAlarm: alarm re-raised:", r.AlarmString(a))
	return nil
}

func (r *RICAlarm) AlarmString(a Alarm) string {
	s := "MOId=%s AppId=%s SP=%d severity=%s IA=%s"
	return fmt.Sprintf(s, a.ManagedObjectId, a.ApplicationId, a.SpecificProblem, a.PerceivedSeverity, a.IdentifyingInfo)
}
