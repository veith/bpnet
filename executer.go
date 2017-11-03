package bpnet

import (
	"github.com/veith/petrinet"
	"time"
	"strconv"
	"github.com/oklog/ulid"
	"math/rand"
)

func (p Process) CreateFlow(Owner string) *Flow {
	flow := Flow{Owner: Owner, Process: p, ID: makeUlid()}
	flow.ProcessName = p.Name
	flow.Net.InputMatrix = p.InputMatrix
	flow.Net.OutputMatrix = p.OutputMatrix
	flow.Net.State = make([]int, len(p.InitialState))
	copy(flow.Net.State, p.InitialState)
	flow.Net.Init()

	flow.TransitionsInProgress = make(map[int]int)
	return &flow
}

func (flow *Flow) Start(data map[string]interface{}) {
	//init
	if flow.Process.OnSubprocessStarted != nil && flow.ParentTransitionTokenID != 0 {
		flow.Process.OnSubprocessStarted(flow, flow.ParentTransitionTokenID)
	} else {
		if flow.Process.OnProcessStarted != nil {
			flow.Process.OnProcessStarted(flow, 0)
		}
	}

	flow.Net.Variables = data
	flow.AvailableUserTransitions = flow.bpnTransitionsCheck();
}

// Fire
func (f *Flow) Fire(transitionIndex int) error {
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

//Entfernt einen Wert aus einem int slice
func removeFromIntValArray(l []int, item int) []int {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

func (f *Flow) checkCompleted() bool {
	if len(f.Net.EnabledTransitions) == 0 {

		if f.ParentTransitionTokenID != 0 {

			// fire parent token

			if f.Process.OnSubprocessCompleted != nil {
				f.Process.OnSubprocessCompleted(f, f.ParentTransitionTokenID)
			}

			parentFlow, err := f.Process.FlowInstanceLoader(f.ParentID)
			if err == nil {
				parentFlow.fireFromSubProccess(f.ParentTransitionTokenID)
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

// fire ohne benachrichtigung nach aussen
func (f *Flow) fire(transitionIndex int) error {
	err := f.Net.Fire(transitionIndex)
	if err == nil {
		f.AvailableUserTransitions = f.bpnTransitionsCheck();
		return nil
	}
	return err
}

func (f *Flow) FireSystemTask(tokenID int) error {
	return f.fireWithTokenId(tokenID)
}

// fire ohne benachrichtigung nach aussen
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

// fire ohne benachrichtigung nach aussen
func (f *Flow) fireFromSubProccess(tokenID int) error {

	transition := f.TransitionsInProgress[tokenID]

	err := f.Net.FireWithTokenId(transition, tokenID)

	delete(f.TransitionsInProgress, tokenID)

	if err == nil {
		return nil
	}

	return err
}

// prüfen der Transitionen und auslösen der autofeuernden triggerpunkte
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

							subflow.Start(f.Net.Variables)
						}

					}
				}
			}

		}

	}

	return f.Net.EnabledTransitions
}
func executeTimer(f *Flow, transition int, tokenID int) {

	if f.Process.OnTimerStarted != nil {
		f.Process.OnTimerStarted(f, transition)
	}

	// verzögert auslösen

	time.AfterFunc(parseDelay(f.Process.Transitions[transition].Details["delay"]), func() {
		if f.tokenRegistred(tokenID) {
			f.fireWithTokenId(tokenID)
			if f.Process.OnTimerCompleted != nil {
				f.Process.OnTimerCompleted(f, transition)
			}
		}
	})
}

// prüfe ob ein Timer bereits angestossen wurde.
func (f *Flow) tokenRegistred(tokenID int) bool {
	if _, ok := f.TransitionsInProgress[tokenID]; ok {
		//do something here
		return true
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
	SUBPROCESS TaskType = 5 // startet einen subprozess, TODO: automatisches anlegen eines Auto vor dem subprozess start
	SYSTEM     TaskType = 6 // assign in transition details (claim macht user)

)

type TaskType int
type Flow struct {
	ID                       ulid.ULID   `json:"id"`                // flow id
	ProcessName              string      `json:"process"`           // Network Name
	ParentID                 ulid.ULID   `json:"parent_id"`         // flow id des parents
	ParentTransitionTokenID  int         `json:"parent_transition"` // die zu feuernde Transition des Parents bei ende des SubFlows
	IsSubflow                bool
	ActivatedSubFlows        []string    `json:"sub_flows"`   // laufende subFlows um bei denen die möglichen Transitionen zu ermitteln (für hateoas)
	Owner                    string      `json:"owner"`       // Owner
	AvailableUserTransitions []int       `json:"usertasks"`   // enabled transitions von user tasks
	TransitionsInProgress    map[int]int `json:"in_progress"` // [tokenID]transition enabled timers, ActivatedTimers, subflows,...
	Net                      petrinet.Net                     // the running net
	Process                  Process     `json:"process"`
	RunningSubProcesses      []ulid.ULID `json:"running_sub_processes"`
}

type Process struct {
	Name                    string       `json:"name"`        // Network Name
	InputMatrix             [][]int      `json:"-"`           // Input Matrix
	OutputMatrix            [][]int      `json:"-"`           // Output Matrix
	ConditionMatrix         [][]string   `json:"-"`           // Condition Matrix
	TransitionTypes         []int        `json:"-"`           // Transition Types
	InitialState            []int        `json:"-"`           // Initial State
	Transitions             []Transition `json:"transitions"` // detailangaben zur transition
	Variables               [] Variable  `json:"variables"`
	StartVariables          []string     `json:"startvariables"`
	OnSystemTask            SystemTask //system task handle TODO: https://siadat.github.io/post/context
	OnFireCompleted         Notify     // nach jedem erfolgreichen Fire
	OnTimerStarted          Notify     //timer hook handle
	OnTimerCompleted        Notify     //timer hook handle
	OnSendMessage           Notify     // message send handler
	OnFlowCreated           Notify     // process started hook, after autofireing hooks
	OnProcessStarted        Notify     // process started hook, after autofireing hooks
	OnProcessCompleted      Notify     // process finished
	OnSubprocessStarted     Notify
	OnSubprocessCompleted   Notify
	FlowInstanceLoader      FlowInstanceLoader      // prozessinstanzen um parent prozesse oder subprozesse zu referenzieren
	ProcessDefinitionLoader ProcessDefinitionLoader // prozessdefinitionen um subprozesse zu starten
}

// interface um bei autofire zu zünden

type Notify func(flow *Flow, transitionIndex int) bool
type SystemTask func(flow *Flow, tokenID int) bool
type ProcessDefinitionLoader func(processName string) (*Process, error)
type FlowInstanceLoader func(flowID ulid.ULID) (*Flow, error)

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
