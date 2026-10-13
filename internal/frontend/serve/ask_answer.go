package serve

import (
	"encoding/json"
	"net/http"
	"reasonix/internal/contract/event"
)

// answer responds to an ask_request.
func (s *Server) answer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID      string            `json:"id"`
		Answers []event.AskAnswer `json:"answers"`
		Engage  bool              `json:"engage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		missingField(w, "id")
		return
	}
	if body.Engage {
		if body.Answers != nil {
			badBody(w)
			return
		}
		if !s.ctl().EngageQuestion(body.ID) {
			refuse(w, http.StatusNotFound, codeAskNotFound, errAskNotFound.Error(), nil)
			return
		}
	} else {
		answerAs(s.ctl(), r, body.ID, body.Answers)
	}
	w.WriteHeader(http.StatusNoContent)
}
