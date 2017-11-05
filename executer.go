package bpnet

import (
	"github.com/veith/petrinet"
	"time"
	"github.com/oklog/ulid"
	"math/rand"
	"errors"
)

// Creates a flow (process instance) from a process
func (p Process) CreateFlow(Owner string) Flow {
	flow := Flow{Owner: Owner, Process: p, ID: makeUlid()}
	flow.ProcessName = p.Name
	flow.Net.InputMatrix = p.InputMatrix
	flow.Net.OutputMatrix = p.OutputMatrix
	flow.Net.State = make([]int, len(p.InitialState))
	copy(flow.Net.State, p.InitialState)
	flow.Net.ConditionMatrix = make([][]string, len(p.ConditionMatrix))
	copy(flow.Net.ConditionMatrix, p.ConditionMatrix)

	flow.TransitionsInProgress = make(map[int]int)
	return flow
}

// read flow data
func (flow *Flow) ReadData() map[string]interface{} {
	return flow.Net.Variables
}

// starts the flow with initial data
func (flow *Flow) Start(data map[string]interface{}) error {
	//init
	if flow.Process.OnSubProcessStarted != nil && flow.ParentTransitionTokenID != 0 {
		flow.Process.OnSubProcessStarted(flow, flow.ParentTransitionTokenID)
	} else {
		if flow.Process.OnProcessStarted != nil {
			flow.Process.OnProcessStarted(flow, 0)
		}
	}
	flow.Net.Variables = make(map[string]interface{})

	err := flow.appendData(data, "___start")
	if err.Len() == 0 {
		flow.Net.Init()
		flow.AvailableUserTransitions = flow.bpnTransitionsCheck();
	}
	return err
}

type RequiredError struct {
	Fields []string
	error
}

func (e *RequiredError) Len() int {
	return len(e.Fields)
}

// // prüfe ob alle in der Transition VERLANGTEN Daten gesendet wurden
func (flow *Flow) appendData(data map[string]interface{}, transitionID string) RequiredError {

	// filter undefined fields
	for _, v := range flow.Process.Variables {
		if val, ok := data[v.ID]; ok {
			flow.Net.Variables[v.ID] = val
		}
	}

	// check for required fields
	var requiredError RequiredError
	if transitionID == "___start" {
		for _, required := range flow.Process.StartVariables {
			if _, ok := data[required]; !ok {
				requiredError.error = errors.New("required variables for start not sent")
				requiredError.Fields = append(requiredError.Fields, required)
			}
		}
	} else {
		// bei transitionen
		for _, transition := range flow.Process.Transitions {
			// build map
			// nur für aktuell gewählte transition prüfen
			if transition.ID == transitionID {
				for _, required := range transition.ReqVariables {
					if _, ok := data[required]; !ok {
						requiredError.error = errors.New("required variables for start not sent")
						requiredError.Fields = append(requiredError.Fields, required)
					}
				}
			}
		}
	}

	return requiredError

}

// Fire a transition / task
func (f *Flow) Fire(transitionIndex int, data map[string]interface{}) error {
	var err RequiredError
	if len(f.Process.Transitions) > transitionIndex {
		err = f.appendData(data, f.Process.Transitions[transitionIndex].ID)
	}

	if err.Len() == 0 {
		err := f.fire(transitionIndex)
		if err == nil {
			f.AvailableUserTransitions = f.bpnTransitionsCheck();
			if f.Process.OnFireCompleted != nil {
				f.Process.OnFireCompleted(f, transitionIndex)
			}
			return nil
		}
		return err
	}
	return err

}

// checks and notify completion of process and sub process
func (f *Flow) checkCompleted() bool {
	if len(f.Net.EnabledTransitions) == 0 {
		if f.ParentTransitionTokenID != 0 {

			// fire parent token

			if f.Process.OnSubProcessCompleted != nil {
				f.Process.OnSubProcessCompleted(f, f.ParentTransitionTokenID)
			}

			parentFlow, err := f.Process.FlowInstanceLoader(f.ParentID)
			if err == nil {

				parentFlow.FireSystemTask(f.ParentTransitionTokenID, f.Net.Variables)
			}

		} else {
			if f.Process.OnProcessCompleted != nil {
				f.Process.OnProcessCompleted(f, 0)
			}
		}
		return true
	}
	return false
}

