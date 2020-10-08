..
..  Copyright (c) 2019 AT&T Intellectual Property.
..  Copyright (c) 2019 Nokia.
..
..  Licensed under the Creative Commons Attribution 4.0 International
..  Public License (the "License"); you may not use this file except
..  in compliance with the License. You may obtain a copy of the License at
..
..    https://creativecommons.org/licenses/by/4.0/
..
..  Unless required by applicable law or agreed to in writing, documentation
..  distributed under the License is distributed on an "AS IS" BASIS,
..  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
..
..  See the License for the specific language governing permissions and
..  limitations under the License.
..

User-Guide
==========

.. contents::
   :depth: 3
   :local:

RIC Alarm System
----------------

Overview
--------
RIC alarm system consists of three components: Alarm Manager, Application Library and Command Line Interface

The Alarm Manager is responsible for managing alarm situations in RIC cluster and interfacing with Northbound applications
such as Prometheus Alert Manager to post the alarms as alerts. Alert Manager takes care of de-duplicating, silencing and
inhibition (suppressing) of alerts, and routing them to the VES-Agent, which, in turn, takes care of converting alerts to
faults and sending them to ONAP as VES events.

The Alarm Library provides a simple interface for RIC applications (both platform application and xApps) to raise and clear
alarms. The Alarm Library interacts with the Alarm Manager via RMR interface.

    .. image:: images/RIC_Alarm_System.png
      :width: 600
      :alt: Place in RIC's software architecture picture


Alarm Manager
-------------
The Alarm Manager listens alarms coming via RMR and REST interfaces. An application can raise or clear alarms via either
of interfaces. Alarm Manager listens also commands coming from CLI (Command Line Interface). In addition Alarm Manager supports few
other commands that can be given through the interfaces. Such as list active alarms, list alarm history, add new alarms
definition, delete existing alarm definition, re-raise alarms and clear all alarms. Those are not typically used by applications while
running. Alarm Manager itself re-raises alarms periodically to keep alarms in active state. The other commands are can be used through
CLI interface by operator or are used when applications is starting up or restarting.

Maximum amount of active alarms and size of alarm history are configurable. By default, the values are Maximum number of active
alarms = 5000, Maximum number of alarm history = 20,000.

Alarm definitions can be updated dynamically via REST interface. Default definitions are read from JSON configuration file when FM
service is deployed.


Alarm Library
-------------
The Alarm Library provides simple interface for RIC applications (both platform application and xApps) to raise and clear
alarms. A new alarm instance is created with InitAlarm()-function. ManagedObject (mo) and Application (ap) identities are
given as parameters for Alarm Context/Object

The Alarm object contains following parameters:

    \* SpecificProblem: problem that is the cause of the alarm

    PerceivedSeverity: The severity of the alarm, see below for possible values

    \* ManagedObjectId: The name of the managed object that is the cause of the fault

    \* ApplicationId: The name of the process raised the alarm

    AdditionalInfo: Additional information given by the application

    \* IdentifyingInfo: Identifying additional information, which is part of alarm identity

Items marked with \*, i.e., ManagedObjectId (mo), SpecificProblem (sp), ApplicationId (ap) and IdentifyingInfo (IdentifyingInfo) make
up the identity of the alarm. All parameters must be according to the alarm definition, i.e. all mandatory parameters should be present,
and parameters should have correct value type or be from some predefined range. Addressing the same alarm instance in a clear() or reraise()
call is done by making sure that all four values are the same is in the original raise() / reraise() call.

Alarm Manager does not allow raising "same alarm" more than once without that the alarm is cleared first. Alarm Manager compares
ManagedObjectId (mo), SpecificProblem (sp), ApplicationId (ap) and IdentifyingInfo (IdentifyingInfo) parameters to check possible 
duplicate. If the values are the same then alarm is suppressed. If application raises the "same alarm" but PerceivedSeverity of the alarm
is changed then Alarm Manager deletes the old alarm and makes new alarm according to new information.


