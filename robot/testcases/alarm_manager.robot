*** Settings ***
Library           OperatingSystem
Library           String
Library           DateTime

*** Variables ***
${alarm_go_dir}   /home/work/RIC/src/alarmgorobottesting/alarm-go/
${host}           --host localhost
${port}           --port 8080
${get_active_alarm}    cli/alarm-cli active
${get_alarm_history}    cli/alarm-cli history
${raise_alarm}    cli/alarm-cli raise
${clear_alarm}    cli/alarm-cli clear
${define_alarm}    cli/alarm-cli define
${undefine_alarm}    cli/alarm-cli undefine
${configure_alarm}    cli/alarm-cli configure
${configure_alarm_manager}    cli/alarm-cli configure
${defne_alarm}    cli/alarm-cli define
${undefne_alarm}    cli/alarm-cli undefine
${get_defined_alarms}    cli/alarm-cli    defined_alarms
${moid8004}    --moid "SEP"
${apid8004}    --apid "MYAPP"
${sp8004}    --sp 8004
${severity8004}    --severity "CRITICAL"
${iinfo8004}    --iinfo "INFO-1"
${aai8004}    --aai "AAI"
${aid8004}    --aid 8004
${atx8004}    --atx "RIC ROUTING TABLE DISTRIBUTION FAILED"
${ety8004}    --ety "Processing error"
${oin8004}    --oin "Not defined"
${moid8005}    --moid "SEP"
${apid8005}    --apid "MYAPP"
${sp8005}    --sp 8005
${severity8005}    --severity "CRITICAL"
${iinfo8005}    --iinfo "INFO-2"
${aai8005}    --aai "AAI"

*** Test Cases ***
LN0739_FM_FR1
    [Documentation]    FCA_MEC-3477 LN0739_FM_FR1: As an operator, I want SEP FM framework to provide client utility to raise and clear alarm

    ###raise alarm and check for active alarms; alarm shall be present
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###clear alarm and check for active alarms; alarm shall not be present
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${iinfo8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

LN0739_FM_FR3
    [Documentation]    FCA_MEC-3479 LN0739_FM_FR3: As an operator, I shall be able to deploy SEP FM service in the SEP cluster using SEP platform deployment framework and procedure
    Skiptest

LN0739_FM_FR7
    [Documentation]    FCA_MEC-3483 LN0739_FM_FR7: As an operator, I shall be able to monitor SEP alarm situation using local user interface (CLI)

    ###undefine alarm and try to raise alarm; attempt fail
    ${resp}=    Run    ${alarm_go_dir}${undefine_alarm} ${host} ${port} ${aid8004}
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

    ###define alarm and try to raise alarm; attempt success
    ${resp}=    Run    ${alarm_go_dir}${define_alarm} ${host} ${port} ${aid8004} ${atx8004} ${ety8004} ${oin8004}
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###check alarm history; alarm shall be present
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###clear alarm and check the active alarms; alarm shall not be present
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${iinfo8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

    ###check the alarm history; alarm shall be present
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

LN0739_FM_FR11
    [Documentation]    FCA_MEC-3487 LN0739_FM_FR11: As an operator, I want SEP alarm framework to suppress the clear alarm request if there is no active alarm found corresponding to clear alarm received

    ###clear alarm 8005 which is not present; shall be suppressed
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Not Contain    ${resp}    8005    MYAPP    CRITICAL    AAI
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8005} ${apid8005} ${sp8005} ${iinfo8005}
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Not Contain    ${resp}    8005    MYAPP    CRITICAL    AAI

LN0713_FM_NFR1
    [Documentation]    FCA_MEC-3490 LN0713_FM_NFR1: SEP alarm framework performance requirements
    Skiptest

LN0739_FM_FR5
    [Documentation]    FCA_MEC-3481 LN0739_FM_FR5: As an operator, I want SEP alarm framework to support functionality to escalate alarm by changing severity of active alarm which is already raised
    
    ###raise alarm with severity major
    ${severity}=     set variable    "MAJOR"
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} --severity ${severity} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    MAJOR    AAI

    ###raise alarm with severity critical
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004    MYAPP    CRITICAL    AAI

    ###clear alarm and check the active alarms; alarm shall not be present
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${iinfo8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

LN0739_FM_FR6
    [Documentation]    FCA_MEC-3482 LN0739_FM_FR6: As an operator, I want SEP FM service shall use local UTC time to update alarm time.

    ###get current time
    ${year}=    Get Time    year
    ${month}=    Get Time    month
    ${day}=    Get Time    day
    ${hour}=    Get Time    hour 
    ${min}=    Get Time    min
    ${second}=    Get Time    second

    ###raise alarm; verify reported time with current time
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004    ${day}/${month}/${year}, ${hour}:${min}:${second}

    ###cleanup: clear alarm
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${iinfo8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8004    MYAPP    CRITICAL    AAI

LN0739_FM_FR8
    [Documentation]    FCA_MEC-3484 LN0739_FM_FR8: As an operator, I shall be able to configure "Maximum number of active alarms" and "Maximum number of alarm history" for the FM service during deployment

    #better to keep the max thresholds as it is dependendent on the above cases as well.
    ${mal}=	set variable	0
    ${mah}=	set variable	11
    
    ###set the max active alarm and max alarm history to 0 and 6 respectively; confirm alarm 8008 toe be absent in active alarm list
    ${resp}=    Run    ${alarm_go_dir}${configure_alarm} ${host} ${port} --mal ${mal} --mah ${mah}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    8008   ALARMMANAGER    WARNING

    ###raise alarm 8004; alarm 8008 shall be raised.
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8008   ALARMMANAGER    WARNING

    ###total entries in alarm hostory is 1 less than max alarm history; so 8009 shall be absent
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Not Contain    ${resp}    8009   ALARMMANAGER    WARNING

    ###clear alarm 8004; it shall trigger raising 8009 
    ${resp}=    Run    ${alarm_go_dir}${clear_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${iinfo8004}
    ${resp}=    Run    ${alarm_go_dir}${get_alarm_history} ${host} ${port}
    Should Contain     ${resp}    8009   ALARMMANAGER    WARNING

    #reset the max thresholds
    ${mal}=	set variable	1000
    ${mah}=	set variable	2000
    ${resp}=    Run    ${alarm_go_dir}${configure_alarm} ${host} ${port} ${mal} ${mah}


LN0739_FM_FR10
    [Documentation]    FCA_MEC-3486 LN0739_FM_FR10: As an operator, I want SEP alarm framework to suppress the alarm which is already reported (duplicate)
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp8004} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Contain     ${resp}    8004

LN0739_FM_FR12
    [Documentation]    FCA_MEC-3488 LN0739_FM_FR12: As an operator, I want SEP alarm framework to suppress the raise alarm, if alarm is raised with unknow Specific problem

    ###raise alarm 1000(unknown); alarm shall be suppressed
    ${sp1000}=	set variable	1000
    ${resp}=    Run    ${alarm_go_dir}${raise_alarm} ${host} ${port} ${moid8004} ${apid8004} ${sp1000} ${severity8004} ${iinfo8004} ${aai8004}
    ${resp}=    Run    ${alarm_go_dir}${get_active_alarm} ${host} ${port}
    Should Not Contain    ${resp}    1000   MYAPP    CRITICAL

LN0739_FM_FR13
    [Documentation]    FCA_MEC-3489 LN0739_FM_FR13: As an operator, I want SEP alarm framework shall support interface to external Prometheus alert manager to notify SEP alarm situation
    Skiptest


*** Keywords ***

Skiptest
    Set Tags          disabled
    Pass Execution    This test is disabled

