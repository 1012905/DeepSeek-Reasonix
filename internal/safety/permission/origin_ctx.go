package permission

import "context"

type originKey struct{}

// WithOrigin marks ctx with where the text in a call's arguments came from, as the
// host recorded it (for example "web:example.com"). The gate reads it; nothing here
// decides what the label means.
func WithOrigin(ctx context.Context, label string) context.Context {
	if label == "" {
		return ctx
	}
	return context.WithValue(ctx, originKey{}, label)
}

// OriginFrom returns the label WithOrigin set, or "".
func OriginFrom(ctx context.Context) string {
	s, _ := ctx.Value(originKey{}).(string)
	return s
}
