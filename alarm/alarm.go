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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"unsafe"
)

/*
#cgo CFLAGS: -I../
#cgo LDFLAGS: -lrmr_si

#include "utils.h"
*/
import "C"

// InitAlarm is the init routine which returns a new alarm instance.
// The MO and APP identities are given as a parameters.
// The identities are used when raising/clearing alarms, unless provided by the applications.
func InitAlarm(mo, id string) (*RICAlarm, error) {
	r := &RICAlarm{
		moId:        mo,
		appId:       id,
		managerUrl:  ALARM_MANAGER_HTTP_URL,
		rmrEndpoint: ALARM_MANAGER_RMR_URL,
	}

	if os.Getenv("ALARM_MANAGER_URL") != "" {
		r.managerUrl = os.Getenv("ALARM_MANAGER_URL")
	} else if namespace != "" {
		r.managerUrl = ALARM_MANAGER_HTTP_URL
	} else {
		r.managerUrl = "http://127.0.0.1:8080"
	}

	if os.Getenv("ALARM_MANAGER_SERVICE_NAME") != "" && os.Getenv("ALARM_MANAGER_SERVICE_PORT") != "" {
		r.rmrEndpoint = fmt.Sprintf("%s:%s", os.Getenv("ALARM_MANAGER_SERVICE_NAME"), os.Getenv("ALARM_MANAGER_SERVICE_PORT"))
	}

	if os.Getenv("ALARM_IF_RMR") == "" {
		if r.moId == "my-pod" {
			r.rmrEndpoint = "127.0.0.1:4560"
		} else if r.moId == "my-pod-lib" {
			r.rmrEndpoint = "127.0.0.1:4588"
		}
	}

	go InitRMR(r)

	return r, nil
}

// Create a new Alarm instance
func (r *RICAlarm) NewAlarm(sp int, severity Severity, ainfo, iinfo string) Alarm {
	return Alarm{
		ManagedObjectId:   r.moId,
		ApplicationId:     r.appId,
		SpecificProblem:   sp,
		PerceivedSeverity: severity,
		IdentifyingInfo:   iinfo,
		AdditionalInfo:    ainfo,
	}
}

// Create a new AlarmMessage instance
func (r *RICAlarm) NewAlarmMessage(a Alarm, alarmAction AlarmAction) AlarmMessage {
	alarmTime := time.Now().UnixNano()
	return AlarmMessage{a, alarmAction, alarmTime}
}

func (r *RICAlarm) SetManagedObjectId(mo string) {
	r.moId = mo
}

func (r *RICAlarm) SetApplicationId(app string) {
	r.appId = app
}

// Raise a RIC alarm
func (r *RICAlarm) Raise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionRaise)
	return r.sendAlarmUpdateReq(m)
}

// Clear a RIC alarm
func (r *RICAlarm) Clear(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionClear)
	return r.sendAlarmUpdateReq(m)
}

// Re-raise a RIC alarm
func (r *RICAlarm) Reraise(a Alarm) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m := r.NewAlarmMessage(a, AlarmActionClear)
	if err := r.sendAlarmUpdateReq(m); err != nil {
		return errors.New(fmt.Sprintf("Reraise failed: %v", err))
	}

	return r.sendAlarmUpdateReq(r.NewAlarmMessage(a, AlarmActionRaise))
}

// Clear all alarms raised by the application
func (r *RICAlarm) ClearAll() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	a := r.NewAlarm(0, SeverityDefault, "", "")
	m := r.NewAlarmMessage(a, AlarmActionClearAll)

	return r.sendAlarmUpdateReq(m)
}

func (r *RICAlarm) AlarmString(a AlarmMessage) string {
	s := "MOId=%s AppId=%s SP=%d severity=%s IA=%s"
	return fmt.Sprintf(s, a.ManagedObjectId, a.ApplicationId, a.SpecificProblem, a.PerceivedSeverity, a.IdentifyingInfo)
}

func (r *RICAlarm) sendAlarmUpdateReqWithHttp(payload []byte) error {
	url := fmt.Sprintf("%s/%s", r.managerUrl, "ric/v1/alarms")
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil || resp == nil {
		return fmt.Errorf("HttpError=Post failed with error: %v", err)
	}
	log.Printf("Alarm posted to %s [status=%d]", url, resp.StatusCode)
	return nil
}

func (r *RICAlarm) sendAlarmUpdateReqWithRmr(payload []byte) error {
	if r.rmrCtx == nil || !r.rmrReady {
		return fmt.Errorf("RmrError=rmr not ready")
	}
	datap := C.CBytes(payload)
	defer C.free(datap)
	meid := C.CString("ric")
	defer C.free(unsafe.Pointer(meid))

	if state := C.rmrSend(r.rmrCtx, RIC_ALARM_UPDATE, datap, C.int(len(payload)), meid); state != C.RMR_OK {
		return errors.New(fmt.Sprintf("RmrError=rmrSend failed with error: %d", state))
	}
	log.Printf("Alarm sent via rmr to %s", r.rmrEndpoint)
	return nil
}

func (r *RICAlarm) sendAlarmUpdateReq(a AlarmMessage) error {

	payload, err := json.Marshal(a)
	if err != nil {
		log.Println("json.Marshal failed with error: ", err)
		return err
	}
	log.Println("Sending alarm: ", fmt.Sprintf("%s", payload))

	// --
	// Try rmr sending
	// --
	err = r.sendAlarmUpdateReqWithRmr(payload)

	// --
	// Try http posting if rmr is not done for some reason: rmrSend error, rmr not initialized yet etc.
	// --
	if err != nil {
		if httperr := r.sendAlarmUpdateReqWithHttp(payload); httperr != nil {
			err = fmt.Errorf("%s and  %s", err.Error(), httperr.Error())
		} else {
			err = nil
		}
	}
	return err
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

func InitRMR(r *RICAlarm) error {
	// Setup static RT for alarm system
	alarmRT := fmt.Sprintf("newrt|start\nrte|13111|%s\nnewrt|end\n", r.rmrEndpoint)
	alarmRTFile := "/tmp/alarm.rt"

	if err := ioutil.WriteFile(alarmRTFile, []byte(alarmRT), 0644); err != nil {
		log.Println("ioutil.WriteFile failed with error: ", err)
		return err
	}

	os.Setenv("RMR_SEED_RT", alarmRTFile)
	os.Setenv("RMR_RTG_SVC", "-1")

	if ctx := C.rmrInit(); ctx != nil {
		r.rmrCtx = ctx
		r.rmrReady = true
		return nil
	}

	return errors.New("rmrInit failed!")
}

func (r *RICAlarm) IsRMRReady() bool {
	return r.rmrReady
}
