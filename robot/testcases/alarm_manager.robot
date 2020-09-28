*** Settings ***
Library           OperatingSystem
Library           String
Library           DateTime

*** Variables ***
${Alarm_go_dir}         	/home/work/RIC/src/alarmgorobottesting/alarm-go/
${Host}                 	--host localhost
${Port}                 	--port 8080
${Get_active_alarm}     	cli/alarm-cli active
${Get_alarm_history}    	cli/alarm-cli history
${Raise_alarm}          	cli/alarm-cli raise
${Clear_alarm}          	cli/alarm-cli clear
${Define_alarm}         	cli/alarm-cli define
${Undefine_alarm}       	cli/alarm-cli undefine
${Configure_alarm}      	cli/alarm-cli configure
${Configure_alarm_manager}      cli/alarm-cli configure
${Defne_alarm}          	cli/alarm-cli define
${Undefne_alarm}        	cli/alarm-cli undefine
${Get_defined_alarms}   	cli/alarm-cli    defined_alarms
${Moid}    			--moid "SEP"
${Apid}    			--apid "MYAPP"
${Sp}    			--sp 8004
${Severity}     		--severity "CRITICAL"
${Iinfo}        		--iinfo "INFO-1"
${Aai}          		--aai "AAI"
${Aid}          		--aid 8004
${Atx}          		--atx "RIC ROUTING TABLE DISTRIBUTION FAILED"
${Ety}          		--ety "Processing error"
${Oin}          		--oin "Not defined"
${Mal}	        		--mal 0
${Mah}          		--mah 11

*** Test Cases ***
Raise Alarm
    [Documentation]    FCA_MEC-3477 LN0739_FM_FR1: As an operator, I want SEP FM framework to provide client utility to raise and clear alarm

    ###raise alarm and check for active alarms; alarm shall be present
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

Clear Alarm
    [Documentation]    FCA_MEC-3477 LN0739_FM_FR1: As an operator, I want SEP FM framework to provide client utility to raise and clear alarm

    ###clear alarm and check for active alarms; alarm shall not be present
    ${resp}=	Clear Alarm
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

Deploy SEP FM
    [Documentation]    FCA_MEC-3479 LN0739_FM_FR3: As an operator, I shall be able to deploy SEP FM service in the SEP cluster using SEP platform deployment framework and procedure
    Skiptest

CLI Utility checks
    [Documentation]    FCA_MEC-3483 LN0739_FM_FR7: As an operator, I shall be able to monitor SEP alarm situation using local user interface (CLI)

    ###undefine alarm and try to raise alarm; attempt fail
    ${resp}=	Undefine Alarm
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

    ###define alarm and try to raise alarm; attempt success
    ${resp}=	Define Alarm
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###check alarm history; alarm shall be present
    ${resp}=    Get Alarm History
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###clear alarm and check the active alarms; alarm shall not be present
    ${resp}=	Clear Alarm
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

    ###check the alarm history; alarm shall be present
    ${resp}=    Get Alarm History
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

Suppress Clear Alarm Request
    [Documentation]    FCA_MEC-3487 LN0739_FM_FR11: As an operator, I want SEP alarm framework to suppress the clear alarm request if there is no active alarm found corresponding to clear alarm received

    ###clear alarm 8005 which is not present; shall be suppressed
    ${moid}=	set variable	"SEP"
    ${apid}=	set variable	"MYAPP"
    ${sp}=	set variable	8005
    ${iinfo}=	set variable	"INFO-2"
    ${resp}=    Get Alarm History
    Should Not Contain    ${resp}    8005    MYAPP    CRITICAL    AAI
    ${resp}=    Run    ${Alarm_go_dir}${Clear_alarm} ${Host} ${Port} --moid ${moid} --apid ${apid} --sp ${sp} --iinfo ${iinfo}
    ${resp}=    Get Alarm History
    Should Not Contain    ${resp}    8005    MYAPP    CRITICAL    AAI

FM Performance checks
    [Documentation]    FCA_MEC-3490 LN0713_FM_NFR1: SEP alarm framework performance requirements
    Skiptest

Change Alarm Severity
    [Documentation]    FCA_MEC-3481 LN0739_FM_FR5: As an operator, I want SEP alarm framework to support functionality to escalate alarm by changing severity of active alarm which is already raised
    
    ###raise alarm with severity major
    ${severity}=     set variable    "MAJOR"
    ${resp}=    Run    ${Alarm_go_dir}${Raise_alarm} ${Host} ${Port} ${Moid} ${Apid} ${sp} --severity ${severity} ${Iinfo} ${Aai}
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004    MYAPP    MAJOR    AAI

    ###raise alarm with severity critical
    ${resp}=    Run    ${Alarm_go_dir}${Raise_alarm} ${Host} ${Port} ${Moid} ${Apid} ${Sp} ${Severity} ${Iinfo} ${Aai}
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###clear alarm and check the active alarms; alarm shall not be present
    ${resp}=    Run    ${Alarm_go_dir}${Clear_alarm} ${Host} ${Port} ${Moid} ${Apid} ${Sp} ${Iinfo}
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

