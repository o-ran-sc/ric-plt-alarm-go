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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
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

// Raise a RIC alarm
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

// Raise a RIC alarm
func (r *RICAlarm) Raise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.SendMessage(a)
}

// Clear a RIC alarm
func (r *RICAlarm) Clear(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.SendMessage(a)
}

// Re-raise a RIC alarm
func (r *RICAlarm) Reraise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.SendMessage(a)
}

// Internal functions
func (r *RICAlarm) AlarmString(a Alarm) string {
	s := "MOId=%s AppId=%s SP=%d severity=%s IA=%s"
	return fmt.Sprintf(s, a.ManagedObjectId, a.ApplicationId, a.SpecificProblem, a.PerceivedSeverity, a.IdentifyingInfo)
}

func (r *RICAlarm) SendMessage(a Alarm) error {
	log.Println("Sending alarm:", r.AlarmString(a))

	payload, err := json.Marshal(a)
	if err != nil {
		return err
	}

	datap := C.CBytes(payload)
	defer C.free(datap)
	meid := C.CString("gnb-123")
	defer C.free(unsafe.Pointer(meid))

	if state := C.rmrSend(r.rmrCtx, 1234, datap, C.int(len(payload)), meid); state != C.RMR_OK {
		return errors.New(fmt.Sprintf("rmrSend failed with error: %d", state))
	}
	return nil
}

func (r *RICAlarm) ReceiveMessage(cb func(Alarm)) error {
	if rbuf := C.rmrRcv(r.rmrCtx); rbuf != nil {
		payload := C.GoBytes(unsafe.Pointer(rbuf.payload), C.int(rbuf.len))
		a := Alarm{}
		if err := json.Unmarshal(payload, &a); err == nil {
			cb(a)
		}
	}
	return errors.New("rmrRcv failed!")
}
