package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"gerrit.o-ran-sc.org/r/ric-plt/alarm-go/alarm"
	clientruntime "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/jedib0t/go-pretty/table"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/thatisuday/commando"
	"github.com/spf13/viper"
)

type CliAlarmDefinitions struct {
	AlarmDefinitions []*alarm.AlarmDefinition `json:"alarmdefinitions"`
}

type AlarmClient struct {
	alarmer *alarm.RICAlarm
}

type RicPerfAlarmObjects struct {
	AlarmObjects []*alarm.Alarm `json:"alarmobjects"`
}

var CLIPerfAlarmObjects map[int]*alarm.Alarm

var wg sync.WaitGroup

var CliPerfAlarmDefinitions CliAlarmDefinitions

const (
	Raise             string = "RAISE"
	Clear             string = "CLEAR"
	End               string = "END"
	PeakTestDuration  int    = 60
	OneSecondDuration int    = 1
)

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
			postAlarm(flags, readAlarmParams(flags, false), alarm.AlarmActionRaise, nil)
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
			postAlarm(flags, readAlarmParams(flags, true), alarm.AlarmActionClear, nil)
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
		// Conduct performance test for alarm-go
	commando.
		Register("perf").
		SetShortDescription("Conduct performance test with given parameters").
		AddFlag("prf", "performance profile id", commando.Int, nil).
		AddFlag("nal", "number of alarms", commando.Int, nil).
		AddFlag("aps", "alarms per sec", commando.Int, nil).
		AddFlag("tim", "total time of test", commando.Int, nil).
		AddFlag("host", "Alarm manager host address", commando.String, "localhost").
		AddFlag("port", "Alarm manager host address", commando.String, "8080").
		AddFlag("if", "http or rmr used as interface", commando.String, "http").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			conductperformancetest(flags)
		})

	// Get alerts from Prometheus Alert Manager
	commando.
		Register("gapam").
		SetShortDescription("Get alerts from Prometheus Alert Manager").
		AddFlag("active", "Active alerts in Prometheus Alert Manager", commando.Bool, true).
		AddFlag("inhibited", "Inhibited alerts in Prometheus Alert Manager", commando.Bool, true).
		AddFlag("silenced", "Silenced alerts in Prometheus Alert Manager", commando.Bool, true).
		AddFlag("unprocessed", "Unprocessed alerts in Prometheus Alert Manager", commando.Bool, true).
		AddFlag("host", "Prometheus Alert Manager host address", commando.String, nil).
		AddFlag("port", "Prometheus Alert Manager port", commando.String, "9093").
		SetAction(func(args map[string]commando.ArgValue, flags map[string]commando.FlagValue) {
			displayAlerts(flags)
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
		fmt.Println("Couldn't fetch active alarm list due to error: ", err)
		return alarms
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll failed: ", err)
		return alarms
	}

	json.Unmarshal([]byte(body), &alarms)
	return alarms
}