// fire without notification
func (f *Flow) fire(transitionIndex int) error {
	err := f.Net.Fire(transitionIndex)
	if err == nil {
		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		return nil
	}
	return err
}

// fires a system task with tokenID
func (f *Flow) FireSystemTask(tokenID int, data map[string]interface{}) error {
	// daten einspielen
	err := f.appendData(data, f.Process.Transitions[f.TransitionsInProgress[tokenID]].ID)
	if err.Len() == 0 {
		return f.fireWithTokenId(tokenID)
	}
	return err
}

// fires a transition from a token in transition
func (f *Flow) fireWithTokenId(tokenID int) error {

	transition := f.TransitionsInProgress[tokenID]

	err := f.Net.FireWithTokenId(transition, tokenID)

	delete(f.TransitionsInProgress, tokenID)

	if err == nil {
		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		return nil
	}

	return err
}

// check transitions for automatic fire or triggers the different types (AUTO, MESSAGE,...)
func (f *Flow) bpnTransitionsCheck() []int {
	f.checkCompleted()

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
				if f.Process.OnSendMessage != nil && f.Process.OnSendMessage(f, transition) {
					f.fire(transition)
				} else {
					panic("OnSendMessage not available")
				}
				break
			}

		}
	}

	for _, transition := range f.Net.EnabledTransitions {
		// places in transition
		for place, val := range f.Net.InputMatrix[transition] {
			if (val > 0) {
				for _, tokenID := range f.Net.TokenIds[place] {

					// alle noch nicht benachrichtigten timer prüfen
					if f.Process.TransitionTypes[transition] == int(TIMED) && !f.tokenRegistred(tokenID) {
						f.TransitionsInProgress[tokenID] = transition
						executeTimer(f, transition, tokenID)
					}

					// systemtask
					if f.Process.TransitionTypes[transition] == int(SYSTEM) && !f.tokenRegistred(tokenID) {
						if f.Process.OnSystemTask(f, tokenID) {
							f.TransitionsInProgress[tokenID] = transition
						}
					}

					// subprocess
					if f.Process.TransitionTypes[transition] == int(SUBPROCESS) && !f.tokenRegistred(tokenID) {

						f.TransitionsInProgress[tokenID] = transition
						// sub erstellen und starten
						subprocess, err := f.Process.ProcessDefinitionLoader(f.Process.Transitions[transition].Details["subprocess"].(string))

						if err == nil {
							subflow := subprocess.CreateFlow(f.Owner)
							subflow.ParentID = f.ID
							subflow.ParentTransitionTokenID = tokenID
							f.RunningSubProcesses = append(f.RunningSubProcesses, subflow.ID)
							//starte mit daten des flows
							subflow.Start(f.Net.Variables)
						}

					}
				}
			}

		}

	}

	return f.Net.EnabledTransitions
}

// executes a timer
func executeTimer(f *Flow, transition int, tokenID int) {

	if f.Process.OnTimerStarted != nil {
		f.Process.OnTimerStarted(f, transition)
	}

	// verzögert auslösen

	time.AfterFunc(parseDelay(f.Process.Transitions[transition].Details["delay"]), func() {
		if f.tokenRegistred(tokenID) {
			//f.fireWithTokenId(tokenID)
			err := f.Net.FireWithTokenId(transition, tokenID)

			if f.Process.OnTimerCompleted != nil {
				f.Process.OnTimerCompleted(f, transition)
			}
			if err == nil {
				f.AvailableUserTransitions = f.bpnTransitionsCheck()
			}
			delete(f.TransitionsInProgress, tokenID)
		}
	})
}

// check if a token is already registred in inTransition
func (f *Flow) tokenRegistred(tokenID int) bool {
	if _, ok := f.TransitionsInProgress[tokenID]; ok {
		//do something here
		return true
	}

	return false
}

