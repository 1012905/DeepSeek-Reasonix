package builtin

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/contract/provider"
	"reasonix/internal/contract/tool"
)

func box(kind string, payload ...[]byte) []byte {
	var body []byte
	for _, p := range payload {
		body = append(body, p...)
	}
	out := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(out, uint32(8+len(body)))
	copy(out[4:], kind)
	return append(out, body...)
}

// mp4Of is a minimal MP4: a file type box, a movie box whose header says the
// duration, and a media data box padded to size.
func mp4Of(seconds, size int) []byte {
	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:], 1000)
	binary.BigEndian.PutUint32(mvhd[16:], uint32(seconds*1000))
	out := append(box("ftyp", []byte("isom"), make([]byte, 4), []byte("isom")), box("moov", box("mvhd", mvhd))...)
	if pad := size - len(out) - 8; pad > 0 {
		out = append(out, box("mdat", make([]byte, pad))...)
	}
	return out
}

func nativeTool(t *testing.T, content []byte, name string) (readVideo, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
		t.Fatal(err)
	}
	media := &fakeMedia{probe: goodProbe}
	return readVideo{
		workDir: dir,
		run:     media.run,
		// No ffmpeg: a native read must not need it.
		look: func(string) (string, error) { return "", os.ErrNotExist },
	}, dir
}

func nativeCtx(limits ...tool.VideoLimit) context.Context {
	return tool.WithNativeVideo(context.Background(), func() (tool.NativeVideo, bool) {
		return tool.NativeVideo{Protocol: "inline_data_url", Model: "vid-model", Limits: limits}, true
	})
}

func callNative(t *testing.T, v readVideo, ctx context.Context, args map[string]any) (string, []string, []string, error) {
	t.Helper()
	raw, _ := json.Marshal(args)
	text, images, videos, err := v.ExecuteWithVideo(ctx, raw)
	var names []string
	for _, r := range videos {
		names = append(names, r.Name)
	}
	_ = videos
	return text, images, names, err
}

func TestNativeReadHandsOverAReferenceWithoutFFmpeg(t *testing.T) {
	content := mp4Of(3, 4096)
	v, _ := nativeTool(t, content, "clip.mp4")
	raw, _ := json.Marshal(map[string]any{"path": "clip.mp4", "mode": "native"})
	text, images, videos, err := v.ExecuteWithVideo(nativeCtx(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 0 || len(videos) != 1 {
		t.Fatalf("images=%d videos=%d, want the video alone", len(images), len(videos))
	}
	sum := sha256.Sum256(content)
	ref := videos[0]
	if ref.Name != "clip.mp4" || ref.MIME != "video/mp4" || ref.Size != int64(len(content)) || ref.DurationMs != 3000 ||
		ref.SHA256 != hex.EncodeToString(sum[:]) || !filepath.IsAbs(ref.Path) || filepath.Base(ref.Path) != "clip.mp4" {
		t.Fatalf("ref = %+v", ref)
	}
	if path, ok := provider.GrantedVideoPath(ref); !ok || path != ref.Path {
		t.Errorf("the tool must grant the path it checked: got %q, %v", path, ok)
	}
	for _, want := range []string{"clip.mp4", "3.0s", "vid-model", "next request"} {
		if !strings.Contains(text, want) {
			t.Errorf("the text does not say %q:\n%s", want, text)
		}
	}
}

func TestExplicitNativeRefusesByCodeWhereNothingOffersIt(t *testing.T) {
	v, _ := nativeTool(t, mp4Of(3, 4096), "clip.mp4")
	_, _, _, err := callNative(t, v, context.Background(), map[string]any{"path": "clip.mp4", "mode": "native"})
	if refusalCode(err) != CodeVideoNativeUnsupported {
		t.Fatalf("code = %q, want %q", refusalCode(err), CodeVideoNativeUnsupported)
	}
}

func TestNativeSaysWhichLimitStoppedIt(t *testing.T) {
	v, _ := nativeTool(t, mp4Of(3, 8192), "clip.mp4")
	ctx := nativeCtx(tool.VideoLimit{Source: "inline_data_url", Bytes: 7_500_000}, tool.VideoLimit{Source: "tools.video_native_max_bytes", Bytes: 4096})
	_, _, _, err := callNative(t, v, ctx, map[string]any{"path": "clip.mp4", "mode": "native"})
	if refusalCode(err) != CodeVideoNativeTooLarge || !strings.Contains(err.Error(), "tools.video_native_max_bytes") || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("err = %v (%q), want the too-large code naming the setting and its bytes", err, refusalCode(err))
	}
}

func TestOnlyMP4IsSentNatively(t *testing.T) {
	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, make([]byte, 4096)...)
	v, _ := nativeTool(t, webm, "clip.webm")
	_, _, _, err := callNative(t, v, nativeCtx(), map[string]any{"path": "clip.webm", "mode": "native"})
	if refusalCode(err) != CodeVideoNativeContainer {
		t.Fatalf("code = %q, want %q", refusalCode(err), CodeVideoNativeContainer)
	}
}