Alarm APIs

    Raise: Raises the alarm instance given as a parameter

    Clear: Clears the alarm instance given as a parameter, if it the alarm active

    Reraise: Attempts to re-raise the alarm instance given as a parameter

    ClearAll: Clears all alarms matching moId and appId given as parameters (not supported yet)


Command line interface
----------------------

Through CLI operator can do the following operations:

 - Check active alarms
 - Check alarm history
 - Raise an alarm
 - Clear an alarm
 - Configure maximum active alarms and maximum alarms in alarm history
 - Add new alarm definitions that can be raised
 - Delete existing alarm definition that can be raised

CLI commands need to be given inside Alarm Manger pod. To get there first print name of the Alarm Manger pod.

 kubectl get pods -A | grep alarmmanager

Output should be look someting like this: 

 ricplt  deployment-ricplt-alarmmanager-6cc8764749-gnwjh 1/1 running 0  15d

Then give this command to enter inside the pod. Replace the pod name with the actual name from the printout.

 kubectl exec -it deployment-ricplt-alarmmanager-6cc8764749-gnwjh bash

CLI commands can have some of the following parameters

 - \--moid        ManagedObjectId, example string: RIC 
 - \--apid        ApplicationId string, example string: UEEC  
 - \--sp          SpecificProblem, example value: 8007
 - \--severity    Severity of the alarm, possible values: UNSPECIFIED, CRITICAL, MAJOR, MINOR, WARNING, CLEARED or DEFAULT
 - \--iinfo       Identifying info, a user specified string, example string: INFO-1
 - \--mal         Maximum number of active alarms, example value 1000
 - \--mah         Maximum number of alarms in alarm history, example value: 2000
 - \--aid         Alarm id, example value: 8007
 - \--atx         Alarm text string, example string: E2 CONNECTIVITY LOST TO E-NODEB
 - \--ety         Event type string, example string: Communication error
 - \--oin         Operation instructions string, example string: Not defined
 - \--prf         Performance profile id, possible values: 1 = peak performance test or 2 = endurance test
 - \--nal         Number of alarms, example value: 50
 - \--aps         Alarms per second, example value: 1
 - \--tim         Total time of test in minutes, example value: 1 
 - \--host        Alarm Manager REST address: default value = localhost
 - \--port        Alarm Manager REST port: default value = 8080
 - \--if          Used Alarm Manager command interface, http or rmr: default value = http


 ``Note that there are two minus signs before parameter name! Copy-paste a command from this web page may not work correctly!``
 
 If parameter contains any white spaces then it must be enclosed in quotation marks like: "INFO 1"