Alarm UTC checks
    [Documentation]    FCA_MEC-3482 LN0739_FM_FR6: As an operator, I want SEP FM service shall use local UTC time to update alarm time.

    ###get current time
    ${year}=    Get Time    year
    ${month}=    Get Time    month
    ${day}=    Get Time    day
    ${hour}=    Get Time    hour 
    ${min}=    Get Time    min
    ${second}=    Get Time    second

    ###raise alarm; verify reported time with current time
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004    ${day}/${month}/${year}, ${hour}:${min}:${second}

    ###cleanup: clear alarm
    ${resp}=	Clear Alarm
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

Alarm Max Active History Thresholds Checks
    [Documentation]    FCA_MEC-3484 LN0739_FM_FR8: As an operator, I shall be able to configure "Maximum number of active alarms" and "Maximum number of alarm history" for the FM service during deployment

    ###set the max active alarm and max alarm history to 0 and 6 respectively; confirm alarm 8008 toe be absent in active alarm list
    ${resp}=    Configure Alarm Threshold
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8008   ALARMMANAGER    WARNING

    ###raise alarm 8004; alarm 8008 shall be raised.
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8008   ALARMMANAGER    WARNING

    ###total entries in alarm hostory is 1 less than max alarm history; so 8009 shall be absent
    ${resp}=    Get Alarm History
    Should Not Contain    ${resp}    8009   ALARMMANAGER    WARNING

    ###clear alarm 8004; it shall trigger raising 8009 
    ${resp}=	Clear Alarm
    ${resp}=    Get Alarm History
    Should Contain     ${resp}    8009   ALARMMANAGER    WARNING


Duplicate Alarm
    [Documentation]    FCA_MEC-3486 LN0739_FM_FR10: As an operator, I want SEP alarm framework to suppress the alarm which is already reported (duplicate)
    ${resp}=	Raise Alarm
    ${resp}=	Raise Alarm
    ${resp}=    Get Active Alarm
    Should Contain     ${resp}    8004
    ${resp}=	Clear Alarm
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    8004

Unknown Alarm
    [Documentation]    FCA_MEC-3488 LN0739_FM_FR12: As an operator, I want SEP alarm framework to suppress the raise alarm, if alarm is raised with unknow Specific problem

    ###raise alarm 1000(unknown); alarm shall be suppressed
    ${sp}=	set variable	1000
    ${resp}=    Run    ${Alarm_go_dir}${Raise_alarm} ${Host} ${Port} ${Moid} ${Apid} --sp ${sp} ${Severity} ${Iinfo} ${Aai}
    ${resp}=    Get Active Alarm
    Should Not Contain    ${resp}    1000   MYAPP    CRITICAL

Fault Alert
    [Documentation]    FCA_MEC-3489 LN0739_FM_FR13: As an operator, I want SEP alarm framework shall support interface to external Prometheus alert manager to notify SEP alarm situation
    Skiptest


*** Keywords ***

Skiptest
    Set Tags          disabled
    Pass Execution    This test is disabled

Raise Alarm
     ${resp}=    Run    ${Alarm_go_dir}${Raise_alarm} ${Host} ${Port} ${Moid} ${Apid} ${Sp} ${Severity} ${Iinfo} ${Aai}

Get Active Alarm
     ${resp}=    Run    ${Alarm_go_dir}${Get_active_alarm} ${Host} ${Port}
     [return]	${resp}

Clear Alarm
     ${resp}=    Run    ${Alarm_go_dir}${Clear_alarm} ${Host} ${Port} ${Moid} ${Apid} ${Sp} ${Iinfo}

Undefine Alarm
     ${resp}=    Run    ${Alarm_go_dir}${Undefine_alarm} ${Host} ${Port} ${Aid}

Define Alarm
     ${resp}=    Run    ${Alarm_go_dir}${Define_alarm} ${Host} ${Port} ${Aid} ${Atx} ${Ety} ${Oin}

Get Alarm History
     ${resp}=    Run    ${Alarm_go_dir}${Get_alarm_history} ${Host} ${Port}
     [return]	${resp}

Configure Alarm Threshold
     ${resp}=    Run    ${Alarm_go_dir}${Configure_alarm} ${Host} ${Port} ${Mal} ${Mah}
     
