package control

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"reasonix/internal/base/i18n"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/tool"
)

// paidSearchApprover asks the user before web_search uses a backend the vendor
// charges for. Only a person agrees to be charged, so no approval mode answers
// it; a session grant covers one backend, and "always allow" records the backend
// in the user's own [tools] web_search_paid_allowed.
type paidSearchApprover struct{ c *Controller }

func (p paidSearchApprover) ApprovePaidSearch(ctx context.Context, req tool.PaidSearchRequest) (bool, error) {
	if p.c.approval.preApprovedForDecision(PaidSearchApprovalTool, req.Backend, nil, true) {
		return true, nil
	}
	args, _ := json.Marshal(map[string]string{"backend": req.Backend, "query": req.Query})
	reply, err := p.c.requestApprovalDecision(ctx, approvalRequest{tool: PaidSearchApprovalTool, subject: req.Backend, args: args,
		reason: fmt.Sprintf(i18n.M.PaidSearchApprovalReasonFmt, req.Backend, req.Price, req.Query), fresh: true})
	if err != nil || !reply.allow {
		return false, err
	}
	if reply.session || reply.persist {
		p.c.approval.grantSession(PaidSearchApprovalTool, req.Backend)
	}
	if reply.persist {
		p.c.emitRememberResult(rememberPaidSearchBackend(req.Backend))
	}
	return true, nil
}

// rememberPaidSearchBackend records the user's agreement to be charged by one
// backend. A backend already recorded is not written again.
func rememberPaidSearchBackend(backend string) RememberResult {
	backend = strings.TrimSpace(backend)
	rule := "tools.web_search_paid_allowed += " + backend
	if backend == "" {
		return RememberResult{Rule: rule, Err: fmt.Errorf("no backend to allow")}
	}
	unlock := config.LockUserConfigEdits()
	defer unlock()
	path := config.UserConfigPath()
	cfg := config.LoadForEdit(path)
	if slices.Contains(cfg.Tools.WebSearchPaidAllowed, backend) {
		return RememberResult{Rule: rule, Path: path, CoveredBy: backend}
	}
	cfg.Tools.WebSearchPaidAllowed = append(slices.Clone(cfg.Tools.WebSearchPaidAllowed), backend)
	if err := cfg.SaveTo(path); err != nil {
		return RememberResult{Rule: rule, Path: path, Err: err}
	}
	return RememberResult{Rule: rule, Path: path, Saved: true}
}
