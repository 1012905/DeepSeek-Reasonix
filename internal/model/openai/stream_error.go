package openai

import (
	"bytes"
	"encoding/json"
	"strings"
)

type streamWireError struct {
	Message string     `json:"message"`
	Code    flexString `json:"code"`
	Type    string     `json:"type"`
	Param   string     `json:"param"`
}

// flexString reads a field compatible servers send as either a string or a
// number: OpenAI and DeepSeek send `"code": "invalid_request"`, a gateway in
// front of them sends `"code": 400`. Insisting on one type fails the whole
// message, hiding the reason the request was refused.
type flexString string

func (s *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*s = ""
		return nil
	}
	if b[0] == '"' {
		var text string
		if err := json.Unmarshal(b, &text); err != nil {
			return err
		}
		*s = flexString(text)
		return nil
	}
	*s = flexString(strings.Trim(string(b), `"`))
	return nil
}

func (s flexString) String() string { return string(s) }
