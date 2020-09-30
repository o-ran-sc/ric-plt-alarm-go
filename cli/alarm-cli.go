package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	"github.com/jedib0t/go-pretty/table"
	"github.com/thatisuday/commando"
)

type CliAlarmDefinitions struct {
	AlarmDefinitions []*alarm.AlarmDefinition `json:"alarmdefinitions"`
}

type AlarmClient struct {
	alarmer *alarm.RICAlarm
}

func main() {

	// configure commando
	commando.
		SetExecutableName("alarm-cli").
		SetVersion("1.0.0").
		SetDescription("This CLI tool provides management interface to SEP alarm system")

	// Get active alarms
	commando.
		Register("active").
		SetShortDescription("Displays the SEP active alarms").
		SetDescription("This command displays more information about the SEP active alarms").
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			displayAlarms(getAlarms(flags, "active"), false)
		})

	// Get alarm history
	commando.
		Register("history").
		SetShortDescription("Displays the SEP alarm history").
		SetDescription("This command displays more information about the SEP alarm history").
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			displayAlarms(getAlarms(flags, "history"), true)
		})

	// Raise an alarm
	commando.
		Register("raise").
		SetShortDescription("Raises alarm with given parameters").
		AddFlag("moid", "Managed object Id", commando.String, nil).
		AddFlag("apid", "Application Id", commando.String, nil).
		AddFlag("sp", "Specific problem Id", commando.Int, nil).
		AddFlag("severity", "Perceived severity", commando.String, nil).
		AddFlag("iinfo", "Application identifying info", commando.String, nil).
		AddFlag("aai", "Application additional info", commando.String, "-").
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		AddFlag("if", "http or rmr used as interface", commando.String, "http").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			postAlarm(flags, readAlarmParams(flags, false), alarm.AlarmActionRaise)
		})

	// Clear an alarm
	commando.
		Register("clear").
		SetShortDescription("Raises alarm with given parameters").
		AddFlag("moid", "Managed object Id", commando.String, nil).
		AddFlag("apid", "Application Id", commando.String, nil).
		AddFlag("sp", "Specific problem Id", commando.Int, nil).
		AddFlag("iinfo", "Application identifying info", commando.String, nil).
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		AddFlag("if", "http or rmr used as interface", commando.String, "http").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			postAlarm(flags, readAlarmParams(flags, true), alarm.AlarmActionClear)
		})

	// Configure an alarm manager
	commando.
		Register("configure").
		SetShortDescription("Configure alarm manager with given parameters").
		AddFlag("mal", "max active alarms", commando.Int, nil).
		AddFlag("mah", "max alarm history", commando.Int, nil).
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			postAlarmConfig(flags)
		})
	// Create alarm definition
	commando.
		Register("define").
		SetShortDescription("Define alarm with given parameters").
		AddFlag("aid", "alarm identifier", commando.Int, nil).
		AddFlag("atx", "alarm text", commando.String, nil).
		AddFlag("ety", "event type", commando.String, nil).
		AddFlag("oin", "operation instructions", commando.String, nil).
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			postAlarmDefinition(flags)
		})
		// Delete alarm definition
	commando.
		Register("undefine").
		SetShortDescription("Define alarm with given parameters").
		AddFlag("aid", "alarm identifier", commando.Int, nil).
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			deleteAlarmDefinition(flags)
		})

	// parse command-line arguments
	commando.Parse(nil)
}

func readAlarmParams(flags map[string]commando.FlagValue, clear bool) (a alarm.Alarm) {
	a.ManagedObjectId, _ = flags["moid"].GetString()
	a.ApplicationId, _ = flags["apid"].GetString()
	a.SpecificProblem, _ = flags["sp"].GetInt()
	a.IdentifyingInfo, _ = flags["iinfo"].GetString()

	if !clear {
		s, _ := flags["severity"].GetString()
		a.PerceivedSeverity = alarm.Severity(s)
	}

	if !clear {
		a.AdditionalInfo, _ = flags["aai"].GetString()
	}
	return
}

func getAlarms(flags map[string]commando.FlagValue, action alarm.AlarmAction) (alarms []alarm.AlarmMessage) {
	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms/%s", host, port, action)
	resp, err := http.Get(targetUrl)
	if err != nil || resp == nil || resp.Body == nil {
		fmt.Println("Couldn't fetch active alarm list due to error: %v", err)
		return alarms
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll failed: %v", err)
		return alarms
	}

	json.Unmarshal([]byte(body), &alarms)
	return alarms
}

