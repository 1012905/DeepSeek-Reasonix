package openai

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"

	"reasonix/internal/contract/provider"
)

// The identities of why a video was not sent, said to the model beside the file's
// name so it does not invent a reason.
const (
	videoFileMissing  = "video.file_missing"
	videoFileChanged  = "video.file_changed"
	videoTooLarge     = "video.native_too_large"
	videoUploadFailed = "video.upload_failed"
	videoReadFailed   = "video.read_failed"
)

// videoWire sends the videos a request carries on the protocol its endpoint
// declares, and remembers what the vendor already holds.
type videoWire struct {
	protocol string
	filesURL string
	mu       sync.Mutex
	ids      map[string]string // file sha256 → the id the vendor gave it
}

func newVideoWire(protocol, baseURL string) *videoWire {
	switch protocol {
	case provider.VideoProtocolInlineDataURL, provider.VideoProtocolMoonshotFile:
	default:
		return nil
	}
	return &videoWire{protocol: protocol, filesURL: strings.TrimRight(baseURL, "/") + "/files", ids: map[string]string{}}
}

// resolveVideos replaces every video a request still carries with what the
// endpoint can read, or with a note naming why not. cached lists the files whose
// reference came from an earlier upload, which a vendor may since have dropped.
func (c *client) resolveVideos(ctx context.Context, msgs []provider.Message) (out []provider.Message, cached []string) {
	out = msgs
	for i, m := range msgs {
		if len(m.Videos) == 0 {
			continue
		}
		if &out[0] == &msgs[0] {
			out = append([]provider.Message(nil), msgs...)
		}
		if c.video == nil {
			out[i] = provider.WithoutVideos(m, provider.VideoNotSentNote)
			continue
		}
		m.VideoParts = append([]provider.VideoPart(nil), m.VideoParts...)
		for _, ref := range m.Videos {
			url, fromCache, code := c.video.resolve(ctx, c.http, c.setUploadHeaders, ref)
			if code != "" {
				m.Content += fmt.Sprintf(provider.VideoFailedNote, provider.SafeVideoName(ref.Name), code)
				continue
			}
			if fromCache {
				cached = append(cached, ref.SHA256)
			}
			m.VideoParts = append(m.VideoParts, provider.VideoPart{Name: ref.Name, URL: url})
		}
		m.Videos = nil
		out[i] = m
	}
	return out, cached
}

// forget drops the references for files a vendor no longer holds.
func (w *videoWire) forget(shas []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, sha := range shas {
		delete(w.ids, sha)
	}
}

// afterVideoOpen is called with the path right after a video file is opened.
// It exists so a test can swap the path there.
var afterVideoOpen = func(string) {}

// beforeVideoOpen is called with the path just before it is opened.
var beforeVideoOpen = func(string) {}

var errVideoChanged = errors.New("the file no longer matches what the tool read")

// resolve turns a reference into what the vendor reads. The file is opened once
// and everything after that reads the opened file, so a path swapped for another
// file between the check and the send cannot change the bytes that leave. A file
// the vendor already holds by its hash needs no read at all: the id stands for
// exactly the bytes that were verified when it was uploaded.
func (w *videoWire) resolve(ctx context.Context, h *http.Client, auth func(*http.Request), ref provider.VideoRef) (url string, fromCache bool, code string) {
	_, limit := provider.VideoProtocolLimit(w.protocol)
	if ref.Size > limit {
		return "", false, videoTooLarge
	}
	if w.protocol == provider.VideoProtocolMoonshotFile {
		w.mu.Lock()
		id, ok := w.ids[ref.SHA256]
		w.mu.Unlock()
		if ok {
			return "ms://" + id, true, ""
		}
	}
	beforeVideoOpen(ref.Path)
	f, err := os.Open(ref.Path)
	if err != nil {
		return "", false, readCode(err)
	}
	defer f.Close()
	afterVideoOpen(ref.Path)
	if w.protocol == provider.VideoProtocolInlineDataURL {
		raw, err := io.ReadAll(io.LimitReader(f, ref.Size+1))
		if err != nil {
			return "", false, videoReadFailed
		}
		if sum := sha256.Sum256(raw); int64(len(raw)) != ref.Size || hex.EncodeToString(sum[:]) != ref.SHA256 {
			return "", false, videoFileChanged
		}
		return "data:" + ref.MIME + ";base64," + base64.StdEncoding.EncodeToString(raw), false, ""
	}
	// The opened file is checked end to end before a byte goes out; the check
	// inside the stream is the second line, against the same file changing in place.
	if code := verifyOpened(f, ref); code != "" {
		return "", false, code
	}
	id, err := w.upload(ctx, h, auth, ref, f)
	switch {
	case errors.Is(err, errVideoChanged):
		return "", false, videoFileChanged
	case err != nil:
		return "", false, videoUploadFailed
	}
	w.mu.Lock()
	w.ids[ref.SHA256] = id
	w.mu.Unlock()
	return "ms://" + id, false, ""
}

