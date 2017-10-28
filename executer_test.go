package bpnet_test

import (
	"testing"
	"github.com/veith/bpnet"

	"fmt"
)

var process bpnet.Process

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

	process.SystemTrigger = triggerhandle
	process.PostFire = triggerhandle
	return process
}

func triggerhandle (taskType bpnet.TaskType, flow *bpnet.Flow, transitionIndex int) bool{
fmt.Println("T",taskType)
	return true
}


func TestProcess_Message(t *testing.T) {

	//func (t *TriggerHandler) Trigger ()

	process := freshProcess()

	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 3, 1, 1, 1, 1, 1}

	var data map[string]interface{}
	f := process.Start("veith", "xxxxx", data)

	if f.Net.State[len(f.Net.State)-1] != 10 {
		t.Error("Should have 10 transition in last place, is %s",  f.Net.State[len(f.Net.State)-1])
	}
}


func TestProcess_Timed(t *testing.T) {
t.Skip()

}


func TestProcess_Auto(t *testing.T) {
	process := freshProcess()
	process.InitialState = []int{10, 0, 0, 0, 0, 0, 0, 0, 0}
	process.TransitionTypes = []int{1, 1, 1, 1, 1, 1, 1}

	var data map[string]interface{}
	f := process.Start("veith", "xxxxx", data)

	// last place should have n tokens
	if f.Net.State[len(f.Net.State)-1] != 10 {
		t.Error("Should have 10 transition in last place, is %s",  f.Net.State[len(f.Net.State)-1])
	}

}

func TestProcess_Start(t *testing.T) {
	process := freshProcess()
	var data map[string]interface{}
	f := process.Start("veith", "xxxxx", data)

	if len(f.AvailableUserTransitions) != 1 {
		t.Error("Should have 1 transition to fire (0)")
	}
	f.Fire(0)

	if f.AvailableUserTransitions[0] != 6 {
		t.Error("Transition 6 should be a user Task")
	}

	f.Fire(6)

	if len(f.AvailableUserTransitions) != 0 {
		t.Error("Should have no transitions left")
	}
}
