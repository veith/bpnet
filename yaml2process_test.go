package bpnet_test

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/oklog/ulid"
	"github.com/veith/bpnet"
	"io/ioutil"
	"testing"
	"time"
)

var parentflow *bpnet.Flow

func TestFireWrongConditions(t *testing.T) {
	process := readfile("test/sample1.yaml")

	flow := process.CreateFlow("veith")
	parentflow = &flow
	d := map[string]interface{}{"counts": 12}
	flow.Start(d)
	err := flow.Fire(0, d)
	if err.(bpnet.RequiredError).Fields[0] != "message" {
		t.Error("missing fields should be message , is", err.(bpnet.RequiredError).Fields[0])
	}
}
func TestStartWithMissingFields(t *testing.T) {
	process := readfile("test/looper.yaml")

	flow := process.CreateFlow("veith")
	d := map[string]interface{}{}
	err := flow.Start(d)

	flow.Fire(0, d)
	time.Sleep(200 * time.Millisecond)

	if err.(bpnet.RequiredError).Fields[0] != "counts" {
		t.Error("missing fields should be counts , is", err.(bpnet.RequiredError).Fields[0])
	}
}

func TestSystem(t *testing.T) {
	process := readfile("test/looper.yaml")

	flow := process.CreateFlow("veith")
	d := map[string]interface{}{"counts": 1}
	flow.Start(d)
	flow.Fire(0, d)
	time.Sleep(200 * time.Millisecond)
	if flow.ReadData()["counts"] != 11 {
		t.Error("daten sollten aktualisiert sein =>11, is", flow.ReadData()["counts"])
	}
}

func TestConditions(t *testing.T) {
	process := readfile("test/sample1.yaml")

	flow := process.CreateFlow("veith")
	parentflow = &flow
	d := map[string]interface{}{"counts": 1}
	flow.Start(d)
	flow.Fire(0, d)

	time.Sleep(120 * time.Millisecond)

	if flow.Net.State[0] != 1 {
		t.Error("process muss aufgrund bedingungen hier aufhören")
	}
}

func TestMakeProcessFromYaml(t *testing.T) {
	handler.FlowInstanceLoader = flowloader
	handler.ProcessDefinitionLoader = loadProcDef
	handler.OnSystemTask = OnSystemTask

	process := readfile("test/sample1.yaml")

	flow := process.CreateFlow("veith")
	parentflow = &flow
	d := map[string]interface{}{"counts": 9, "message": "messagemessage"}
	flow.Start(d)

	if flow.Net.State[0] != 1 {
		t.Error("process muss komplett durchlaufen", flow.Net.State)
	}
}

func loadProcDef(processName string) (*bpnet.Process, error) {
	subprocess := readfile("test/subprocess.yaml")
	return &subprocess, nil
}
func flowloader(flowID ulid.ULID) (*bpnet.Flow, error) {
	return parentflow, nil
}

func OnSystemTask(flow *bpnet.Flow, tokenID int) bool {
	time.AfterFunc(100, func() {
		d := map[string]interface{}{"counts": 11}
		flow.FireSystemTask(tokenID, d)
	})

	return true
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

	return process

}
