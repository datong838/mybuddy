package eino

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/openocta/openocta/pkg/agent/stream"
	"github.com/openocta/openocta/pkg/agent/types"
)

const (
	turnItemMessage  = "message"
	turnItemApproval = "approval"
)

// TurnApproval carries user approval for an interrupted tool call.
type TurnApproval struct {
	Approved bool                      `json:"approved"`
	Reason   string                    `json:"reason,omitempty"`
	Targets  map[string]any            `json:"targets,omitempty"`
	Contexts []stream.InterruptContext `json:"contexts,omitempty"`
}

// TurnItem is a gob-safe queue item for adk.TurnLoop.
type TurnItem struct {
	Kind        string
	RunID       string
	RequestJSON string
	Approval    *TurnApproval
}

type turnSink struct {
	events chan stream.StreamEvent
	once   sync.Once
}

func (s *turnSink) close() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		close(s.events)
	})
}

func (s *turnSink) send(ctx context.Context, evt stream.StreamEvent) {
	defer func() { recover() }()
	if s == nil {
		return
	}
	select {
	case s.events <- evt:
	case <-ctx.Done():
	}
}

var turnSinks sync.Map // runID -> *turnSink

func registerTurnSink(runID string) (*turnSink, chan stream.StreamEvent) {
	runID = strings.TrimSpace(runID)
	out := make(chan stream.StreamEvent, 64)
	s := &turnSink{events: out}
	if runID != "" {
		turnSinks.Store(runID, s)
	}
	return s, out
}

// finishTurnSink removes and closes the per-run event channel.
func finishTurnSink(runID string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	if v, ok := turnSinks.LoadAndDelete(runID); ok {
		if s, ok := v.(*turnSink); ok && s != nil {
			s.close()
		}
	}
}

func unregisterTurnSink(runID string) {
	finishTurnSink(runID)
}

func sinkForRunID(runID string) *turnSink {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil
	}
	if v, ok := turnSinks.Load(runID); ok {
		return v.(*turnSink)
	}
	return nil
}

func encodeTurnRequest(req types.Request) (string, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeTurnRequest(raw string) (types.Request, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.Request{}, fmt.Errorf("empty turn request")
	}
	var req types.Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return types.Request{}, err
	}
	return req, nil
}

// TurnSession binds one chat session to an Eino TurnLoop (preempt / abort / resume).
type TurnSession struct {
	engine       *Engine
	sessionID    string
	checkpointID string
	store        adk.CheckPointStore

	mu              sync.Mutex
	loop            *adk.TurnLoop[TurnItem, *schema.Message]
	loopCtx         context.Context
	loopCancel      context.CancelFunc
	pushCtxMu       sync.Mutex
	pushCtx         context.Context
	running         bool
	lastInterrupt   *stream.InterruptPayload
	needsResumeLoop bool
}

// NewTurnSession creates a session-scoped TurnLoop manager.
func NewTurnSession(engine *Engine, sessionID string) *TurnSession {
	if engine == nil {
		return nil
	}
	sessionID = strings.TrimSpace(sessionID)
	store := engine.CheckPointStore()
	if store == nil {
		store = SharedCheckPointStore(nil)
	}
	return &TurnSession{
		engine:       engine,
		sessionID:    sessionID,
		checkpointID: TurnLoopCheckpointID(sessionID),
		store:        store,
	}
}

// SessionID returns the chat session id.
func (ts *TurnSession) SessionID() string {
	if ts == nil {
		return ""
	}
	return ts.sessionID
}

