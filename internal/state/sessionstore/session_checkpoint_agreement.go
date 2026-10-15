// Keeping the .jsonl checkpoint and the event log describing one transcript:
// which one answers a torn replay, and how a declined in-place append is undone.
package sessionstore

import (
	"encoding/json"
	"log/slog"
	"os"

	"reasonix/internal/contract/provider"
)

// resolveReplayedMessages answers a replay that applied at least one record.
// A torn log is met with the checkpoint beside it when that checkpoint carries
// the same history further: the turns the torn record dropped are usually
// already durable there, and taking the shorter log alone is what let an
// unclean exit lose them. damaged still rides out so the next save heals.
func resolveReplayedMessages(sessionPath string, replay sessionEventReplay) ([]provider.Message, bool, bool, error) {
	if replay.damaged {
		if healed, ok := checkpointExtendingReplay(sessionPath, replay.msgs); ok {
			return healed, false, true, nil
		}
	}
	return replay.msgs, true, replay.damaged, nil
}

// checkpointExtendingReplay reports the checkpoint when it continues the prefix
// the log still replays. One that is shorter is a read model that has not
// caught up; one that contradicts what survived is another writer's file.
func checkpointExtendingReplay(sessionPath string, replayed []provider.Message) ([]provider.Message, bool) {
	msgs, err := loadSessionMessagesFromJSONL(sessionPath)
	if err != nil {
		// The checkpoint's tail is torn. Keep what decodes rather than reading
		// the tear as "a stranger's file": that fold returned the shorter replay
		// as a clean success and let the next save rewrite it from that.
		msgs = loadSessionMessagesFromJSONLKeepingDecodedTail(sessionPath)
	}
	if len(msgs) <= len(replayed) {
		return nil, false
	}
	if !messagesHavePrefix(msgs, replayed) && !messagesHavePrefixWithCompatibleSystem(msgs, replayed) {
		return nil, false
	}
	return msgs, true
}

// loadSessionMessagesFromJSONLKeepingDecodedTail reads the checkpoint as far as
// its records decode, so a torn final line — the half-written in-place append a
// crash leaves — costs only the record it truncated. Whether the result is
// usable is the caller's call; this only refuses to discard the intact lines.
func loadSessionMessagesFromJSONLKeepingDecodedTail(path string) []provider.Message {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var msgs []provider.Message
	dec := json.NewDecoder(f)
	for {
		var m provider.Message
		if err := dec.Decode(&m); err != nil {
			break
		}
		msgs = append(msgs, m)
	}
	return msgs
}

// settleDisplayReadModel republishes the read model whole when the in-place
// append declined it. Leaving it is what let the checkpoint keep the transcript
// from before this save — a second lineage beside the log, which a fallback
// reader would take as the session. It reports whether the read model is
// current and the boundary the display index refresh may extend from.
func settleDisplayReadModel(path string, msgs []provider.Message, current bool, appendFrom int) (bool, int) {
	if current {
		return true, appendFrom
	}
	if err := writeSessionMessages(path, msgs); err != nil {
		// The transcript and its revision are already durable; a read model
		// that could not be republished stays for background repair.
		slog.Warn("session: keeping save after display read-model rewrite failure", "path", path, "err", err)
		return false, appendFrom
	}
	// Rebuilt whole, so there is no previous index left to extend.
	return true, -1
}
