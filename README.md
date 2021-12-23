RIC Alarm Manager and Library Interface
=======================================


This repository containts Golang implementation of Alarm Manager and related application library interface.

Architecture
------------

![Architecture](assets/alarm-adapter.png)

The **Alarm Library** provides a simple interface for RIC applications (both platform application and xApps) to raise, clear and re-raise. The **Alarm Library** interacts with the **Alarm Manager** via RMR interface.

The **Alarm Manager** is responsible for managing alarm situations in RIC cluster and interfacing with **Northboubd** applications such as **Prometheus AlertManager** to post the alarms as alerts. AlertManager takes care of deduplicating, silencing and inhibition (suppressing) of alerts, and routing them to the VESAgent, which, in turn, takes care of converting alerts to fault and send to ONAP as VES events.

Overview for Alarm Manager
--------------------------

### TBD

Overview for Alarm Library
--------------------------

## Initialization

A new alarm instance is created with InitAlarm function. MO and application identities are given as a parameter.

## Alarm Context and Format

The Alarm object contains following parameters:
 * *SpecificProblem*: problem that is the cause of the alarm
 * *PerceivedSeverity*: The severity of the alarm, see above for possible values
 * *ManagedObjectId*: The name of the managed object that is the cause of the fault
 * *ApplicationId*: The name of the process raised the alarm
 * *AdditionalInfo*: Additional information given by the application
 * *IdentifyingInfo*: Identifying additional information, which is part of alarm identity

 *ManagedObjectId* (mo), *SpecificProblem* (sp), *ApplicationId* (ap) and *IdentifyingInfo* (IdentifyingInfo) make up the identity of the alarm. All parameters must be according to the alarm definition, i.e. all mandatory parameters should be present, and parameters should have correct value type or be from some predefined range. Addressing the same alarm instance in a clear() or reraise() call is done by making sure that all four values are the same is in the original raise / reraise call. 

## Alarm APIs
* *Raise*: Raises the alarm instance given as a parameter
* *Clear*: Clears the alarm instance given as a parameter, if it the alarm active
* *Reraise*: Attempts to re-raise the alarm instance given as a parameter
* *ClearAll*: Clears all alarms matching moId and appId given as parameters

## Aux. Alarm APIs
* *SetManagedObjectId*: Sets the default MOId
* *SetApplicationId*: Sets the default AppId


## Example
-------

```go
package main

import (
	alarm "gerrit.o-ran-sc.org/r/ric-plt/alarm-go.git/alarm"
)

func main() {
	// Initialize the alarm component
	alarmer, err := alarm.InitAlarm("my-pod", "my-app")

	// Create a new Alarm object
	alarm := alarmer.NewAlarm(1234, alarm.SeverityMajor, "Some App data", "eth 0 1")

	// Raise an alarm (SP=1234, etc)
	err := alarmer.Raise(alarm)

	// Clear an alarm (SP=1234)
	err := alarmer.Clear(alarm)

	// Re-raise an alarm (SP=1234)
	err := alarmer.Reraise(alarm)

	// Clear all alarms raised by the application
	err := alarmer.ClearAll()
}
```

CI
--

The Dockerfile in the `ci` directory _only_ runs, when build, the library unit tests for the repository.

License
-------
 Copyright (c) 2020 AT&T Intellectual Property.
 Copyright (c) 2020 Nokia.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.

 This source code is part of the near-RT RIC (RAN Intelligent Controller)
 platform project (RICP).



