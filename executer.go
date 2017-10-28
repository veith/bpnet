package bpnet

import (
	"github.com/veith/petrinet"
)

// starte einen prozess
func (p Process) Start(Owner string, flowID string, data map[string]interface{}) Flow {
	//init
	f := Flow{Owner: Owner, Process: p}
	f.ID = flowID

	f.Net.Variables = data
	f.Net.InputMatrix = p.InputMatrix
	f.Net.OutputMatrix = p.OutputMatrix
	f.Net.State = make([]int, len(p.InitialState))
	copy(f.Net.State, p.InitialState)
	f.Net.Init()

	f.AvailableUserTransitions = f.bpnTransitionsCheck();
	return f
}

// Fire
func (f *Flow) Fire(transitionIndex int) error {
	err := f.fire(transitionIndex)
	if err == nil {

		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		if f.Process.PostFire != nil {
			f.Process.PostFire(AUTO, f, transitionIndex)
		}
		return nil
	}
	return err
}

// fire
func (f *Flow) fire(transitionIndex int) error {
	err := f.Net.Fire(transitionIndex)
	if err == nil {
		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		return nil
	}
	return err
}


func (f *Flow) bpnTransitionsCheck() []int {
	//

	// selbstfeuernde transitionen auslösen
	for f.hasEnabledAutofireing(f.Net.EnabledTransitions) {
		for _, transition := range f.Net.EnabledTransitions {
			// auf alle autofire typen pruefen
			if f.Process.TransitionTypes[transition] == int(AUTO) {
				//autofire
				f.fire(transition)
				break
			}
			if f.Process.TransitionTypes[transition] == int(MESSAGE) {
				// send message via extHandler, continue on true

				if f.Process.SystemTrigger(MESSAGE, f, transition) {
					f.fire(transition)
				}
				break
			}
		}
	}

	for _, transition := range f.Net.EnabledTransitions {

		// alle noch nicht benachrichtigten timer prüfen
		if f.Process.TransitionTypes[transition] == int(TIMED) && !itemAlreadyNotified(transition, f.TransitionsInProgress) {
			//f.Process.Transitions[transition].Delay
			if f.Process.SystemTrigger(TIMED, f, transition) {
				f.TransitionsInProgress = append(f.TransitionsInProgress, transition)
			}

		}
		// alle noch nicht gestarteten subprocesse
		if f.Process.TransitionTypes[transition]  == int(SUBPROCESS) && !itemAlreadyNotified(transition, f.TransitionsInProgress) {
			if f.Process.SystemTrigger(SUBPROCESS, f, transition) {
				f.TransitionsInProgress = append(f.TransitionsInProgress, transition)
			}

		}
	}
	return f.Net.EnabledTransitions
}

// prüfe ob ein Timer bereits angestossen wurde.
func itemAlreadyNotified(t int, list []int) bool {
	for _, v := range list {
		if v == t {
			return true
		}
	}
	return false
}

//prüfe ob enablete Transitionen mit Autofeuer existieren
func (f *Flow) hasEnabledAutofireing(enabledTransitions []int) bool {
	for _, transition := range enabledTransitions {
		// auf alle autofire pruefen
		if f.Process.TransitionTypes[transition]&(int(AUTO)) == int(AUTO) {
			return true
		}
		// auf alle autofire pruefen
		if f.Process.TransitionTypes[transition]&(int(MESSAGE)) == int(MESSAGE) {
			return true
		}
	}
	return false
}

// TaskTypen
// SYSTEM:
// Prozesse, AI können via transition details

const (
	AUTO       TaskType = 1 // feuert selbst ab
	USER       TaskType = 2 // claim in transition details (assign macht system)
	MESSAGE    TaskType = 3 // sendet via Trigger
	TIMED      TaskType = 4 // feuert durch Zeit
	SUBPROCESS TaskType = 5 // startet einen subprozess, TODO: automatisches anlegen eines Auto vor dem subprozess start
	SYSTEM     TaskType = 6 // assign in transition details (claim macht user)

)

type TaskType int

func (t *TaskType) shouldLock() {
	// je nach typ sollte die Transition locked sein
}
func (t *TaskType) trigger() {
	// je nach typ sollte ein TriggerHandler zünden
}

type Flow struct {
	ID                       string   `json:"id"`                // flow id
	ProcessName              string   `json:"process"`           // Network Name
	ParentID                 string   `json:"parent_id"`         // flow id des parents
	ParentTransition         int      `json:"parent_transition"` // die zu feuernde Transition des Parents bei ende des SubFlows
	ActivatedSubFlows        []string `json:"sub_flows"`         // laufende subFlows um bei denen die möglichen Transitionen zu ermitteln (für hateoas)
	Owner                    string   `json:"owner"`             // Owner
	AvailableUserTransitions []int    `json:"usertasks"`         // enabled transitions von user tasks
	TransitionsInProgress    []int    `json:"activated_timers"`  // enabled timers, ActivatedTimers, subflows,...
	Net                      petrinet.Net                        // the running net
	Process                  Process  `json:"process"`
}

type Process struct {
	Name            string       `json:"name"`        // Network Name
	InputMatrix     [][]int      `json:"-"`           // Input Matrix
	OutputMatrix    [][]int      `json:"-"`           // Output Matrix
	ConditionMatrix [][]string   `json:"-"`           // Condition Matrix
	TransitionTypes []int        `json:"-"`           // Transition Types
	InitialState    []int        `json:"-"`           // Initial State
	Transitions     []Transition `json:"transitions"` // detailangaben zur transition
	Variables       [] Variable  `json:"variables"`
	StartVariables  []string     `json:"startvariables"`
	SystemTrigger   Trigger //system trigger handle
	PostFire        Trigger //post fire hood handle
}

// interface um bei autofire zu zünden
type Trigger func(taskType TaskType, flow *Flow, transitionIndex int) bool

// eine Transition
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

// um Yaml einzulesen
type importNet struct {
	Title          string       `json:"title"`
	Transitions    []Transition `json:"transitions"`
	Variables      []Variable   `json:"variables"`
	StartVariables []string     `json:"startvariables"`
	Places         []Place      `json:"places"`
	Arcs           []Arc        `json:"arcs"`
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
