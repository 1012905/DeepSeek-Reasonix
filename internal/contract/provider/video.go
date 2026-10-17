package provider

import "fmt"

// VideoRef names a video file the host read for the model. It carries no bytes:
// a transcript must not hold a hundred megabytes per read, and the file on disk
// is the one copy.
type VideoRef struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	SHA256     string `json:"sha256"`
	MIME       string `json:"mime"`
	Size       int64  `json:"size"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// VideoShownNote is what every request after the model's answer says in place of
// a video it was shown once, so the model neither guesses why it is gone nor
// assumes it can still see it.
const VideoShownNote = "\n[video %s was shown to you when it was read and is not attached again; call read_video to look at it again]"

// VideoNotSentNote is what a message says when its provider cannot take video.
const VideoNotSentNote = "\n[video %s was not sent: this model cannot take video natively]"

// WithoutVideos returns m with its videos replaced by note, one line per video,
// for a provider that has no wire for them. note takes the file name.
func WithoutVideos(m Message, note string) Message {
	if len(m.Videos) == 0 {
		return m
	}
	for _, v := range m.Videos {
		m.Content += fmt.Sprintf(note, v.Name)
	}
	m.Videos = nil
	return m
}

// answeredAfter marks, for each message, whether an assistant message follows it.
func answeredAfter(msgs []Message) []bool {
	out := make([]bool, len(msgs))
	seen := false
	for i := len(msgs) - 1; i >= 0; i-- {
		out[i] = seen
		if msgs[i].Role == RoleAssistant {
			seen = true
		}
	}
	return out
}
