package source

import (
	"strings"
)

// Trailer keys written by gitai.
const (
	TrailerAutoRevert = "Graft-Auto-Revert"
	TrailerResolvedBy = "Graft-Resolved-By"
)

// ParseTrailers returns the trailers of the last paragraph of a commit
// message, following git interpret-trailers: every line of the last
// paragraph must look like "Key: value" (continuation lines start with
// whitespace).
func ParseTrailers(message string) []Trailer {
	msg := strings.TrimRight(strings.ReplaceAll(message, "\r\n", "\n"), "\n ")
	idx := strings.LastIndex(msg, "\n\n")
	if idx < 0 {
		return nil
	}
	para := msg[idx+2:]
	var out []Trailer
	for _, line := range strings.Split(para, "\n") {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(out) > 0 {
			out[len(out)-1].Value += " " + strings.TrimSpace(line)
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil
		}
		out = append(out, Trailer{Key: key, Value: strings.TrimSpace(value)})
	}
	return out
}

// ApplyTrailers sets the trailers of c and the fields derived from them.
// Trailers already set (for example by the server) are kept.
func ApplyTrailers(c *Commit) {
	if len(c.Trailers) == 0 {
		c.Trailers = ParseTrailers(c.Subject + "\n\n" + c.Body)
	}
	for _, t := range c.Trailers {
		switch {
		case strings.EqualFold(t.Key, TrailerAutoRevert):
			c.AutoRevertOf = firstField(t.Value)
		case strings.EqualFold(t.Key, TrailerResolvedBy):
			c.ResolvedBy = t.Value
		}
	}
}

func firstField(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}
