package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
	"reasonix/internal/contract/tool"
)

type proposal struct {
	receipt tool.SettingReceipt
	err     error
}

func settingController(t *testing.T) (*Controller, chan event.Approval, chan event.Event) {
	t.Helper()
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	requests := make(chan event.Approval, 8)
	notices := make(chan event.Event, 8)
	c := New(Options{Sink: event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.ApprovalRequest:
			requests <- e.Approval
		case event.Notice:
			notices <- e
		}
	})})
	return c, requests, notices
}

func propose(c *Controller, id, value, reason string) chan proposal {
	done := make(chan proposal, 1)
	go func() {
		r, err := c.ProposeSetting(context.Background(), id, json.RawMessage(value), reason)
		done <- proposal{r, err}
	}()
	return done
}

func userFile() *config.Config { return config.LoadForEdit(config.UserConfigPath()) }

func asRefusal(err error) (tool.Refusal, bool) {
	var r tool.Refusal
	ok := errors.As(err, &r)
	return r, ok
}

func refusalCode(err error) string {
	r, _ := asRefusal(err)
	return r.Code
}

func idle(t *testing.T, done chan proposal, what string) {
	t.Helper()
	select {
	case got := <-done:
		t.Fatalf("%s: answered without a person: %+v", what, got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestAPreferenceChangeWaitsForAPersonEvenInYoloAndWritesOnTheClick(t *testing.T) {
	c, requests, notices := settingController(t)
	c.SetAutoApproveTools(true)
	was := userFile().UI.ShowReceipt
	done := propose(c, "ui.show_receipt", jsonBool(!was), "the receipt card is in the way")
	a := nextApproval(t, requests)
	if a.Tool != SettingApprovalTool || a.Subject != "ui.show_receipt" || !a.Fresh || !a.AllowsSession || a.AllowsPersist {
		t.Fatalf("approval = %+v", a)
	}
	for _, want := range []string{"ui.show_receipt", "in the way"} {
		if !strings.Contains(a.Reason, want) {
			t.Errorf("the card does not say %q: %q", want, a.Reason)
		}
	}
	idle(t, done, "yolo")
	if userFile().UI.ShowReceipt != was {
		t.Fatal("the file changed before the click")
	}
	c.Approve(a.ID, true, false, false)
	got := <-done
	if got.err != nil || got.receipt.Via != "click" || got.receipt.From != was || got.receipt.To != !was {
		t.Fatalf("receipt = %+v err=%v", got.receipt, got.err)
	}
	if userFile().UI.ShowReceipt == was {
		t.Fatal("the click did not change the file")
	}
	select {
	case n := <-notices:
		if n.Code != event.NoticeCodeSettingChanged || n.Detail != "ui.show_receipt" || !strings.Contains(n.Text, "ui.show_receipt") {
			t.Fatalf("notice = %+v", n)
		}
	case <-time.After(time.Second):
		t.Fatal("no receipt notice")
	}
}

func jsonBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestAnElevatedChangeOffersNoSessionGrantAndAsksEveryTime(t *testing.T) {
	c, requests, _ := settingController(t)
	done := propose(c, "peers.enabled", "true", "")
	a := nextApproval(t, requests)
	if a.Tool != SettingElevatedApprovalTool || a.AllowsSession {
		t.Fatalf("approval = %+v, want the elevated card with no session grant", a)
	}
	c.Approve(a.ID, true, true, false)
	if got := <-done; got.err != nil || got.receipt.Via != "click" {
		t.Fatalf("receipt = %+v err=%v", got.receipt, got.err)
	}
	c.settings.proposals.Store(0)
	again := propose(c, "peers.enabled", "false", "")
	b := nextApproval(t, requests)
	if b.Tool != SettingElevatedApprovalTool {
		t.Fatalf("second card = %+v", b)
	}
	c.Approve(b.ID, false, false, false)
	<-again
}

func TestASessionGrantCoversThatOnePreferenceAndNoOther(t *testing.T) {
	c, requests, _ := settingController(t)
	was := userFile().UI.ShowReceipt
	first := propose(c, "ui.show_receipt", jsonBool(!was), "")
	a := nextApproval(t, requests)
	c.Approve(a.ID, true, true, false)
	<-first
	second := <-propose(c, "ui.show_receipt", jsonBool(was), "")
	if second.err != nil || second.receipt.Via != "session_grant" {
		t.Fatalf("second = %+v err=%v, want it applied by the grant", second.receipt, second.err)
	}
	other := propose(c, "ui.show_changed_files", jsonBool(!userFile().UI.ShowChangedFiles), "")
	card := nextApproval(t, requests)
	idle(t, other, "a different setting")
	c.Approve(card.ID, false, false, false)
	<-other
}

func TestAGuardedSettingIsRefusedWithAnOpenRemedyAndNoCard(t *testing.T) {
	c, requests, _ := settingController(t)
	before := userFile().Permissions.Mode
	_, err := c.ProposeSetting(context.Background(), "permissions.mode", json.RawMessage(`"yolo"`), "faster")
	r, ok := asRefusal(err)
	if !ok || r.Code != "setting.guarded" || r.Remedy == nil || r.Remedy.Op != tool.RemedyOpen || r.Remedy.Setting != "permissions.mode" {
		t.Fatalf("err = %#v", err)
	}
	select {
	case a := <-requests:
		t.Fatalf("a card was shown for a boundary setting: %+v", a)
	default:
	}
	if userFile().Permissions.Mode != before {
		t.Fatal("a guarded setting was written")
	}
}

func TestADeclinedChangeLeavesTheFileAndSaysSo(t *testing.T) {
	c, requests, _ := settingController(t)
	was := userFile().UI.ShowReceipt
	done := propose(c, "ui.show_receipt", jsonBool(!was), "")
	a := nextApproval(t, requests)
	c.Approve(a.ID, false, false, false)
	got := <-done
	if refusalCode(got.err) != CodeSettingDeclined || userFile().UI.ShowReceipt != was {
		t.Fatalf("err = %v, show = %v", got.err, userFile().UI.ShowReceipt)
	}
}

func TestAValueTheKeyCannotHoldIsRefusedBeforeAnyCard(t *testing.T) {
	c, requests, _ := settingController(t)
	_, err := c.ProposeSetting(context.Background(), "ui.show_receipt", json.RawMessage(`"maybe"`), "")
	if refusalCode(err) != "setting.invalid_value" {
		t.Fatalf("err = %v", err)
	}
	select {
	case a := <-requests:
		t.Fatalf("a card for an unusable value: %+v", a)
	default:
	}
}

func TestAChangeToTheValueAlreadyHeldAsksNothing(t *testing.T) {
	c, requests, _ := settingController(t)
	got, err := c.ProposeSetting(context.Background(), "ui.show_receipt", json.RawMessage(jsonBool(userFile().UI.ShowReceipt)), "")
	if err != nil || got.Via != "unchanged" {
		t.Fatalf("got = %+v err=%v", got, err)
	}
	select {
	case a := <-requests:
		t.Fatalf("a card for no change: %+v", a)
	default:
	}
}

func TestOnlyThreeProposalsAreAskedPerTurn(t *testing.T) {
	c, requests, _ := settingController(t)
	for range settingProposalsPerTurn {
		d := propose(c, "notifications.sound", "true", "")
		a := nextApproval(t, requests)
		c.Approve(a.ID, false, false, false)
		<-d
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.ProposeSetting(ctx, "notifications.sound", json.RawMessage(`true`), "")
	if refusalCode(err) != CodeSettingLimit {
		t.Fatalf("fourth proposal: %v", err)
	}
}

func TestTheModelsReasonIsQuotedTidyAndBounded(t *testing.T) {
	c, requests, _ := settingController(t)
	d := propose(c, "ui.show_receipt", jsonBool(!userFile().UI.ShowReceipt), "line one\n\nline   two "+strings.Repeat("x", 400))
	a := nextApproval(t, requests)
	if strings.Contains(a.Reason, "\n\n") || strings.Contains(a.Reason, strings.Repeat("x", 200)) {
		t.Fatalf("the reason was not tidied or bounded: %q", a.Reason)
	}
	c.Approve(a.ID, false, false, false)
	<-d
}

func TestSettingsListFindsByWordsAndNeverShowsASecret(t *testing.T) {
	c, _, _ := settingController(t)
	rows := c.SettingsList("peers dispatch")
	if len(rows) != 1 || rows[0].ID != "peers.dispatch" || rows[0].Class != "elevated" {
		t.Fatalf("rows = %+v", rows)
	}
	for _, r := range c.SettingsList("serve token") {
		if r.ID == "serve.token" && r.Value != nil {
			t.Fatalf("a secret's value was listed: %+v", r)
		}
	}
}

func TestOpeningASettingEmitsANoticeAndChangesNothing(t *testing.T) {
	c, _, notices := settingController(t)
	before := userFile().Permissions.Mode
	if err := c.OpenSetting("permissions.mode"); err != nil {
		t.Fatal(err)
	}
	n := <-notices
	if n.Code != event.NoticeCodeSettingOpen || n.Detail != "permissions.mode" || userFile().Permissions.Mode != before {
		t.Fatalf("notice = %+v", n)
	}
	if refusalCode(c.OpenSetting("no.such")) != "setting.unknown" {
		t.Fatal("an undeclared id was opened")
	}
}

func TestADisabledDispatchNamesTheSwitchThatWouldLiftIt(t *testing.T) {
	c, requests, _ := settingController(t)
	_, err := peerDirectory{c}.DispatchToProject(context.Background(), "other", "do it")
	r, ok := asRefusal(err)
	if !ok || r.Code != "dispatch.disabled" || r.Remedy == nil || r.Remedy.Setting != "peers.enabled" || r.Remedy.Op != tool.RemedyPropose {
		t.Fatalf("err = %#v, want the refusal to name peers.enabled", err)
	}
	done := propose(c, r.Remedy.Setting, string(r.Remedy.Value), "hand a task to the other project")
	c.Approve(nextApproval(t, requests).ID, true, false, false)
	if got := <-done; got.err != nil || !userFile().Peers.Enabled {
		t.Fatalf("applying the remedy: %+v err=%v enabled=%v", got.receipt, got.err, userFile().Peers.Enabled)
	}
	c.settings.proposals.Store(0)
	_, err = peerDirectory{c}.DispatchToProject(context.Background(), "other", "do it")
	if r, ok = asRefusal(err); !ok || r.Remedy == nil || r.Remedy.Setting != "peers.dispatch" {
		t.Fatalf("with messaging on, err = %#v, want the dispatch switch next", err)
	}
}
