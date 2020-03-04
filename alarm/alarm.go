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

package alarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"
)

/*
#cgo CFLAGS: -I../
#cgo LDFLAGS: -lrmr_nng -lnng

#include "utils.h"
*/
import "C"

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
	ManagedObjectId   string   `json:"managedObjectId"`
	ApplicationId     string   `json:"applicationId"`
	SpecificProblem   int      `json:"specificProblem"`
	PerceivedSeverity Severity `json:"perceivedSeverity"`
	AdditionalInfo    string   `json:"additionalInfo"`
	IdentifyingInfo   string   `json:"identifyingInfo"`
}

// Alarm actions
type AlarmAction string

// Possible values for alarm actions
const (
	AlarmActionRaise    AlarmAction = "RAISE"
	AlarmActionClear    AlarmAction = "CLEAR"
	AlarmActionReraise  AlarmAction = "RERAISE"
	AlarmActionClearAll AlarmAction = "CLEARALL"
)

type AlarmMessage struct {
	Alarm
	AlarmAction
	AlarmTime int64
}

// RICAlarm is an alarm instance
type RICAlarm struct {
	moId   string
	appId  string
	rmrCtx unsafe.Pointer
	mutex  sync.Mutex
}

// InitAlarm is the init routine which returns a new alarm instance.
// The MO and APP identities are given as a parameters.
// The identities are used when raising/clearing alarms, unless provided by the applications.
func InitAlarm(mo, id string) (*RICAlarm, error) {
	if ctx := C.rmrInit(); ctx != nil {
		r := &RICAlarm{
			moId:   mo,
			appId:  id,
			rmrCtx: ctx,
		}

		return r, nil
	}

	return nil, errors.New("rmrInit failed!")
}

// Create a new Alarm instance
func (r *RICAlarm) NewAlarm(sp int, severity Severity, ainfo, iinfo string) Alarm {
	return Alarm{
		ManagedObjectId:   r.moId,
		ApplicationId:     r.appId,
		SpecificProblem:   sp,
		PerceivedSeverity: severity,
		AdditionalInfo:    ainfo,
		IdentifyingInfo:   iinfo,
	}
}

// Create a new AlarmMessage instance
func (r *RICAlarm) NewAlarmMessage(a Alarm, alarmAction AlarmAction) AlarmMessage {
	alarmTime := time.Now().UnixNano() / 1000
	return AlarmMessage{a, alarmAction, alarmTime}
}

// Raise a RIC alarm
func (r *RICAlarm) Raise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionRaise)
	return r.SendMessage(m)
}

// Clear a RIC alarm
func (r *RICAlarm) Clear(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionClear)
	return r.SendMessage(m)
}

// Re-raise a RIC alarm
func (r *RICAlarm) Reraise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionReraise)
	return r.SendMessage(m)
}

// Clear all alarms raised by the application
func (r *RICAlarm) ClearAll() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	a := r.NewAlarm(0, SeverityDefault, "", "")
	m := r.NewAlarmMessage(a, AlarmActionClearAll)

	return r.SendMessage(m)
}

// Internal functions
func (r *RICAlarm) AlarmString(a AlarmMessage) string {
	s := "MOId=%s AppId=%s SP=%d severity=%s IA=%s"
	return fmt.Sprintf(s, a.ManagedObjectId, a.ApplicationId, a.SpecificProblem, a.PerceivedSeverity, a.IdentifyingInfo)
}

func (r *RICAlarm) SendMessage(a AlarmMessage) error {
	log.Println("Sending alarm:", r.AlarmString(a))

	payload, err := json.Marshal(a)
	if err != nil {
		return err
	}

	datap := C.CBytes(payload)
	defer C.free(datap)
	meid := C.CString("ric")
	defer C.free(unsafe.Pointer(meid))

	if state := C.rmrSend(r.rmrCtx, 1234, datap, C.int(len(payload)), meid); state != C.RMR_OK {
		return errors.New(fmt.Sprintf("rmrSend failed with error: %d", state))
	}
	return nil
}

func (r *RICAlarm) ReceiveMessage(cb func(AlarmMessage)) error {
	if rbuf := C.rmrRcv(r.rmrCtx); rbuf != nil {
		payload := C.GoBytes(unsafe.Pointer(rbuf.payload), C.int(rbuf.len))
		a := AlarmMessage{}
		if err := json.Unmarshal(payload, &a); err == nil {
			cb(a)
		}
	}
	return errors.New("rmrRcv failed!")
}
