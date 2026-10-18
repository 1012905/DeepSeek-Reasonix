package sidecar

import (
	"context"
	"encoding/json"
	"testing"

	"reasonix/internal/ext/extension/protocol"
	"reasonix/internal/ext/pluginpkg"
)

type sessionUI struct {
	unavailableUIHandler
	reads, submits int
}

func (s *sessionUI) ReadSession(_ context.Context, p protocol.SessionReadParams) (protocol.SessionReadResult, error) {
	s.reads++
	return protocol.SessionReadResult{Messages: []protocol.SessionMessage{{Role: "user", Text: string(p.What)}}}, nil
}

func (s *sessionUI) SubmitPrompt(_ context.Context, p protocol.PromptSubmitParams) (protocol.PromptSubmitResult, error) {
	s.submits++
	return protocol.PromptSubmitResult{Queued: p.Text != ""}, nil
}

func sessionClient(caps []string, ui UIHandler) *Client {
	return &Client{pluginID: "mod-x", rt: &pluginpkg.RuntimeSpec{Capabilities: caps}, ui: ui}
}

// Session reads and prompt submissions are served only to a runtime that
// declared the session capability, whatever the host could answer.
func TestSessionCallsNeedTheDeclaredCapability(t *testing.T) {
	ui := &sessionUI{}
	read := json.RawMessage(`{"what":"messages","limit":5}`)
	submit := json.RawMessage(`{"text":"go"}`)

	without := sessionClient([]string{"interceptors", "ui"}, ui)
	if _, err := without.handleSessionRead(context.Background(), read); err == nil {
		t.Fatal("a read was served without the session capability")
	}
	if _, err := without.handlePromptSubmit(context.Background(), submit); err == nil {
		t.Fatal("a submit was served without the session capability")
	}
	if ui.reads+ui.submits != 0 {
		t.Fatalf("the host was asked %d reads and %d submits before the capability check", ui.reads, ui.submits)
	}

	with := sessionClient([]string{"session"}, ui)
	got, err := with.handleSessionRead(context.Background(), read)
	if err != nil || got.(protocol.SessionReadResult).Messages[0].Text != "messages" {
		t.Fatalf("read = %v, %v", got, err)
	}
	if got, err := with.handlePromptSubmit(context.Background(), submit); err != nil || !got.(protocol.PromptSubmitResult).Queued {
		t.Fatalf("submit = %v, %v", got, err)
	}
}

// A host that serves UI but not sessions says so, the same way it does for UI.
func TestSessionCallsWithoutAHostHandlerAreUnknownMethods(t *testing.T) {
	c := sessionClient([]string{"session"}, unavailableUIHandler{})
	if _, err := c.handleSessionRead(context.Background(), json.RawMessage(`{"what":"usage"}`)); err == nil {
		t.Fatal("a read was served with no host handler")
	}
	if _, err := c.handlePromptSubmit(context.Background(), json.RawMessage(`{"text":""}`)); err == nil {
		t.Fatal("an empty submit was accepted")
	}
}
