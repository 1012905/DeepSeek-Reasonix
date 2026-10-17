package config

import "strings"

// How a video file reaches a model that reads video natively. A protocol is a
// property of the endpoint's wire, so it is declared beside the models and never
// read off a host name or a model name.
const (
	// VideoProtocolInlineDataURL sends the file in the request itself, as a
	// video_url content part holding a data URL.
	VideoProtocolInlineDataURL = "inline_data_url"
	// VideoProtocolMoonshotFile uploads the file once and refers to it by id in
	// a video_url content part.
	VideoProtocolMoonshotFile = "moonshot_file"
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

// VideoDeclared reports whether anyone answered the native-video question for
// this entry, as opposed to nobody having said: an explicit empty list is an
// answer, a missing one is not.
func VideoDeclared(e *ProviderEntry) bool {
	return e != nil && e.VideoModels != nil
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
