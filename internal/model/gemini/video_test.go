package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/contract/provider"
)

func grantedVideo(t *testing.T, content []byte) provider.VideoRef {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	ref := provider.VideoRef{Path: path, Name: "clip.mp4", SHA256: hex.EncodeToString(sum[:]), MIME: "video/mp4", Size: int64(len(content))}
	provider.GrantVideo(ref)
	return ref
}

func videoRequest(ref provider.VideoRef) provider.Request {
	return provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "what happens?", Videos: []provider.VideoRef{ref}}}}
}

const completed = `{"event_type":"interaction.completed","interaction":{"status":"completed"}}`

// https://ai.google.dev/gemini-api/docs/video-understanding (read 2026-10-17):
// inline video is base64 "data" with a "mime_type". Not run against the real endpoint.
func TestInlineVideoRidesAsBase64DataWithItsMIMEType(t *testing.T) {
	content := []byte("not really an mp4 but the adapter does not decode it")
	ref := grantedVideo(t, content)
	var seen []byte
	srv := serve(t, sse(completed), &seen, nil, nil)
	collect(t, newTestClient(t, srv, map[string]any{"video_protocol": provider.VideoProtocolGeminiInline}), videoRequest(ref))
	want := `"input":[{"type":"user_input","content":[{"type":"text","text":"what happens?"},{"type":"video","data":"` +
		base64.StdEncoding.EncodeToString(content) + `","mime_type":"video/mp4"}]}]`
	if !strings.Contains(string(seen), want) {
		t.Fatalf("request = %s\nwant it to contain %s", seen, want)
	}
}

func TestAVideoThatWasNeverGrantedIsNamedAsUnavailableAndNothingIsSent(t *testing.T) {
	ref := provider.VideoRef{Path: "/etc/hostname", Name: "x.mp4", SHA256: strings.Repeat("a", 64), MIME: "video/mp4", Size: 10}
	var seen []byte
	srv := serve(t, sse(completed), &seen, nil, nil)
	collect(t, newTestClient(t, srv, map[string]any{"video_protocol": provider.VideoProtocolGeminiInline}), videoRequest(ref))
	if !strings.Contains(string(seen), provider.VideoUnavailable) || strings.Contains(string(seen), `"type":"video"`) {
		t.Fatalf("request = %s", seen)
	}
}

func TestAVideoOverTheInlineLimitIsNotSent(t *testing.T) {
	_, limit := provider.VideoProtocolLimit(provider.VideoProtocolGeminiInline)
	ref := provider.VideoRef{Path: "/x", Name: "big.mp4", SHA256: strings.Repeat("a", 64), MIME: "video/mp4", Size: limit + 1}
	var seen []byte
	srv := serve(t, sse(completed), &seen, nil, nil)
	collect(t, newTestClient(t, srv, map[string]any{"video_protocol": provider.VideoProtocolGeminiInline}), videoRequest(ref))
	if !strings.Contains(string(seen), provider.VideoTooLarge) {
		t.Fatalf("request = %s", seen)
	}
}

// The inline limit is the narrower of Google's two figures, held on the encoded
// request: 20,000,000 bytes of request is 15,000,000 raw.
func TestTheInlineLimitIsThreeQuartersOfTheRequestBound(t *testing.T) {
	if source, limit := provider.VideoProtocolLimit(provider.VideoProtocolGeminiInline); source != "gemini_inline" || limit != 15_000_000 {
		t.Fatalf("limit = %s %d", source, limit)
	}
	if _, limit := provider.VideoProtocolLimit(provider.VideoProtocolGeminiFiles); limit != 2<<30 {
		t.Fatalf("files limit = %d", limit)
	}
}

func TestAnEditedFileIsNotSentWhateverItsReferenceSays(t *testing.T) {
	ref := grantedVideo(t, []byte("original bytes"))
	beforeVideoOpen = func(path string) { _ = os.WriteFile(path, []byte("swapped bytes!"), 0o644) }
	defer func() { beforeVideoOpen = func(string) {} }()
	var seen []byte
	srv := serve(t, sse(completed), &seen, nil, nil)
	collect(t, newTestClient(t, srv, map[string]any{"video_protocol": provider.VideoProtocolGeminiInline}), videoRequest(ref))
	if !strings.Contains(string(seen), provider.VideoFileChanged) || strings.Contains(string(seen), `"type":"video"`) {
		t.Fatalf("request = %s", seen)
	}
}

// filesAPI is a fake of the Files API as the files and video pages describe it.
type filesAPI struct {
	mu        sync.Mutex
	srv       *httptest.Server
	startHdr  http.Header
	startBody string
	putHdr    http.Header
	uploaded  []byte
	polls     int
	states    []string // states the polls answer with, in order
	deleted   []string
	request   []byte
	wrongKey  bool
}

