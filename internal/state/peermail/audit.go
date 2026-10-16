package peermail

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Decision is what became of a message on this side.
type Decision string

const (
	DecisionHeld     Decision = "held"
	DecisionApproved Decision = "approved"
	DecisionDenied   Decision = "denied"
	DecisionRejected Decision = "rejected"
)

// AuditRecord is one line of the audit trail. It names the parties and says
// what was decided; it never carries the words of the message, only their
// length and a digest, so reading the trail reveals no conversation.
type AuditRecord struct {
	At       time.Time `json:"at"`
	Receiver string    `json:"receiver"`
	Sender   string    `json:"sender,omitempty"`
	ID       string    `json:"id,omitempty"`
	Decision Decision  `json:"decision"`
	Reason   Reason    `json:"reason,omitempty"`
	Bytes    int       `json:"bytes,omitempty"`
	SHA256   string    `json:"sha256,omitempty"`
}

// Digest is the audit's stand-in for a message body.
func Digest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// Audit appends one record to the month's file under the peers directory.
func (d *Dir) Audit(r AuditRecord) error {
	if r.At.IsZero() {
		r.At = time.Now().UTC()
	}
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	dir := filepath.Join(d.root, "audit")
	if err := checkPrivate(dir, true); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, r.At.UTC().Format("2006-01")+".jsonl"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}