// auto sends natively when it can and otherwise samples frames, as before.
func TestAutoFallsBackToFramesWhenNativeIsNotOffered(t *testing.T) {
	v, _ := nativeTool(t, mp4Of(3, 4096), "clip.mp4")
	_, _, _, err := callNative(t, v, context.Background(), map[string]any{"path": "clip.mp4"})
	if refusalCode(err) != CodeVideoFFmpegMissing {
		t.Fatalf("auto without native took %q, want the frame route (which needs ffmpeg)", refusalCode(err))
	}
	_, _, videos, err := callNative(t, v, nativeCtx(), map[string]any{"path": "clip.mp4"})
	if err != nil || len(videos) != 1 {
		t.Fatalf("auto with native offered: videos=%v err=%v", videos, err)
	}
}

func TestAutoSaysWhyItSampledFramesInsteadOfSendingTheFile(t *testing.T) {
	v, dir := nativeTool(t, mp4Of(3, 8192), "clip.mp4")
	media := &fakeMedia{probe: goodProbe}
	v.run = media.run
	v.look = func(name string) (string, error) { return filepath.Join("/opt/bin", name), nil }
	ctx := nativeCtx(tool.VideoLimit{Source: "tools.video_native_max_bytes", Bytes: 4096})
	text, images, videos, err := callNative(t, v, ctx, map[string]any{"path": "clip.mp4"})
	_ = dir
	if err != nil || len(videos) != 0 || len(images) == 0 {
		t.Fatalf("videos=%v images=%d err=%v, want frames", videos, len(images), err)
	}
	if !strings.Contains(text, CodeVideoNativeTooLarge) || !strings.Contains(text, "tools.video_native_max_bytes") {
		t.Fatalf("the model is not told why frames were sampled:\n%s", text)
	}
}

func nativeCtxOn(protocol string) context.Context {
	return tool.WithNativeVideo(context.Background(), func() (tool.NativeVideo, bool) {
		return tool.NativeVideo{Protocol: protocol, Model: "vid-model"}, true
	})
}

func quickTimeOf(seconds, size int) []byte {
	b := mp4Of(seconds, size)
	copy(b[8:12], "qt  ")
	return b
}

// An EBML header whose doctype element (0x4282) says webm; Matroska's own says matroska.
func ebmlOf(doctype string, size int) []byte {
	out := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x80 | byte(4+len(doctype)), 0x42, 0x82, 0x80 | byte(len(doctype))}
	out = append(out, doctype...)
	return append(out, make([]byte, size)...)
}

// https://ai.google.dev/gemini-api/docs/video-understanding (read 2026-10-17)
// lists video/mp4, video/mpeg, video/mov, video/avi, video/x-flv, video/mpg,
// video/webm, video/wmv and video/3gpp; the container sniff can tell three apart.
func TestGeminiProtocolsSendMP4WebMAndMOVWithTheirOwnMIMEType(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
		mime    string
	}{
		{"clip.mp4", mp4Of(3, 4096), "video/mp4"},
		{"clip.mov", quickTimeOf(3, 4096), "video/mov"},
		{"clip.webm", ebmlOf("webm", 4096), "video/webm"},
	} {
		for _, protocol := range []string{provider.VideoProtocolGeminiInline, provider.VideoProtocolGeminiFiles} {
			v, _ := nativeTool(t, tc.content, tc.name)
			raw, _ := json.Marshal(map[string]any{"path": tc.name, "mode": "native"})
			_, _, videos, err := v.ExecuteWithVideo(nativeCtxOn(protocol), raw)
			if err != nil || len(videos) != 1 || videos[0].MIME != tc.mime {
				t.Errorf("%s on %s: videos=%+v err=%v, want %s", tc.name, protocol, videos, err, tc.mime)
			}
		}
	}
}

func TestTheOtherProtocolsStillSendOnlyMP4(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
	}{{"clip.mov", quickTimeOf(3, 4096)}, {"clip.webm", ebmlOf("webm", 4096)}} {
		v, _ := nativeTool(t, tc.content, tc.name)
		raw, _ := json.Marshal(map[string]any{"path": tc.name, "mode": "native"})
		_, _, _, err := v.ExecuteWithVideo(nativeCtxOn(provider.VideoProtocolInlineDataURL), raw)
		if refusalCode(err) != CodeVideoNativeContainer {
			t.Errorf("%s: code = %q, want %q", tc.name, refusalCode(err), CodeVideoNativeContainer)
		}
	}
}

func TestMatroskaIsNotWebM(t *testing.T) {
	v, _ := nativeTool(t, ebmlOf("matroska", 4096), "clip.mkv")
	raw, _ := json.Marshal(map[string]any{"path": "clip.mkv", "mode": "native"})
	_, _, _, err := v.ExecuteWithVideo(nativeCtxOn(provider.VideoProtocolGeminiFiles), raw)
	if refusalCode(err) != CodeVideoNativeContainer {
		t.Fatalf("code = %q, want %q", refusalCode(err), CodeVideoNativeContainer)
	}
}