func newFilesAPI(t *testing.T, states ...string) *filesAPI {
	t.Helper()
	f := &filesAPI{states: states}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload/v1beta/files", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		b, _ := io.ReadAll(r.Body)
		f.startHdr, f.startBody = r.Header.Clone(), string(b)
		w.Header().Set("X-Goog-Upload-URL", f.srv.URL+"/upload-target")
	})
	mux.HandleFunc("POST /upload-target", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.putHdr = r.Header.Clone()
		f.uploaded, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"file":{"name":"files/abc123","uri":"`+f.srv.URL+`/v1beta/files/abc123","state":"PROCESSING"}}`)
	})
	mux.HandleFunc("GET /v1beta/files/abc123", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		state := f.states[min(f.polls, len(f.states)-1)]
		f.polls++
		_, _ = io.WriteString(w, `{"name":"files/abc123","uri":"`+f.srv.URL+`/v1beta/files/abc123","state":"`+state+`"}`)
	})
	mux.HandleFunc("DELETE /v1beta/files/abc123", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.deleted = append(f.deleted, r.URL.Path)
		_, _ = io.WriteString(w, `{}`)
	})
	mux.HandleFunc("POST /v1beta/interactions", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.request, _ = io.ReadAll(r.Body)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sse(completed))
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *filesAPI) run(t *testing.T, ref provider.VideoRef) {
	t.Helper()
	p, err := New(provider.Config{Name: "g", BaseURL: f.srv.URL + "/v1beta", Model: "gemini-3.8-flash", APIKey: "k-test",
		Extra: map[string]any{"video_protocol": provider.VideoProtocolGeminiFiles}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ch, err := p.Stream(ctx, videoRequest(ref))
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	// The delete runs after the stream ends; give it the moment it needs.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		n := len(f.deleted)
		f.mu.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestALargeVideoGoesThroughTheResumableFilesProtocolAndIsDeletedAfterwards(t *testing.T) {
	filePollEvery = time.Millisecond
	defer func() { filePollEvery = 2 * time.Second }()
	content := []byte(strings.Repeat("video-bytes-", 1000))
	ref := grantedVideo(t, content)
	f := newFilesAPI(t, "PROCESSING", "ACTIVE")
	f.run(t, ref)

	if f.startHdr.Get("X-Goog-Upload-Protocol") != "resumable" || f.startHdr.Get("X-Goog-Upload-Command") != "start" ||
		f.startHdr.Get("X-Goog-Upload-Header-Content-Length") != "12000" || f.startHdr.Get("X-Goog-Upload-Header-Content-Type") != "video/mp4" ||
		f.startHdr.Get("x-goog-api-key") != "k-test" {
		t.Fatalf("start headers = %v", f.startHdr)
	}
	if f.startBody != `{"file":{"display_name":"clip.mp4"}}` {
		t.Fatalf("start body = %s", f.startBody)
	}
	if f.putHdr.Get("X-Goog-Upload-Offset") != "0" || f.putHdr.Get("X-Goog-Upload-Command") != "upload, finalize" || string(f.uploaded) != string(content) {
		t.Fatalf("upload headers = %v, %d bytes", f.putHdr, len(f.uploaded))
	}
	if f.polls != 2 {
		t.Fatalf("polled %d times, want it to wait for ACTIVE", f.polls)
	}
	want := `{"type":"video","uri":"` + f.srv.URL + `/v1beta/files/abc123","mime_type":"video/mp4"}`
	if !strings.Contains(string(f.request), want) {
		t.Fatalf("request = %s\nwant it to contain %s", f.request, want)
	}
	if len(f.deleted) != 1 || f.deleted[0] != "/v1beta/files/abc123" {
		t.Fatalf("deleted = %v", f.deleted)
	}
}

func TestAFileGoogleFailsToProcessIsNotSentAndIsStillDeleted(t *testing.T) {
	filePollEvery = time.Millisecond
	defer func() { filePollEvery = 2 * time.Second }()
	ref := grantedVideo(t, []byte("bytes"))
	f := newFilesAPI(t, "FAILED")
	f.run(t, ref)
	if !strings.Contains(string(f.request), provider.VideoUploadFailed) || strings.Contains(string(f.request), `"type":"video"`) {
		t.Fatalf("request = %s", f.request)
	}
	if len(f.deleted) != 1 {
		t.Fatalf("a file that failed to process was left behind: %v", f.deleted)
	}
}

func TestAStateTheAdapterDoesNotKnowIsAFailureNotAWait(t *testing.T) {
	filePollEvery = time.Millisecond
	defer func() { filePollEvery = 2 * time.Second }()
	ref := grantedVideo(t, []byte("bytes"))
	f := newFilesAPI(t, "SOMETHING_NEW")
	f.run(t, ref)
	if f.polls != 1 || !strings.Contains(string(f.request), provider.VideoUploadFailed) {
		t.Fatalf("polls = %d, request = %s", f.polls, f.request)
	}
}

func TestVideoIsNotSentWhereNoProtocolIsDeclared(t *testing.T) {
	ref := grantedVideo(t, []byte("bytes"))
	var seen []byte
	srv := serve(t, sse(completed), &seen, nil, nil)
	collect(t, newTestClient(t, srv, nil), videoRequest(ref))
	if !strings.Contains(string(seen), "was not sent") || strings.Contains(string(seen), `"type":"video"`) {
		t.Fatalf("request = %s", seen)
	}
}

func TestTheFileURLsAreDerivedFromTheBase(t *testing.T) {
	w := newVideoWire(provider.VideoProtocolGeminiFiles, "https://generativelanguage.googleapis.com/v1beta")
	if w == nil || w.uploadURL != "https://generativelanguage.googleapis.com/upload/v1beta/files" || w.root != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("wire = %+v", w)
	}
}