func postAlarmWithRmrIf(a alarm.Alarm, action alarm.AlarmAction, alarmClient *AlarmClient) {
	if alarmClient == nil {
		alarmClient = NewAlarmClient("my-pod", "my-app")
	}
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

func postAlarmWithHttpIf(targetUrl string, a alarm.Alarm, action alarm.AlarmAction) {
	m := alarm.AlarmMessage{Alarm: a, AlarmAction: action}
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("json.Marshal failed: ", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't fetch active alarm list due to error: ", err)
		return
	}
}

func postAlarm(flags map[string]commando.FlagValue, a alarm.Alarm, action alarm.AlarmAction, alarmClient *AlarmClient) {
	// Check the interface to be used for raise or clear the alarm
	rmr_or_http, _ := flags["if"].GetString()
	if rmr_or_http == "rmr" {
		postAlarmWithRmrIf(a, action, alarmClient)
	} else {

		host, _ := flags["host"].GetString()
		port, _ := flags["port"].GetString()
		targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms", host, port)
		postAlarmWithHttpIf(targetUrl, a, action)
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
		fmt.Println("json.Marshal failed: ", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't fetch post alarm configuration due to error: ", err)
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
		fmt.Println("json.Marshal failed: ", err)
		return
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("Couldn't post alarm definition due to error: ", err)
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
		fmt.Println("Couldn't make delete request due to error: ", err)
		return
	}
	resp, errr := client.Do(req)
	if errr != nil || resp == nil {
		fmt.Println("Couldn't send delete request due to error: ", err)
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

// Conduct performance testing
func conductperformancetest(flags map[string]commando.FlagValue) {
	var readerror error
	var senderror error
	var readobjerror error
	host, _ := flags["host"].GetString()
	port, _ := flags["port"].GetString()
	targetUrl := fmt.Sprintf("http://%s:%s/ric/v1/alarms/define", host, port)
	readerror = readPerfAlarmDefinitionFromJson()
	if readerror == nil {
		senderror = sendPerfAlarmDefinitionToAlarmManager(targetUrl)
		if senderror == nil {
			fmt.Println("sent performance alarm definitions to alarm manager")
			CLIPerfAlarmObjects = make(map[int]*alarm.Alarm)
			readobjerror = readPerfAlarmObjectFromJson()
			if readobjerror == nil {
				profile, _ := flags["prf"].GetInt()
				if profile == 1 {
					fmt.Println("starting peak performance test")
					peakPerformanceTest(flags)
				} else if profile == 2 {
					fmt.Println("starting endurance test")
					enduranceTest(flags)
				} else {
					fmt.Println("Unknown profile, received profile = ", profile)
				}
			} else {
				fmt.Println("reading performance alarm objects from json file failed ")
			}
		} else {
			fmt.Println("sending performance alarm definitions to alarm manager failed ")
		}

	} else {
		fmt.Println("reading performance alarm definitions from json file failed ")
	}

}

func peakPerformanceTest(flags map[string]commando.FlagValue) {
	nalarms, _ := flags["nal"].GetInt()
	var count int = 0
	for aid, obj := range CLIPerfAlarmObjects {
		count = count + 1
		if count <= nalarms {
			fmt.Println("peakPerformanceTest: invoking worker routine ", count, aid, *obj)
			wg.Add(1)
			go raiseClearAlarmOnce(obj, flags)
		} else {
			break
		}
	}
	fmt.Println("peakPerformanceTest: Waiting for workers to finish")
	wg.Wait()
	fmt.Println("peakPerformanceTest: Wait completed")
}

func enduranceTest(flags map[string]commando.FlagValue) {
	alarmspersec, _ := flags["aps"].GetInt()
	var count int = 0
	for aid, obj := range CLIPerfAlarmObjects {
		count = count + 1
		if count <= alarmspersec {
			fmt.Println("enduranceTest: invoking worker routine ", count, aid, *obj)
			wg.Add(1)
			go raiseClearAlarmOverPeriod(obj, flags)
		} else {
			break
		}
	}
	fmt.Println("enduranceTest: Waiting for workers to finish")
	wg.Wait()
	fmt.Println("enduranceTest: Wait completed")
}

func readPerfAlarmObjectFromJson() error {
	filename := os.Getenv("PERF_OBJ_FILE")
	file, err := ioutil.ReadFile(filename)
	if err == nil {
		data := RicPerfAlarmObjects{}
		err = json.Unmarshal([]byte(file), &data)
		if err == nil {
			for _, alarmObject := range data.AlarmObjects {
				ricAlarmObject := new(alarm.Alarm)
				ricAlarmObject.ManagedObjectId = alarmObject.ManagedObjectId
				ricAlarmObject.ApplicationId = alarmObject.ApplicationId
				ricAlarmObject.SpecificProblem = alarmObject.SpecificProblem
				ricAlarmObject.PerceivedSeverity = alarmObject.PerceivedSeverity
				ricAlarmObject.AdditionalInfo = alarmObject.AdditionalInfo
				ricAlarmObject.IdentifyingInfo = alarmObject.IdentifyingInfo
				CLIPerfAlarmObjects[alarmObject.SpecificProblem] = ricAlarmObject
			}
		} else {
			fmt.Println("readPerfAlarmObjectFromJson: json.Unmarshal failed with error ", err)
			return err
		}
	} else {
		fmt.Println("readPerfAlarmObjectFromJson: ioutil.ReadFile failed with error ", err)
		return err
	}
	return nil
}

func readPerfAlarmDefinitionFromJson() error {
	filename := os.Getenv("PERF_DEF_FILE")
	file, err := ioutil.ReadFile(filename)
	if err == nil {
		data := CliAlarmDefinitions{}
		err = json.Unmarshal([]byte(file), &data)
		if err == nil {
			for _, alarmDefinition := range data.AlarmDefinitions {
				_, exists := alarm.RICAlarmDefinitions[alarmDefinition.AlarmId]
				if exists {
					fmt.Println("ReadPerfAlarmDefinitionFromJson: alarm definition already exists for ", alarmDefinition.AlarmId)
				} else {
					fmt.Println("ReadPerfAlarmDefinitionFromJson: alarm ", alarmDefinition.AlarmId)
					ricAlarmDefintion := new(alarm.AlarmDefinition)
					ricAlarmDefintion.AlarmId = alarmDefinition.AlarmId
					ricAlarmDefintion.AlarmText = alarmDefinition.AlarmText
					ricAlarmDefintion.EventType = alarmDefinition.EventType
					ricAlarmDefintion.OperationInstructions = alarmDefinition.OperationInstructions
					CliPerfAlarmDefinitions.AlarmDefinitions = append(CliPerfAlarmDefinitions.AlarmDefinitions, ricAlarmDefintion)
				}
			}
		} else {
			fmt.Println("ReadPerfAlarmDefinitionFromJson: json.Unmarshal failed with error: ", err)
			return err
		}
	} else {
		fmt.Println("ReadPerfAlarmDefinitionFromJson: ioutil.ReadFile failed with error: ", err)
		return err
	}
	return nil
}

func sendPerfAlarmDefinitionToAlarmManager(targetUrl string) error {

	jsonData, err := json.Marshal(CliPerfAlarmDefinitions)
	if err != nil {
		fmt.Println("sendPerfAlarmDefinitionToAlarmManager: json.Marshal failed: ", err)
		return err
	}

	resp, err := http.Post(targetUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp == nil {
		fmt.Println("sendPerfAlarmDefinitionToAlarmManager: Couldn't post alarm definition to targeturl due to error: ", targetUrl, err)
		return err
	}
	return nil
}

func wakeUpAfterTime(timeinseconds int, chn chan string, action string) {
	time.Sleep(time.Second * time.Duration(timeinseconds))
	chn <- action
}

func raiseClearAlarmOnce(alarmobject *alarm.Alarm, flags map[string]commando.FlagValue) {
	var alarmClient *AlarmClient = nil
	defer wg.Done()
	chn := make(chan string, 1)
	rmr_or_http, _ := flags["if"].GetString()
	if rmr_or_http == "rmr" {
		alarmClient = NewAlarmClient("my-pod", "my-app")
	}
	postAlarm(flags, *alarmobject, alarm.AlarmActionRaise, alarmClient)
	go wakeUpAfterTime(PeakTestDuration, chn, Clear)
	select {
	case res := <-chn:
		if res == Clear {
			postAlarm(flags, *alarmobject, alarm.AlarmActionClear, alarmClient)
			go wakeUpAfterTime(PeakTestDuration, chn, End)
		} else if res == End {
			return
		}
	}
}

func raiseClearAlarmOverPeriod(alarmobject *alarm.Alarm, flags map[string]commando.FlagValue) {
	var alarmClient *AlarmClient = nil
	defer wg.Done()
	timeinminutes, _ := flags["tim"].GetInt()
	timeinseconds := timeinminutes * 60
	chn := make(chan string, 1)
	rmr_or_http, _ := flags["if"].GetString()
	if rmr_or_http == "rmr" {
		alarmClient = NewAlarmClient("my-pod", "my-app")
	}
	postAlarm(flags, *alarmobject, alarm.AlarmActionRaise, alarmClient)
	go wakeUpAfterTime(OneSecondDuration, chn, Clear)
	go wakeUpAfterTime(timeinseconds, chn, End)
	for {
		select {
		case res := <-chn:
			if res == Raise {
				postAlarm(flags, *alarmobject, alarm.AlarmActionRaise, alarmClient)
				go wakeUpAfterTime(OneSecondDuration, chn, Clear)
			} else if res == Clear {
				postAlarm(flags, *alarmobject, alarm.AlarmActionClear, alarmClient)
				go wakeUpAfterTime(OneSecondDuration, chn, Raise)
			} else if res == End {
				return
			}
		}
	}
}

func displayAlerts(flags map[string]commando.FlagValue) {
	resp, err := getAlerts(flags)
	if err != nil {
		fmt.Println(err)
		return
	}

	if resp == nil {
		fmt.Println("resp= nil")
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Alerts from Prometheus Alert Manager"})
	for _, gettableAlert := range resp.Payload{
		t.AppendRow([]interface{}{"------------------------------------"})
		if gettableAlert != nil {
			for key, item := range gettableAlert.Annotations {
				t.AppendRow([]interface{}{key, item})	
			}
			if gettableAlert.EndsAt != nil {
				t.AppendRow([]interface{}{"EndsAt", *gettableAlert.EndsAt})
			}
			if gettableAlert.Fingerprint != nil {
				t.AppendRow([]interface{}{"Fingerprint", *gettableAlert.Fingerprint})
			}
			for key, item := range gettableAlert.Receivers {
				if gettableAlert.Receivers != nil {
					t.AppendRow([]interface{}{key, *item.Name})	
				}
			}
			if gettableAlert.StartsAt != nil {
				t.AppendRow([]interface{}{"StartsAt", *gettableAlert.StartsAt})
			}
			if gettableAlert.Status != nil {
				t.AppendRow([]interface{}{"InhibitedBy", gettableAlert.Status.InhibitedBy})
				t.AppendRow([]interface{}{"SilencedBy", gettableAlert.Status.SilencedBy})
				t.AppendRow([]interface{}{"State", *gettableAlert.Status.State})
			}
			if gettableAlert.UpdatedAt != nil {
				t.AppendRow([]interface{}{"UpdatedAt", *gettableAlert.UpdatedAt})
			}
			t.AppendRow([]interface{}{"GeneratorURL", gettableAlert.Alert.GeneratorURL})
			for key, item := range gettableAlert.Alert.Labels {
				t.AppendRow([]interface{}{key, item})	
			}
		}
	}
	t.SetStyle(table.StyleColoredBright)
	t.Render()
}
	
func getAlerts(flags map[string]commando.FlagValue) (*alert.GetAlertsOK, error) {
	active, _ := flags["active"].GetBool()
	inhibited, _ := flags["inhibited"].GetBool()
	silenced, _ := flags["silenced"].GetBool()
	unprocessed, _ := flags["unprocessed"].GetBool()
	amHost, _ := flags["host"].GetString()
	amPort, _ := flags["port"].GetString()
	var amAddress string
	if amHost == "" {
		amAddress = viper.GetString("controls.promAlertManager.address")
	} else {
		amAddress = amHost + ":" + amPort
	}

	alertParams := alert.NewGetAlertsParams()
	alertParams.Active = &active
	alertParams.Inhibited = &inhibited
	alertParams.Silenced = &silenced
	alertParams.Unprocessed = &unprocessed
	amBaseUrl := viper.GetString("controls.promAlertManager.baseUrl")
	amSchemes := []string{viper.GetString("controls.promAlertManager.schemes")}
	resp, err := newAlertManagerClient(amAddress, amBaseUrl, amSchemes).Alert.GetAlerts(alertParams)
	if err != nil {
		err = fmt.Errorf("GetAlerts from '%s%s' failed with error: %v", amAddress, amBaseUrl, err)
	}
	return resp, err
}

func newAlertManagerClient(amAddress string, amBaseUrl string, amSchemes []string) *client.Alertmanager {
	cr := clientruntime.New(amAddress, amBaseUrl, amSchemes)
	return client.New(cr, strfmt.Default)
}