CLI command examples:

 Following command are given at top level directory!

 Check active alarms:

   Syntax: cli/alarm-cli active [--host] [--port]
   
   Example: cli/alarm-cli active

   Example: cli/alarm-cli active --host localhost --port 8080

 Check alarm history:

   Syntax: cli/alarm-cli active  [--host] [--port]

   Example: cli/alarm-cli history

   Example: cli/alarm-cli history --host localhost --port 8080

 Raise alarm:

   Syntax: cli/alarm-cli raise --moid --apid --sp --severity --iinfo [--host] [--port] [--if]

   Example: cli/alarm-cli raise --moid RIC --apid UEEC --sp 8007 --severity CRITICAL --iinfo INFO-1

   Following is meant only for testing and verification purpose!

   Example: cli/alarm-cli raise --moid RIC --apid UEEC --sp 8007 --severity CRITICAL --iinfo INFO-1 --host localhost --port 8080 --if rmr

 Clear alarm:

   Syntax: cli/alarm-cli clear --moid --apid --sp --severity --iinfo [--host] [--port] [--if]

   Example: cli/alarm-cli clear --moid RIC --apid UEEC --sp 8007 --iinfo INFO-1

   Example: cli/alarm-cli clear --moid RIC --apid UEEC --sp 8007 --iinfo INFO-1 --host localhost --port 8080 --if rmr

 Configure maximum active alarms and maximum alarms in alarm history:

   Syntax: cli/alarm-cli configure --mal --mah [--host] [--port]

   Example: cli/alarm-cli configure --mal 1000 --mah 5000

   Example: cli/alarm-cli configure --mal 1000 --mah 5000 --host localhost --port 8080

 Add new alarm definition:

   Syntax: cli/alarm-cli define --aid 8007 --atx "E2 CONNECTIVITY LOST TO E-NODEB" --ety "Communication error" --oin "Not defined" [--host] [--port]

   Example: cli/alarm-cli define --aid 8007 --atx "E2 CONNECTIVITY LOST TO E-NODEB" --ety "Communication error" --oin "Not defined"

   Example: cli/alarm-cli define --aid 8007 --atx "E2 CONNECTIVITY LOST TO E-NODEB" --ety "Communication error" --oin "Not defined" --host localhost --port 8080

 Delete existing alarm definition:

   Syntax: cli/alarm-cli undefine --aid [--host] [--port]

   Example: cli/alarm-cli undefine --aid 8007

   Example: cli/alarm-cli undefine --aid 8007 --host localhost --port 8080

 Conduct performance test:

   Note that this is meant only for testing and verification purpose!

   Before any performance test command can be issued, an environment variable needs to be set. The variable holds information where
   test alarm object file is stored.

   PERF_OBJ_FILE=cli/perf-alarm-object.json

   Syntax: cli/alarm-cli perf --prf --nal --aps --tim [--host] [--port] [--if]

   Peak performance test example: cli/alarm-cli perf --prf 1 --nal 50 --aps 1 --tim 1 --if rmr

   Peak performance test example: cli/alarm-cli perf --prf 1 --nal 50 --aps 1 --tim 1 --if http

   Peak performance test example: cli/alarm-cli perf --prf 1 --nal 50 --aps 1 --tim 1 --host localhost --port 8080 --if rmr

   Endurance test example: cli/alarm-cli perf --prf 2 --nal 50 --aps 1 --tim 1 --if rmr

   Endurance test example: cli/alarm-cli perf --prf 2 --nal 50 --aps 1 --tim 1 --if http

   Endurance test example: cli/alarm-cli perf --prf 2 --nal 50 --aps 1 --tim 1 --host localhost --port 8080 --if rmr


REST interface usage guide
--------------------------

REST interface offers all the same services plus some more that are available via CLI. The CLI also uses the REST interface to implement the services it offers.

Below are examples for REST interface. Curl tool is used to send REST commands.

 Check active alarms:

   Example: curl -X GET "http://localhost:8080/ric/v1/alarms/active" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

 Check alarm history:

   Example: curl -X GET "http://localhost:8080/ric/v1/alarms/history" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

 Raise alarm:

   Example: curl -X POST "http://localhost:8080/ric/v1/alarms" -H "accept: application/json" -H "Content-Type: application/json" -d "{\"managedObjectId\": \"RIC\", \"applicationId\": \"UEEC\", \"specificProblem\": 8007, \"perceivedSeverity\": \"CRITICAL\", \"additionalInfo\": \"-\", \"identifyingInfo\": \"INFO-1\", \"AlarmAction\": \"RAISE\", \"AlarmTime\": 0}"

 Clear alarm:

   Example: curl -X DELETE "http://localhost:8080/ric/v1/alarms" -H "accept: application/json" -H "Content-Type: application/json" -d "{\"managedObjectId\": \"RIC\", \"applicationId\": \"UEEC\", \"specificProblem\": 8007, \"perceivedSeverity\": \"\", \"additionalInfo\": \"-\", \"identifyingInfo\": \"INFO-1\", \"AlarmAction\": \"CLEAR\", \"AlarmTime\": 0}"

 Get configuration of maximum active alarms and maximum alarms in alarm history:

   Example: curl -X GET "http://localhost:8080/ric/v1/alarms/config" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

 Configure maximum active alarms and maximum alarms in alarm history:

   Example: curl -X POST "http://localhost:8080/ric/v1/alarms/config" -H "accept: application/json" -H "Content-Type: application/json" -d "{\"maxactivealarms\": 1000, \"maxalarmhistory\": 5000}"

 Get all alarm definitions:

   Example: curl -X GET "http://localhost:8080/ric/v1/alarms/define" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

 Get an alarm definition:

   Syntax: curl -X GET "http://localhost:8080/ric/v1/alarms/define/{alarmId}" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

   Example: curl -X GET "http://localhost:8080/ric/v1/alarms/define/8007" -H  "accept: application/json" -H "Content-Type: application/json" -d "{}"

 Add one new alarm definition:

   Example: curl -X POST "http://localhost:8080/ric/v1/alarms/define" -H "accept: application/json" -H "Content-Type: application/json" -d "{\"alarmdefinitions\": [{\"alarmId\": 8007, \"alarmText\": \"E2 CONNECTIVITY LOST TO E-NODEB\", \"eventtype\": \"Communication error\", \"operationinstructions\": \"Not defined\"}]}"

 Add two new alarm definitions:

   Example: curl -X POST "http://localhost:8080/ric/v1/alarms/define" -H "accept: application/json" -H "Content-Type: application/json" -d "{\"alarmdefinitions\": [{\"alarmId\": 8007, \"alarmText\": \"E2 CONNECTIVITY LOST TO E-NODEB\", \"eventtype\": \"Communication error\", \"operationinstructions\": \"Not defined\"},{\"alarmId\": 8008, \"alarmText\": \"ACTIVE ALARM EXCEED MAX THRESHOLD\", \"eventtype\": \"storage warning\", \"operationinstructions\": \"Clear alarms or raise threshold\"}]}"

 Delete one existing alarm definition:

   Syntax: curl -X DELETE "http://localhost:8080/ric/v1/alarms/define/{alarmId}" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"

   Example: curl -X DELETE "http://localhost:8080/ric/v1/alarms/define/8007" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"