// LastInterrupt returns the most recent interrupt payload, if any.
func (ts *TurnSession) LastInterrupt() *stream.InterruptPayload {
	if ts == nil {
		return nil
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.lastInterrupt
}

func (ts *TurnSession) newLoopConfig() adk.TurnLoopConfig[TurnItem, *schema.Message] {
	return adk.TurnLoopConfig[TurnItem, *schema.Message]{
		Store:         ts.store,
		CheckpointID:  ts.checkpointID,
		GenInput:      ts.genInput,
		GenResume:     ts.genResume,
		PrepareAgent:  ts.prepareAgent,
		OnAgentEvents: ts.onAgentEvents,
	}
}

func (ts *TurnSession) ensureLoop(ctx context.Context, forceNew bool) *adk.TurnLoop[TurnItem, *schema.Message] {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if forceNew || ts.loop == nil || !ts.running {
		if ts.loopCancel != nil {
			ts.loopCancel()
		}
		// TurnLoop lifetime is session-scoped; per-turn timeouts come from GenInput RunCtx.
		loopCtx, cancel := context.WithCancel(context.Background())
		ts.loopCtx = loopCtx
		ts.loopCancel = cancel
		ts.loop = adk.NewTurnLoop(ts.newLoopConfig())
		ts.running = true
		go func() {
			ts.loop.Run(loopCtx)
			exit := ts.loop.Wait()
			ts.mu.Lock()
			ts.running = false
			if exit != nil {
				var ie *adk.InterruptError
				if errors.As(exit.ExitReason, &ie) {
					ts.needsResumeLoop = true
				}
			}
			ts.mu.Unlock()
		}()
	}
	return ts.loop
}

// PushMessage enqueues a user message. When preempt is true, cancels the in-flight turn at a safe point.
func (ts *TurnSession) PushMessage(ctx context.Context, req types.Request, runID string, preempt bool) (<-chan stream.StreamEvent, error) {
	if ts == nil || ts.engine == nil {
		return nil, ErrRuntimeClosed
	}
	ts.setPushContext(ctx)
	reqJSON, err := encodeTurnRequest(req)
	if err != nil {
		return nil, err
	}
	_, out := registerTurnSink(runID)
	item := TurnItem{Kind: turnItemMessage, RunID: runID, RequestJSON: reqJSON}

	ts.mu.Lock()
	forceNew := ts.needsResumeLoop
	ts.needsResumeLoop = false
	ts.mu.Unlock()

	loop := ts.ensureLoop(ctx, forceNew)
	var pushOpts []adk.PushOption[TurnItem, *schema.Message]
	if preempt {
		pushOpts = append(pushOpts, adk.WithPreempt[TurnItem, *schema.Message](adk.AfterToolCalls))
	}
	ok, _ := loop.Push(item, pushOpts...)
	if !ok {
		unregisterTurnSink(runID)
		return nil, fmt.Errorf("turn loop stopped")
	}
	return out, nil
}

// PushApproval resumes after interrupt with user approval decision.
func (ts *TurnSession) PushApproval(ctx context.Context, runID string, approval TurnApproval) (<-chan stream.StreamEvent, error) {
	if ts == nil || ts.engine == nil {
		return nil, ErrRuntimeClosed
	}
	ts.setPushContext(ctx)
	_, out := registerTurnSink(runID)
	item := TurnItem{Kind: turnItemApproval, RunID: runID, Approval: &approval}

	ts.mu.Lock()
	forceNew := ts.needsResumeLoop
	ts.needsResumeLoop = false
	if approval.Contexts == nil && ts.lastInterrupt != nil {
		approval.Contexts = ts.lastInterrupt.Contexts
		item.Approval = &approval
	}
	ts.mu.Unlock()

	loop := ts.ensureLoop(ctx, forceNew)
	ok, _ := loop.Push(item)
	if !ok {
		unregisterTurnSink(runID)
		return nil, fmt.Errorf("turn loop stopped")
	}
	return out, nil
}

// StopImmediate aborts the current turn loop (Eino TurnLoop Stop).
func (ts *TurnSession) StopImmediate() {
	if ts == nil {
		return
	}
	ts.mu.Lock()
	loop := ts.loop
	ts.mu.Unlock()
	if loop != nil {
		loop.Stop(adk.WithImmediate())
		_ = loop.Wait()
	}
	ts.mu.Lock()
	ts.running = false
	ts.loop = nil
	if ts.loopCancel != nil {
		ts.loopCancel()
		ts.loopCancel = nil
	}
	ts.mu.Unlock()
}

func (ts *TurnSession) genInput(_ context.Context, _ *adk.TurnLoop[TurnItem, *schema.Message], items []TurnItem) (*adk.GenInputResult[TurnItem, *schema.Message], error) {
	for i, item := range items {
		if item.Kind != turnItemMessage {
			continue
		}
		req, err := decodeTurnRequest(item.RequestJSON)
		if err != nil {
			return nil, err
		}
		msgs, err := BuildAgentMessages(req)
		if err != nil {
			return nil, err
		}
		remaining := append([]TurnItem(nil), items[:i]...)
		remaining = append(remaining, items[i+1:]...)
		return &adk.GenInputResult[TurnItem, *schema.Message]{
			RunCtx: ts.turnRunContext(),
			Input: &adk.TypedAgentInput[*schema.Message]{
				Messages:        msgs,
				EnableStreaming: true,
			},
			Consumed:  []TurnItem{item},
			Remaining: remaining,
		}, nil
	}
	return nil, fmt.Errorf("no message item in turn buffer")
}

func (ts *TurnSession) genResume(_ context.Context, _ *adk.TurnLoop[TurnItem, *schema.Message], interrupted, unhandled, newItems []TurnItem) (*adk.GenResumeResult[TurnItem, *schema.Message], error) {
	all := append(append(append([]TurnItem{}, interrupted...), unhandled...), newItems...)
	var approval *TurnApproval
	var consumed []TurnItem
	var remaining []TurnItem
	for _, item := range all {
		switch item.Kind {
		case turnItemApproval:
			if approval == nil {
				approval = item.Approval
				consumed = append(consumed, item)
			} else {
				remaining = append(remaining, item)
			}
		case turnItemMessage:
			consumed = append(consumed, item)
		default:
			remaining = append(remaining, item)
		}
	}
	if approval == nil {
		return nil, fmt.Errorf("approval item required for resume")
	}
	targets := approval.Targets
	if len(targets) == 0 {
		targets = resumeTargetsFromPayload(approval.Contexts, approval.Approved, approval.Reason)
	}
	if len(targets) == 0 {
		ts.mu.Lock()
		if ts.lastInterrupt != nil {
			targets = resumeTargetsFromPayload(ts.lastInterrupt.Contexts, approval.Approved, approval.Reason)
		}
		ts.mu.Unlock()
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no resume targets")
	}
	return &adk.GenResumeResult[TurnItem, *schema.Message]{
		RunCtx:       ts.turnRunContext(),
		ResumeParams: &adk.ResumeParams{Targets: targets},
		Consumed:     consumed,
		Remaining:    remaining,
	}, nil
}

func (ts *TurnSession) prepareAgent(_ context.Context, _ *adk.TurnLoop[TurnItem, *schema.Message], _ []TurnItem) (adk.TypedAgent[*schema.Message], error) {
	agent := ts.engine.Agent()
	if agent == nil {
		return nil, fmt.Errorf("agent not configured")
	}
	typed, ok := agent.(adk.TypedAgent[*schema.Message])
	if !ok {
		return nil, fmt.Errorf("agent does not support message type")
	}
	return typed, nil
}

func (ts *TurnSession) onAgentEvents(ctx context.Context, tc *adk.TurnContext[TurnItem, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
	var runID string
	if len(tc.Consumed) > 0 {
		runID = tc.Consumed[len(tc.Consumed)-1].RunID
	}
	sink := sinkForRunID(runID)
	if sink == nil {
		drainAgentEvents(events)
		return nil
	}
	defer finishTurnSink(runID)

	eventCh := StreamEventsFromIterator(ctx, ts.sessionID, runID, events)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tc.Preempted:
			sink.send(ctx, stream.StreamEvent{
				Type:      stream.EventMessageStop,
				SessionID: ts.sessionID,
				Delta:     &stream.Delta{StopReason: "preempted"},
			})
			return nil
		case <-tc.Stopped:
			sink.send(ctx, stream.StreamEvent{
				Type:      stream.EventMessageStop,
				SessionID: ts.sessionID,
				Delta:     &stream.Delta{StopReason: "aborted"},
			})
			return nil
		case evt, ok := <-eventCh:
			if !ok {
				return nil
			}
			if evt.Type == stream.EventRunInterrupted && evt.Interrupt != nil {
				ts.mu.Lock()
				cp := *evt.Interrupt
				ts.lastInterrupt = &cp
				ts.needsResumeLoop = true
				ts.mu.Unlock()
			}
			sink.send(ctx, evt)
			if evt.Type == stream.EventError {
				return nil
			}
			if evt.Type == stream.EventMessageStop && messageStopEndsAgentEventStream(evt) {
				return nil
			}
		}
	}
}

// messageStopEndsAgentEventStream reports whether a message_stop ends the agent event stream.
// tool_use stops mark a model turn boundary; the same agent run continues with tool execution.
func messageStopEndsAgentEventStream(evt stream.StreamEvent) bool {
	if evt.Type != stream.EventMessageStop {
		return false
	}
	reason := ""
	if evt.Delta != nil {
		reason = strings.TrimSpace(evt.Delta.StopReason)
	}
	switch reason {
	case "tool_use", "":
		return false
	default:
		return true
	}
}

func drainAgentEvents(events *adk.AsyncIterator[*adk.AgentEvent]) {
	for {
		if _, ok := events.Next(); !ok {
			return
		}
	}
}

func (ts *TurnSession) setPushContext(ctx context.Context) {
	ts.pushCtxMu.Lock()
	ts.pushCtx = ctx
	ts.pushCtxMu.Unlock()
}

func (ts *TurnSession) turnRunContext() context.Context {
	ts.pushCtxMu.Lock()
	push := ts.pushCtx
	ts.pushCtxMu.Unlock()
	if push == nil {
		push = context.Background()
	}
	var budget time.Duration
	if ts.engine != nil {
		budget = ts.engine.AgentRunBudget()
	}
	return wrapRunContext(push, budget)
}
