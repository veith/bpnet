package bpnet

import (
	"strings"
)

/**
 * Erstellt aus einem ImportNet ein Process Struct
 */
func MakeProcessFromYaml(yamlstruct ImportNet) Process {


	var targetNetwork Process
	targetNetwork.Name = yamlstruct.Title

	// places
	places := make(map[string]int)
	for index, place := range yamlstruct.Place {
		// inital state
		targetNetwork.InitialState = append(targetNetwork.InitialState, place.Tokens)
		// build map
		places[place.ID] = index
	}

	// transitions
	transitions := make(map[string]int)
	for index, transition := range yamlstruct.Transition {
		// build map
		transitions[transition.ID] = index
		targetNetwork.Transitions = append(targetNetwork.Transitions, transition)

		// ttype ist eine bitmatrix
		var ttype TaskType
		ttype = 0
		ttype = ttypeCheck(transition.TransitionType, "auto", ttype, AUTO)
		ttype = ttypeCheck(transition.TransitionType, "form", ttype, USER)
		ttype = ttypeCheck(transition.TransitionType, "user", ttype, USER)
		ttype = ttypeCheck(transition.TransitionType, "message", ttype, MESSAGE)
		ttype = ttypeCheck(transition.TransitionType, "timed", ttype, TIMED)
		ttype = ttypeCheck(transition.TransitionType, "subprocess", ttype, SUBPROCESS)
		ttype = ttypeCheck(transition.TransitionType, "system", ttype, SYSTEM)
		ttype = ttypeCheck(transition.TransitionType, "call", ttype, SYSTEM)
		ttype = ttypeCheck(transition.TransitionType, "ai", ttype, SYSTEM)
		ttype = ttypeCheck(transition.TransitionType, "api", ttype, SYSTEM)
		targetNetwork.TransitionTypes = append(targetNetwork.TransitionTypes, int(ttype))

	}

	// Variables
	targetNetwork.Variables = yamlstruct.Variables
	targetNetwork.StartVariables = yamlstruct.StartVariables

	// create null Matrix
	inputMatrix := make([][]int, len(transitions))
	outputMatrix := make([][]int, len(transitions))
	conditionMatrix := make([][]string, len(transitions))
	for i := 0; i < len(transitions); i++ {
		innerLen := len(places)
		inputMatrix[i] = make([]int, innerLen)
		outputMatrix[i] = make([]int, innerLen)
		//conditionMatrix[i] = make([]string, innerLen)
		for j := 0; j < innerLen; j++ {
			inputMatrix[i][j] = 0
			outputMatrix[i][j] = 0

		}
	}

	// build input and output Matrix
	for _, arc := range yamlstruct.Arc {
		if (arc.Type == "pt") {
			inputMatrix[transitions[arc.Destination]][places[arc.Source]] = max(1, arc.Weight)
			if len(arc.Condition) > 0 {
				conditionMatrix[transitions[arc.Destination]] = append(conditionMatrix[transitions[arc.Destination]], arc.Condition)
			}

		}

		if (arc.Type == "tp") {
			outputMatrix[transitions[arc.Source]][places[arc.Destination]] = max(arc.Weight,1)
		}
	}
	targetNetwork.InputMatrix = inputMatrix
	targetNetwork.OutputMatrix = outputMatrix
	targetNetwork.ConditionMatrix = conditionMatrix

	return targetNetwork

}
func ttypeCheck(transitionType string, substr string, ttype TaskType, assign TaskType) TaskType {
	if strings.Contains(strings.ToLower(transitionType), substr) {
		ttype = ttype  | assign
	}
	return ttype
}


// structs werden gehoistet wie es aussieht
type ImportNet struct {
	Title          string       `json:"title"` // Affects YAML field names too.
	Transition     []Transition `json:"transitions"`
	Variables      []Variable   `json:"variables"`
	StartVariables []string     `json:"startvariables"`
	Place          []Place      `json:"places"`
	Arc            []Arc        `json:"arcs"`
}

type Transition struct {
	ID             string                 `json:"id"`
	TransitionType string                 `json:"type"`
	Details        map[string]interface{} `json:"details"`
	ReqVariables   []string               `json:"variables"`
}

type Variable struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Place struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Tokens int    `json:"tokens"`
}

type Arc struct {
	Source      string `json:"sourceId"`
	Destination string `json:"destinationId"`
	Condition   string `json:"condition"`
	Type        string `json:"type"`
	Weight      int    `json:"weight"`
}
