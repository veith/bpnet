package bpnet_test

import (
	"github.com/veith/bpnet"
	"testing"

	"fmt"
	"github.com/oklog/ulid"
	"time"
)

var process bpnet.Process
var FlowCollection map[ulid.ULID]*bpnet.Flow
var handler bpnet.Handler

func init() {
	FlowCollection = map[ulid.ULID]*bpnet.Flow{}
	handler.OnProcessStarted = OnProcessStarted
	handler.OnSystemTask = OnSystemTask
	handler.OnFireCompleted = fireCompleted
	handler.OnTimerStarted = triggerhandle
	handler.OnTimerCompleted = OnTimerCompleted
	handler.OnSubProcessCompleted = OnSubprocessCompleted
	handler.OnProcessCompleted = OnProcessCompleted
	handler.OnSubProcessStarted = OnSubprocessStarted
	handler.ProcessDefinitionLoader = loadSubProcess
	handler.FlowInstanceLoader = loadFlowInstance
	handler.OnSendMessage = sendMessage

	handler.OnStateChanged = func(flow *bpnet.Flow) bool {
		return true
	}

	bpnet.RegisterHandler(&handler)
}

func TestProcess_CompleteOnce(t *testing.T) {
	process := freshProcess()
	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 1, 1, 1, 1, 1, 1}
	f := process.CreateFlow("veith")
	completed = 0
	FlowCollection[f.ID] = &f
	var data map[string]interface{}
	f.Start(data)

	// last place should have n tokens
	if f.Net.State[len(f.Net.State)-1] != 10 {
		t.Error("Should have 10 transition in last place, is", f.Net.State[len(f.Net.State)-1])
	}
	time.Sleep(10 * time.Millisecond)
	// sollte nur ein mal beenden
	if completed != 1 {
		t.Error("Should only complete once, is", completed)
	}
}

func TestDataPointer(t *testing.T) {
	process := freshProcess()
	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 1, 1, 1, 1, 1, 1}
	f := process.CreateFlow("veith")

	data := map[string]interface{}{"a": 12}
	f.Start(data)

	d := f.ReadData()

	if d["a"] != f.Net.Variables["a"] {
		t.Error("Data should point to f.Net.Variables")
	}

}

func TestProcess_Subflow(t *testing.T) {

	//func (t *TriggerHandler) Trigger ()

	process := freshProcess()

	process.InitialState = []int{3, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 5, 1, 1, 5, 1, 1}

	process.Transitions = make([]bpnet.Transition, 7)
	process.Transitions[4].Details = map[string]interface{}{"process": "sub"}
	process.Transitions[1].Details = map[string]interface{}{"process": "sub"}

	var data map[string]interface{}
	f := process.CreateFlow("veith")

	FlowCollection[f.ID] = &f
	f.Start(data)

	if f.Net.State[len(f.Net.State)-1] != 3 {
		t.Error("Should have 10 transition in last place, is", f.Net.State[len(f.Net.State)-1])
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
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 3, 1}

	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"broker": "sms"}
	broker = "oh"
	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)
	if broker != "sms" {
		t.Error("broker should be sms, is", broker)
	}
	if len(f.Net.EnabledTransitions) != 0 {
		t.Error("message should fire after sending")
	}
}

func TestProcess_Auto(t *testing.T) {
	process := freshProcess()
	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 1, 1, 1, 1, 1, 1}
	f := process.CreateFlow("veith")

	FlowCollection[f.ID] = &f
	var data map[string]interface{}
	f.Start(data)

	// last place should have n tokens
	if f.Net.State[len(f.Net.State)-1] != 10 {
		t.Error("Should have 10 transition in last place, is", f.Net.State[len(f.Net.State)-1])
	}

}

func TestProcess_TimedUnset(t *testing.T) {
	process := freshProcess()
	process.InputMatrix = [][]int{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}
	process.OutputMatrix = [][]int{
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1}

	process.Transitions = make([]bpnet.Transition, 3)
	// Typo is part of the test
	process.Transitions[1].Details = map[string]interface{}{"delay": ""}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(110 * time.Millisecond)
	if len(f.Net.TokenIds[3]) != 2 {
		t.Error("Should fired both timers", f.Net.TokenIds)
	}
	if f.Net.State[len(f.Net.State)-1] != 2 {
		t.Error("Should have 2 transition in last place, is", f.Net.State[len(f.Net.State)-1])
	}
}
func TestProcess_TimedFloat(t *testing.T) {
	process := freshProcess()
	process.InputMatrix = [][]int{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}
	process.OutputMatrix = [][]int{
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1}

	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"delay": 0.01}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(20 * time.Millisecond)
	if len(f.Net.TokenIds[3]) != 2 {
		t.Error("Should fired both timers", f.Net.TokenIds)
	}
	if f.Net.State[len(f.Net.State)-1] != 2 {
		t.Error("Should have 2 transition in last place, is", f.Net.State[len(f.Net.State)-1])
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
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	process.InitialState = []int{2, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1}

	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"delay": 0.01}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(200 * time.Millisecond)
	if len(f.Net.TokenIds[3]) != 2 {
		t.Error("Should fired both timers", f.Net.TokenIds)
	}
	if f.Net.State[len(f.Net.State)-1] != 2 {
		t.Error("Should have 2 transition in last place, but have", f.Net.State[len(f.Net.State)-1])
	}
}

