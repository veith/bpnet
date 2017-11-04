package bpnet_test

import (
	"io/ioutil"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/veith/bpnet"
	"testing"
	"time"
)

func TestMakeProcessFromYaml(t *testing.T) {
	yaml := readfile("test/sample1.yaml")
	process := bpnet.MakeProcessFromYaml(yaml)
	process.OnTimerStarted = OnTimerStarted
	process.OnTimerCompleted = OnTimerCompleted
	process.OnSubProcessStarted = OnSubprocessStarted
	process.OnSendMessage = sendMessage
	process.OnSubProcessCompleted = OnSubprocessCompleted

	flow := process.CreateFlow("veith")
	var d map[string]interface{}
	flow.Start(d)
	fmt.Println(flow.Net.State)
	flow.Fire(0)
	fmt.Println(flow.Net.State)
	time.Sleep(1500 * time.Millisecond)
	fmt.Println(flow.Net.State)
}

func readfile(filename string) bpnet.ImportNet {
	b, _ := ioutil.ReadFile(filename)

	// Unmarshal the YAML
	var yamlstruct bpnet.ImportNet
	err := yaml.Unmarshal([]byte(b), &yamlstruct)
	if err != nil {
		fmt.Printf("err: %v\n", err)

	}
	return yamlstruct

}