RMR interface usage guide
-------------------------
Through RMR interface application can only raise and clear alarms. RMR message payload is similar JSON message as in above REST interface use cases.

 Supported events via RMR interface
  
  - Raise alarm
  - Clear alarm
  - Reraise alarm
  - ClearAll alarms (not supported yet)


Example on how to use the API from Golang code
----------------------------------------------
Alarm library functions can be used directly from Golang code. Rising and clearing alarms goes via RMR interface from alarm library to Alarm Manager.

 ```go

 package main

 import (
    alarm "gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
 )

 func main() {
    // Initialize the alarm component
    alarmer, err := alarm.InitAlarm("my-pod", "my-app")

    // Create a new Alarm object (SP=8004, etc)
    alarm := alarmer.NewAlarm(8004, alarm.SeverityMajor, "NetworkDown", "eth0")

    // Raise an alarm (SP=8004, etc)
    err := alarmer.Raise(alarm)

    // Clear an alarm (SP=8004)
    err := alarmer.Clear(alarm)

    // Re-raise an alarm (SP=8004)
    err := alarmer.Reraise(alarm)

    // Clear all alarms raised by the application - (not supported yet)
    err := alarmer.ClearAll()
 }
 
 ```

Example VES event
-----------------

INFO[2020-06-08T07:50:10Z]
 {
   "event": {
     "commonEventHeader": {
       "domain": "fault",
       "eventId": "fault0000000001",
       "eventName": "Fault_ricp_E2 CONNECTIVITY LOST TO G-NODEB",
       "lastEpochMicrosec": 1591602610944553,
       "nfNamingCode": "ricp",
       "priority": "Medium",
       "reportingEntityId": "035EEB88-7BA2-4C23-A349-3B6696F0E2C4",
       "reportingEntityName": "Vespa",
       "sequence": 1,
       "sourceName": "RIC",
       "startEpochMicrosec": 1591602610944553,
       "version": 3
     },

     "faultFields": {
       "alarmCondition": "E2 CONNECTIVITY LOST TO G-NODEB",
       "eventSeverity": "MAJOR",
       "eventSourceType": "virtualMachine",
       "faultFieldsVersion": 2,
       "specificProblem": "eth12",
       "vfStatus": "Active"
     }
   }
 }
INFO[2020-06-08T07:50:10Z] Schema validation succeeded
