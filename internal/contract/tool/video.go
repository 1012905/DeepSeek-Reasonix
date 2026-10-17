package tool

import (
	"context"
	"encoding/json"

	"reasonix/internal/contract/provider"
)

// VideoTool is an optional capability a Tool may implement when its result can
// carry video files for the model to watch natively, beside the text and images
// ImageTool already carries. The files travel as references: the provider reads
// the bytes only when it builds the request that needs them.
type VideoTool interface {
	ExecuteWithVideo(ctx context.Context, args json.RawMessage) (text string, images []string, videos []provider.VideoRef, err error)
}

// VideoLimit is one named ceiling on a file sent natively. The name travels with
// the number so a refusal can say which of them stopped it.
type VideoLimit struct {
	Source string
	Bytes  int64
}

// NativeVideo is what the selected model offers for watching a file natively:
// the protocol a file reaches it on, and every limit that applies to it.
type NativeVideo struct {
	Protocol string
	Model    string
	Limits   []VideoLimit
}

// Tightest is the limit that binds: the smallest of those named. No limits reads
// as no ceiling, never as zero bytes allowed.
func (n NativeVideo) Tightest() VideoLimit {
	var best VideoLimit
	for _, l := range n.Limits {
		if l.Bytes > 0 && (best.Bytes == 0 || l.Bytes < best.Bytes) {
			best = l
		}
	}
	return best
}

// NativeVideoSource answers, at the moment a tool runs, whether the selected
// model takes video natively. It is asked per call because the model can change
// between calls.
type NativeVideoSource func() (NativeVideo, bool)

type nativeVideoContextKey struct{}

// WithNativeVideo stamps the source onto a tool execution context.
func WithNativeVideo(ctx context.Context, src NativeVideoSource) context.Context {
	if src == nil {
		return ctx
	}
	return context.WithValue(ctx, nativeVideoContextKey{}, src)
}

// NativeVideoFrom asks the source carried by ctx. ok is false when there is none
// or when the model does not take video natively.
func NativeVideoFrom(ctx context.Context) (NativeVideo, bool) {
	if ctx == nil {
		return NativeVideo{}, false
	}
	src, _ := ctx.Value(nativeVideoContextKey{}).(NativeVideoSource)
	if src == nil {
		return NativeVideo{}, false
	}
	n, ok := src()
	return n, ok && n.Protocol != ""
}
