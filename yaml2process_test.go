package bpnet_test

import (
	"io/ioutil"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/veith/bpnet"
	"testing"
	"time"
	"github.com/oklog/ulid"
)

var subprocess bpnet.Process
var parentflow *bpnet.Flow

func TestConditions(t *testing.T) {
	process := readfile("test/sample1.yaml")




	flow := process.CreateFlow("veith")
	parentflow = flow
	d :=  map[string]interface{}{"counts":1}
	flow.Start(d)
	flow.Fire(0)

	time.Sleep(300 * time.Millisecond)

	if flow.Net.State[0] != 1{
		t.Error("process muss aufgrund bedingungen hier aufhören")
	}
}

func TestMakeProcessFromYaml(t *testing.T) {

	process := readfile("test/sample1.yaml")

	flow := process.CreateFlow("veith")
	parentflow = flow
	d :=  map[string]interface{}{"counts":9}
	flow.Start(d)

	flow.Fire(0)
	time.Sleep(300 * time.Millisecond) // inner delay

	if flow.Net.State[3] != 1{
		t.Error("process muss komplett durchlaufen")
	}
}

func loadProcDef(processName string) (*bpnet.Process, error) {
	subprocess := readfile("test/subprocess.yaml")
	return &subprocess, nil
}
func flowloader(flowID ulid.ULID) (*bpnet.Flow, error) {
	return parentflow, nil
}

func readfile(filename string) bpnet.Process {
	b, _ := ioutil.ReadFile(filename)

	// Unmarshal the YAML
	var yamlstruct bpnet.ImportNet
	err := yaml.Unmarshal([]byte(b), &yamlstruct)
	if err != nil {
		fmt.Printf("err: %v\n", err)

	}

	process = bpnet.MakeProcessFromYaml(yamlstruct)
	process.OnTimerStarted = OnTimerStarted
	process.OnTimerCompleted = OnTimerCompleted
	process.OnSubProcessStarted = OnSubprocessStarted
	process.OnSendMessage = sendMessage
	process.OnSubProcessCompleted = OnSubprocessCompleted
	process.OnProcessCompleted = OnProcessCompleted
	process.ProcessDefinitionLoader = loadProcDef
	process.FlowInstanceLoader = flowloader

	return process

}
