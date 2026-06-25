package lark

import "strings"

const (
	newCommandPrefix = "/new"
)

// FreshSessionCommand is the normalized fresh-start directive extracted from a
// Lark inbound message.
type FreshSessionCommand struct {
}

// parseFreshSessionCommand accepts only a message whose ENTIRE trimmed body is
// exactly /new. Anything with extra body text, extra lines, or a longer token
// is treated as a normal chat message.
func parseFreshSessionCommand(body string) (*FreshSessionCommand, bool) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil, false
	}
	if trimmed != newCommandPrefix {
		return nil, false
	}
	return &FreshSessionCommand{}, true
}
