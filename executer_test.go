package bpnet_test

import (
	"testing"
	"github.com/veith/bpnet"

	"fmt"
	"github.com/oklog/ulid"
	"time"
)

var process bpnet.Process
var FlowCollection map[ulid.ULID]*bpnet.Flow

func init() {
	FlowCollection = map[ulid.ULID]*bpnet.Flow{}
}

func TestProcess_System(t *testing.T) {
	process := freshProcess()

	process.InitialState = []int{1, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 6, 1, 1, 1, 1, 1}

	process.Transitions = make([]bpnet.Transition, 7)
	process.Transitions[1].Details = map[string]interface{}{"delay": 2}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)
	if len(f.TransitionsInProgress) < 1 {
		t.Error("Sollte eine erlaubte Systemtransition haben")
	}
	// tokenId
	f.FireSystemTask(sysTaskToken) // sysTaskToken is set from SystemTaskHandler

	if len(f.TransitionsInProgress) > 0 {
		t.Error("Sollte keine erlaubte Systemtransition haben")
	}

	if f.Net.State[len(f.Net.State)-1] != 1 {
		t.Error("Should have 1 transition in last place, is %s", f.Net.State[len(f.Net.State)-1])
	}
}


func TestMessage(t *testing.T) {
	process := freshProcess()
	process.InputMatrix = [][]int{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}
	process.OutputMatrix = [][]int{
		{0, 1, 0, 0,},
		{0, 0, 1, 0,},
		{0, 0, 0, 1,},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 3, 1}
	process.OnSendMessage = sendMessage
	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"broker": "sms"}
	broker = "oh"
	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)
	if broker != "sms" {
		t.Error("broker should be sms, is", broker)
	}
	if len(f.Net.EnabledTransitions) != 0{
		t.Error("message should fire after sending")
	}
}

func TestProcess_Auto(t *testing.T) {
	process := freshProcess()
	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 1, 1, 1, 1, 1, 1}
	f := process.CreateFlow("veith")

	FlowCollection[f.ID] = f
	var data map[string]interface{}
	f.Start(data)

	// last place should have n tokens
	if f.Net.State[len(f.Net.State)-1] != 10 {
		t.Error("Should have 10 transition in last place, is %s", f.Net.State[len(f.Net.State)-1])
	}

}

func TestProcess_TimedParallel(t *testing.T) {
	process := freshProcess()
	process.InputMatrix = [][]int{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}
	process.OutputMatrix = [][]int{
		{0, 1, 0, 0,},
		{0, 0, 1, 0,},
		{0, 0, 0, 1,},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1}

	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"delay": 1}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(2 * time.Second)
	if len(f.Net.TokenIds[3]) != 2 {
		t.Error("Should fired both timers", f.Net.TokenIds)
	}
	if f.Net.State[len(f.Net.State)-1] != 2 {
		t.Error("Should have 2 transition in last place, is %s", f.Net.State[len(f.Net.State)-1])
	}
}

func TestProcess_Timed(t *testing.T) {
	process := freshProcess()

	process.InitialState = []int{1, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1, 1, 1, 1, 1}

	process.Transitions = make([]bpnet.Transition, 5)
	process.Transitions[1].Details = map[string]interface{}{"delay": 1}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(2 * time.Second)
	if f.Net.State[len(f.Net.State)-1] != 1 {
		t.Error("Should have 10 transition in last place, is %s", f.Net.State[len(f.Net.State)-1])
	}
}

