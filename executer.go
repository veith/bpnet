package bpnet

import (
	"github.com/veith/petrinet"
	"time"
	"strconv"
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
		if f.Process.OnFireCompleted != nil {
			f.Process.OnFireCompleted(AUTO, f, transitionIndex)
		}
		return nil
	}
	return err
}

// Fire
func (f *Flow) FireSytemTask(transitionIndex int) error {
	var err error
	if transitionRegistred(transitionIndex, f.TransitionsInProgress) {
		err = f.fire(transitionIndex)
		if err == nil {
			f.AvailableUserTransitions = f.bpnTransitionsCheck();
			f.AvailableSystemTransitions = removeFromIntValArray(f.AvailableSystemTransitions, transitionIndex)
			if f.Process.OnFireCompleted != nil {
				f.Process.OnFireCompleted(AUTO, f, transitionIndex)
			}
			return nil
		}
	}
	return err
}

//Entfernt einen Wert aus einem int slice
func removeFromIntValArray(l []int, item int) []int {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

// fire ohne benachrichtigung nach aussen
func (f *Flow) fire(transitionIndex int) error {
	err := f.Net.Fire(transitionIndex)
	if err == nil {
		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		return nil
	}
	return err
}

// prüfen der Transitionen und auslösen der autofeuernden triggerpunkte
func (f *Flow) bpnTransitionsCheck() []int {
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

		// timer trigger
		// todo timer storage ermöglichen (eventuell über methode?, damit nicht Fire verwendet werden muss)
		// alle noch nicht benachrichtigten timer prüfen
		if f.Process.TransitionTypes[transition] == int(TIMED) && !transitionRegistred(transition, f.TransitionsInProgress) {
			f.TransitionsInProgress = append(f.TransitionsInProgress, transition)

			if f.Process.OnTimerStarted != nil {
				f.Process.OnTimerStarted(TIMED, f, transition)
			}
			// verzögert auslösen
			time.AfterFunc(parseDelay(f.Process.Transitions[transition].Details["delay"]), func() {
				f.fire(transition)
				if f.Process.OnTimerCompleted != nil {
					f.Process.OnTimerCompleted(TIMED, f, transition)
				}
			})

		}
		// alle noch nicht gestarteten subprocesses
		if f.Process.TransitionTypes[transition] == int(SUBPROCESS) && !transitionRegistred(transition, f.TransitionsInProgress) {
			if f.Process.SystemTrigger(SUBPROCESS, f, transition) {
				f.TransitionsInProgress = append(f.TransitionsInProgress, transition)
			}
		}
		// systemtask
		if f.Process.TransitionTypes[transition] == int(SYSTEM) && !transitionRegistred(transition, f.TransitionsInProgress) {
			f.AvailableSystemTransitions = append(f.AvailableSystemTransitions, transition)
			if f.Process.SystemTrigger(SYSTEM, f, transition) {
				f.TransitionsInProgress = append(f.TransitionsInProgress, transition)
			}
		}
	}
	return f.Net.EnabledTransitions
}

// prüfe ob ein Timer bereits angestossen wurde.
func transitionRegistred(t int, list []int) bool {
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
		if f.Process.TransitionTypes[transition] == int(AUTO) {
			return true
		}
		// auf alle autofire pruefen
		if f.Process.TransitionTypes[transition] == int(MESSAGE) {
			return true
		}
	}
	return false
}

// momentan einfach nur sekunden, aber ist ausbaubar zu eow, eom, 1h,...
func parseDelay(s interface{}) time.Duration {
	var delay int
	var err error

	switch s.(type) {
	case float64:
		delay = int(s.(float64))
	case int:
		delay = s.(int)
	case string:
		delay, err = strconv.Atoi(s.(string))
	}

	if err == nil {
		return time.Duration(max(1, delay)) * time.Second
	} else {
		return time.Duration(1) * time.Second
	}
}

//  die eingebaute max kann nicht mit int umgehen :-(
// max von zwei Int
func max(a, b int) int {
	if a < b {
		return b
	}
	return a
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

type Flow struct {
	ID                         string   `json:"id"`                // flow id
	ProcessName                string   `json:"process"`           // Network Name
	ParentID                   string   `json:"parent_id"`         // flow id des parents
	ParentTransition           int      `json:"parent_transition"` // die zu feuernde Transition des Parents bei ende des SubFlows
	ActivatedSubFlows          []string `json:"sub_flows"`         // laufende subFlows um bei denen die möglichen Transitionen zu ermitteln (für hateoas)
	Owner                      string   `json:"owner"`             // Owner
	AvailableUserTransitions   []int    `json:"usertasks"`         // enabled transitions von user tasks
	AvailableSystemTransitions []int    `json:"systemtasks"`       // enabled transitions von user tasks
	TransitionsInProgress      []int    `json:"in_progress"`       // enabled timers, ActivatedTimers, subflows,...
	Net                        petrinet.Net                        // the running net
	Process                    Process  `json:"process"`
}

type Process struct {
	Name             string       `json:"name"`        // Network Name
	InputMatrix      [][]int      `json:"-"`           // Input Matrix
	OutputMatrix     [][]int      `json:"-"`           // Output Matrix
	ConditionMatrix  [][]string   `json:"-"`           // Condition Matrix
	TransitionTypes  []int        `json:"-"`           // Transition Types
	InitialState     []int        `json:"-"`           // Initial State
	Transitions      []Transition `json:"transitions"` // detailangaben zur transition
	Variables        [] Variable  `json:"variables"`
	StartVariables   []string     `json:"startvariables"`
	SystemTrigger    Trigger //system trigger handle
	OnFireCompleted  Trigger //post fire hook handle
	OnTimerStarted   Trigger //timer hook handle
	OnTimerCompleted Trigger //timer hook handle
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
