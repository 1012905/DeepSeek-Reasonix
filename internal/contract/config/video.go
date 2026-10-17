package config

import (
	"strings"

	"reasonix/internal/contract/provider"
)

// The protocols a provider may declare; the wire code that speaks them lives
// with the providers.
const (
	VideoProtocolInlineDataURL = provider.VideoProtocolInlineDataURL
	VideoProtocolMoonshotFile  = provider.VideoProtocolMoonshotFile
)

// KnownVideoProtocols lists the protocols a provider may declare.
func KnownVideoProtocols() []string {
	return []string{VideoProtocolInlineDataURL, VideoProtocolMoonshotFile}
}

// EffectiveVideo reports how the selected model takes a video file natively. ok
// is false unless the model is declared in video_models and the provider names a
// protocol this build knows; every other case is sampled into frames.
func EffectiveVideo(e *ProviderEntry) (protocol string, ok bool) {
	if e == nil || !e.HasVideoModel(e.Model) {
		return "", false
	}
	for _, known := range KnownVideoProtocols() {
		if e.VideoProtocol == known {
			return known, true
		}
	}
	return "", false
}

// HasVideoModel reports whether model is declared in video_models.
func (e *ProviderEntry) HasVideoModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, candidate := range e.VideoModels {
		if strings.EqualFold(strings.TrimSpace(candidate), model) {
			return true
		}
	}
	return false
}