// prüfe ob enablete Transitionen mit Autofeuer existieren
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
	var delay float64
	var err error
	switch s.(type) {
	case float64:
		delay = s.(float64)
	case int:
		delay = float64(s.(int))
	}

	if err == nil && delay != 0 {
		return time.Duration(delay) * 1000 * time.Millisecond
	} else {
		return time.Duration(100) * time.Millisecond
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

// erstellt eine ulid
func makeUlid() ulid.ULID {
	return ulid.MustNew(ulid.Now(), rand.New(rand.NewSource(time.Now().UnixNano())))

}

// TaskTypen
// SYSTEM:
// Prozesse, AI können via transition details

const (
	AUTO       TaskType = 1 // feuert selbst ab
	USER       TaskType = 2 // claim in transition details (assign macht system)
	MESSAGE    TaskType = 3 // sendet via Trigger
	TIMED      TaskType = 4 // feuert durch Zeit
	SUBPROCESS TaskType = 5 // startet einen subprozess
	SYSTEM     TaskType = 6 // assign in transition details (claim macht user)

)

type TaskType int
type Flow struct {
	ID                       ulid.ULID   `json:"id"`                // flow id
	ProcessName              string      `json:"process"`           // Network Name
	ParentID                 ulid.ULID   `json:"parent_id"`         // flow id des parents
	ParentTransitionTokenID  int         `json:"parent_transition"` // die zu feuernde Transition des Parents bei ende des SubFlows
	ActivatedSubFlows        []string    `json:"sub_flows"`         // laufende subFlows um bei denen die möglichen Transitionen zu ermitteln (für hateoas)
	Owner                    string      `json:"owner"`             // Owner
	AvailableUserTransitions []int       `json:"usertasks"`         // enabled transitions von user tasks
	TransitionsInProgress    map[int]int `json:"in_progress"`       // [tokenID]transition enabled timers, ActivatedTimers, subflows,...
	Net                      petrinet.Net                           // the running net
	Process                  Process     `json:"process"`
	RunningSubProcesses      []ulid.ULID `json:"running_sub_processes"`
}

type Process struct {
	Name                    string                  `json:"name"`        // Network Name
	InputMatrix             [][]int                 `json:"-"`           // Input Matrix
	OutputMatrix            [][]int                 `json:"-"`           // Output Matrix
	ConditionMatrix         [][]string              `json:"-"`           // Condition Matrix
	TransitionTypes         []int                   `json:"-"`           // Transition Types
	InitialState            []int                   `json:"-"`           // Initial State
	Transitions             []Transition            `json:"transitions"` // detailangaben zur transition
	Variables               [] Variable             `json:"variables"`
	StartVariables          []string                `json:"startvariables"`
	OnSystemTask            SystemTask              `json:"_"` //system task handle TODO: check https://siadat.github.io/post/context
	OnFireCompleted         Notify                  `json:"_"` // nach jedem erfolgreichen Fire
	OnTimerStarted          Notify                  `json:"_"` //timer hook handle
	OnTimerCompleted        Notify                  `json:"_"` //timer hook handle
	OnSendMessage           Notify                  `json:"_"` // message send handler
	OnFlowCreated           Notify                  `json:"_"` // process started hook, after autofireing hooks
	OnProcessStarted        Notify                  `json:"_"` // process started hook, after autofireing hooks
	OnProcessCompleted      Notify                  `json:"_"` // process finished
	OnSubProcessStarted     Notify                  `json:"_"`
	OnSubProcessCompleted   Notify                  `json:"_"`
	FlowInstanceLoader      FlowInstanceLoader      `json:"_"` // prozessinstanzen um parent prozesse oder subprozesse zu referenzieren
	ProcessDefinitionLoader ProcessDefinitionLoader `json:"_"` // prozessdefinitionen um subprozesse zu starten
}

// interface um bei autofire zu zünden

type Notify func(flow *Flow, transitionIndex int) bool
type SystemTask func(flow *Flow, tokenID int) bool
type ProcessDefinitionLoader func(processName string) (*Process, error)
type FlowInstanceLoader func(flowID ulid.ULID) (*Flow, error)
