package event

// Attachment is an image a message references that the host stored, or that the
// session was handed outside the workspace (a dropped folder or file).
//
// For a workspace image Path is workspace-relative and slash-separated, and Ref
// is empty. For an external image Path is the absolute, native-spelled location
// the image lives at — what a paste reads back — and Ref is the opaque
// external-folder token the line was submitted with, so the surface can hide
// that token from the visible body now that it previews the image by Path.
type Attachment struct {
	Path string
	Ref  string
}