func postAlarm(flags map[string]commando.FlagValue, a alarm.Alarm, action alarm.AlarmAction) {

	// Check the interface to be used for raise or clear the alarm
	rmr_or_http, _ := flags["if"].GetString()
	if rmr_or_http == "rmr" {
		alarmClient := NewAlarmClient("my-pod", "my-app")
		if alarmClient == nil {
			return
		}

		// Wait until RMR is up-and-running
		for !alarmClient.alarmer.IsRMRReady() {
			time.Sleep(100 * time.Millisecond)
		}

		if action == alarm.AlarmActionRaise {
			alarmClient.alarmer.Raise(a)
		}

		if action == alarm.AlarmActionClear {
			alarmClient.alarmer.Clear(a)
		}
		return
	}

	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms", host, port)

	m := alarm.AlarmMessage{Alarm: a, AlarmAction: action}
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("json.Marshal failed: %v", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't fetch active alarm list due to error: %v", err)
		return
	}
}

func displayAlarms(alarms []alarm.AlarmMessage, isHistory bool) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if isHistory {
		t.AppendHeader(table.Row{"SP", "MOID", "APPID", "IINFO", "SEVERITY", "AAI", "ACTION", "TIME"})
	} else {
		t.AppendHeader(table.Row{"SP", "MOID", "APPID", "IINFO", "SEVERITY", "AAI", "TIME"})
	}

	for _, a := range alarms {
		alarmTime := time.Unix(0, a.AlarmTime).Format("02/01/2006, 15:04:05")
		if isHistory {
			t.AppendRows([]table.Row{
				{a.SpecificProblem, a.ManagedObjectId, a.ApplicationId, a.IdentifyingInfo, a.PerceivedSeverity, a.AdditionalInfo, a.AlarmAction, alarmTime},
			})
		} else {
			t.AppendRows([]table.Row{
				{a.SpecificProblem, a.ManagedObjectId, a.ApplicationId, a.IdentifyingInfo, a.PerceivedSeverity, a.AdditionalInfo, alarmTime},
			})
		}
	}

	t.SetStyle(table.StyleColoredBright)
	t.Render()
}

func postAlarmConfig(flags map[string]commando.FlagValue) {
	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	maxactivealarms, _ := flags["mal"].GetInt()
	maxalarmhistory, _ := flags["mah"].GetInt()
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms/config", host, port)

	m := alarm.AlarmConfigParams{MaxActiveAlarms: maxactivealarms, MaxAlarmHistory: maxalarmhistory}
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("json.Marshal failed: %v", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't fetch post alarm configuration due to error: %v", err)
		return
	}
}

func postAlarmDefinition(flags map[string]commando.FlagValue) {
	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	alarmid, _ := flags["aid"].GetInt()
	alarmtxt, _ := flags["atx"].GetString()
	etype, _ := flags["ety"].GetString()
	operation, _ := flags["oin"].GetString()
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms/define", host, port)

	var alarmdefinition alarm.AlarmDefinition
	alarmdefinition.AlarmId = alarmid
	alarmdefinition.AlarmText = alarmtxt
	alarmdefinition.EventType = etype
	alarmdefinition.OperationInstructions = operation

	m := CliAlarmDefinitions{AlarmDefinitions: []*alarm.AlarmDefinition{&alarmdefinition}}
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("json.Marshal failed: %v", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't post alarm definition due to error: %v", err)
		return
	}
}

func deleteAlarmDefinition(flags map[string]commando.FlagValue) {
	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	alarmid, _ := flags["aid"].GetInt()
	salarmid := strconv.FormatUint(uint64(alarmid), 10)
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms/define/%s", host, port, salarmid)

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", targetUrl, nil)
	if err != nil || req == nil {
		fmt.Println("Couldn't make delete request due to error: %v", err)
		return
	}
	resp, errr := client.Do(req)
	if errr != nil || resp == nil {
		fmt.Println("Couldn't send delete request due to error: %v", err)
		return
	}
}

// NewAlarmClient returns a new AlarmClient.
func NewAlarmClient(moId, appId string) *AlarmClient {
	alarmInstance, err := alarm.InitAlarm(moId, appId)
	if err == nil {
		return &AlarmClient{
			alarmer: alarmInstance,
		}
	}
	fmt.Println("Failed to create alarmInstance", err)
	return nil
}
