package sessionstore

import "reasonix/internal/contract/provider"

const sessionEventCompositionSchemaVersion = 4

func compositionRemoved(previousSchema int, msgs []provider.Message) bool {
	return previousSchema == sessionEventCompositionSchemaVersion && compositionEventSchema("", msgs, nil) != sessionEventCompositionSchemaVersion
}

func compositionEventSchema(_ string, msgs []provider.Message, images []sessionEventImages) int {
	for _, m := range msgs {
		if m.AgentComposition != nil {
			return sessionEventCompositionSchemaVersion
		}
	}
	return sessionEventRecordSchema(images)
}