// verifyOpened hashes the open file from its start and leaves it rewound.
func verifyOpened(f *os.File, ref provider.VideoRef) string {
	sum := sha256.New()
	n, err := io.Copy(sum, f)
	if err != nil {
		return videoReadFailed
	}
	if n != ref.Size || hex.EncodeToString(sum.Sum(nil)) != ref.SHA256 {
		return videoFileChanged
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return videoReadFailed
	}
	return ""
}

func readCode(err error) string {
	if errors.Is(err, fs.ErrNotExist) {
		return videoFileMissing
	}
	return videoReadFailed
}

// upload streams the opened file as the multipart body. The bytes are hashed as
// they go and the body is closed only if they match what the tool read, so a file
// that changes under the upload is never completed on the vendor's side.
func (w *videoWire) upload(ctx context.Context, h *http.Client, auth func(*http.Request), ref provider.VideoRef, f *os.File) (string, error) {
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	go func() {
		err := form.WriteField("purpose", "video")
		if err == nil {
			var part io.Writer
			if part, err = form.CreateFormFile("file", ref.Name); err == nil {
				sum := sha256.New()
				var n int64
				if n, err = io.Copy(part, io.TeeReader(f, sum)); err == nil && (n != ref.Size || hex.EncodeToString(sum.Sum(nil)) != ref.SHA256) {
					err = errVideoChanged
				}
			}
		}
		if err == nil {
			err = form.Close()
		}
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.filesURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	auth(req)
	resp, err := h.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &out) != nil || out.ID == "" {
		return "", errors.New("no file id in the answer")
	}
	return out.ID, nil
}

// videoContentParts is the user message that carries the videos tool results
// returned, after the whole run of tool results.
func videoContentParts(text string, parts []provider.VideoPart) []chatContentPart {
	out := make([]chatContentPart, 0, len(parts)+1)
	if text != "" {
		out = append(out, chatContentPart{Type: "text", Text: text})
	}
	for _, p := range parts {
		out = append(out, chatContentPart{Type: "video_url", VideoURL: &chatVideoURL{URL: p.URL}})
	}
	return out
}

type chatVideoURL struct {
	URL string `json:"url"`
}

// setUploadHeaders signs a file upload the way a chat request is signed, without
// the streaming headers that belong to one.
func (c *client) setUploadHeaders(req *http.Request) {
	applyAPIKeyHeader(req.Header, c.baseURL, c.apiKey())
	applyCustomHeaders(req.Header, c.headers)
	provider.ApplyClientIdentity(req)
}

// toolMedia is what tool results returned beside their text, held until the run
// of tool messages ends: the API takes only text under role "tool".
type toolMedia struct {
	images []string
	videos []provider.VideoPart
}

func (p *toolMedia) add(m provider.Message, vision bool) {
	if vision {
		p.images = append(p.images, m.Images...)
	}
	p.videos = append(p.videos, m.VideoParts...)
}

// flush appends the held media as user messages and empties the hold.
func (p *toolMedia) flush(msgs []chatMessage, detail string) []chatMessage {
	if len(p.videos) > 0 {
		msgs = append(msgs, chatMessage{Role: "user", Content: videoContentParts("Video returned by the preceding tool call(s):", p.videos)})
	}
	if len(p.images) > 0 {
		msgs = append(msgs, chatMessage{Role: "user", Content: imageContentParts("Images returned by the preceding tool call(s):", p.images, detail)})
	}
	*p = toolMedia{}
	return msgs
}