func TestProcess_Timed(t *testing.T) {
	process := freshProcess()

	process.InitialState = []int{1, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 4, 1, 1, 1, 1, 1}

	process.Transitions = make([]bpnet.Transition, 5)
	process.Transitions[1].Details = map[string]interface{}{"delay": 0.01}

	var data map[string]interface{}
	f := process.CreateFlow("veith")
	f.Start(data)

	time.Sleep(12 * time.Millisecond)
	if f.Net.State[len(f.Net.State)-1] != 1 {
		t.Error("Should have 10 transition in last place, is", f.Net.State[len(f.Net.State)-1])
	}
}

func TestFlow_Fire2(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = &f
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
	f.Fire(0, data)
	err := f.Fire(0, data)

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
	FlowCollection[f.ID] = &f
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
	f.Fire(0, data)

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
	FlowCollection[f.ID] = &f
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

func BenchmarkProcess_CreateFlow(b *testing.B) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}
	for i := 0; i < b.N; i++ {
		process.CreateFlow("veith")

	}

}

// erstellen eines flows
func TestProcess_CreateFlow(t *testing.T) {
	process := freshProcess()
	process.TransitionTypes = []int{2, 1, 1, 1, 1, 1, 1}

	f := process.CreateFlow("veith")
	FlowCollection[f.ID] = &f
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
		TransitionTypes: []int{1, 3, 1},
	}

	process.Transitions = make([]bpnet.Transition, 3)
	process.Transitions[1].Details = map[string]interface{}{"broker": "sms"}

	return process
}

func loadSubProcess(processID string) (*bpnet.Process, error) {
	p := freshSubProcess()
	return &p, nil
}

func loadFlowInstance(flowID ulid.ULID) (*bpnet.Flow, error) {
	f := FlowCollection[flowID]
	return f, nil
}

func fireCompleted(flow *bpnet.Flow, transitionIndex int) bool {
	//fmt.Println(transitionIndex)
	return true
}
func SUBfireCompleted(flow *bpnet.Flow, transitionIndex int) bool {
	fmt.Println(transitionIndex)
	return true
}
func OnTimerCompleted(flow *bpnet.Flow, transitionIndex int) bool {
	fmt.Println("Timer completed", flow.Net.TokenIds)
	return true
}

func OnTimerStarted(flow *bpnet.Flow, transitionIndex int) bool {
	fmt.Println("Timer Started", flow.Net.TokenIds)
	return true
}

func triggerhandle(flow *bpnet.Flow, transitionIndex int) bool {
	fmt.Println("T")
	return true
}

var sysTaskToken []int // wird vom Test verwendet
func SystemTaskHandler(flow *bpnet.Flow, tokenID int) bool {
	sysTaskToken = append(sysTaskToken, tokenID)
	return true
}

func OnSubprocessStarted(flow *bpnet.Flow, tokenID int) bool {
	// subflow starten
	fmt.Println("start subprocess", tokenID, flow.ProcessName)
	return true
}

func OnSubprocessCompleted(flow *bpnet.Flow, tokenID int) bool {
	// subflow starten
	fmt.Println("completed subprocess", flow.ID, flow.ProcessName)
	return true
}
func OnProcessStarted(flow *bpnet.Flow, tokenID int) bool {
	// subflow starten
	fmt.Println("START process", flow.ID, tokenID)
	return true
}

var completed int

func OnProcessCompleted(flow *bpnet.Flow, tokenID int) bool {
	// subflow starten
	fmt.Println("COMPLETE process", flow.ID, tokenID)
	completed += 1
	return true
}

var broker string

func sendMessage(flow *bpnet.Flow, transitionIndex int) bool {
	// subflow starten

	broker = flow.Process.Transitions[transitionIndex].Details["broker"].(string)
	fmt.Println(broker)
	return true
}