func TestFlow_Fire2(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = f
	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
	var data map[string]interface{}

	f.Start(data)

	if len(f.AvailableUserTransitions) == 0 {
		t.Error("Sollte eine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
	f.Fire(0)
	err := f.Fire(0)

	if err == nil {
		t.Error("Sollte einen Fehler ausgeben, weil Transition bereits gezündet wurde")
	}
	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition mehr haben")
		fmt.Println(f.AvailableUserTransitions)
	}
}

func TestFlow_Fire(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = f
	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
	var data map[string]interface{}

	f.Start(data)

	if len(f.AvailableUserTransitions) == 0 {
		t.Error("Sollte eine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
	f.Fire(0)

	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition mehr haben")
		fmt.Println(f.AvailableUserTransitions)
	}
}

// erstellen eines flows
func TestFlow_Start(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = f
	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
	var data map[string]interface{}

	f.Start(data)

	if len(f.AvailableUserTransitions) == 0 {
		t.Error("Sollte eine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}
}

// erstellen eines flows
func TestProcess_CreateFlow(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = f
	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Sollte keine erlaubte Systemtransition haben")
		fmt.Println(f.AvailableUserTransitions)
	}

	process.InitialState = []int{100, 0, 0, 0, 0, 0, 0, 0, 0}

	if f.Net.State[0] != 1 {
		t.Error("objekte sollten entkoppelt sein")
		fmt.Println(f.Net.State)
	}

}

func freshProcess() bpnet.Process {
	process = bpnet.Process{
		Name: "name",
		InputMatrix: [][]int{
			{1, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 1, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 1, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 1, 1, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 1, 0},
		},
		OutputMatrix: [][]int{
			{0, 1, 1, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 1, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 1, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 1, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 1, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		InitialState:    []int{1, 0, 0, 0, 0, 0, 0, 0, 0},
		TransitionTypes: []int{2, 1, 1, 1, 1, 1, 2},
	}

	process.OnSystemTask = SystemTaskHandler
	process.OnFireCompleted = fireCompleted
	process.OnTimerStarted = triggerhandle
	process.OnTimerCompleted = OnTimerCompleted
	process.OnSubprocessStarted = subflowtriggerhandle
	process.ProcessDefinitionLoader = loadSubProcess
	process.FlowInstanceLoader = loadFlowInstance
	return process
}
func freshSubProcess() bpnet.Process {
	process = bpnet.Process{
		Name: "name",
		InputMatrix: [][]int{
			{1, 0, 0},
			{0, 1, 0},
		},
		OutputMatrix: [][]int{
			{0, 1, 0},
			{0, 0, 1},
		},
		InitialState:    []int{1, 0, 0},
		TransitionTypes: []int{1, 1, 1},
	}

	process.OnSystemTask = SystemTaskHandler
	process.OnFireCompleted = fireCompleted
	process.OnTimerStarted = triggerhandle
	process.OnTimerCompleted = OnTimerCompleted
	process.OnSubprocessStarted = subflowtriggerhandle
	process.OnSubprocessCompleted = subflowtriggerhandle
	process.ProcessDefinitionLoader = loadSubProcess
	process.FlowInstanceLoader = loadFlowInstance
	return process
}

func loadSubProcess(processID string) *bpnet.Process {
	p := freshSubProcess()
	return &p
}

func loadFlowInstance(flowID ulid.ULID) *bpnet.Flow {
	f := FlowCollection[flowID]

	return f
}

func fireCompleted(flow *bpnet.Flow, transitionIndex int) bool {
	//fmt.Println(transitionIndex)
	return true
}
func OnTimerCompleted(flow *bpnet.Flow, transitionIndex int) bool {
	//fmt.Println("Timer", flow.Net.TokenIds)
	return true
}
func triggerhandle(flow *bpnet.Flow, transitionIndex int) bool {
	fmt.Println("T")
	return true
}
var sysTaskToken int // wird vom Test verwendet
func SystemTaskHandler(flow *bpnet.Flow, tokenID int) bool {
	sysTaskToken = tokenID
	return true
}

func subflowtriggerhandle(flow *bpnet.Flow, transitionIndex int) bool {
	// subflow starten
	fmt.Println(flow.AvailableUserTransitions)
	return true
}

var broker string

func sendMessage(flow *bpnet.Flow, transitionIndex int) bool {
	// subflow starten

	broker = flow.Process.Transitions[transitionIndex].Details["broker"].(string)
	fmt.Println(broker)
	return true
}
