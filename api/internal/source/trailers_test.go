package source

import "testing"

func TestParseTrailers(t *testing.T) {
	msg := "Fix the parser\n\nLonger body\nwith lines.\n\nGraft-Auto-Revert: abc123 (verify failed)\nGraft-Resolved-By: agent/builder\nSigned-off-by: a\n  continued\n"
	tr := ParseTrailers(msg)
	if len(tr) != 3 {
		t.Fatalf("want 3 trailers, got %v", tr)
	}
	if tr[2].Value != "a continued" {
		t.Fatalf("continuation: %q", tr[2].Value)
	}
	c := &Commit{Subject: "Fix the parser", Body: "Longer body\nwith lines.\n\nGraft-Auto-Revert: abc123 (verify failed)\nGraft-Resolved-By: agent/builder\n"}
	ApplyTrailers(c)
	if c.AutoRevertOf != "abc123" || c.ResolvedBy != "agent/builder" {
		t.Fatalf("derived fields: %+v", c)
	}
}

func TestParseTrailersNone(t *testing.T) {
	for _, msg := range []string{"Just a subject", "Subject\n\nA paragraph that is not: a trailer block because spaces in key", "Subject\n\nbody line\nKey: v"} {
		if tr := ParseTrailers(msg); tr != nil {
			t.Errorf("%q: want no trailers, got %v", msg, tr)
		}
	}
}

func TestCommitAgentFallback(t *testing.T) {
	c := &Commit{Author: Signature{Name: "someone"}}
	if c.Agent() != "someone" || c.AgentVerified() {
		t.Fatal("fallback to author")
	}
	c.Landing = &Landing{Agent: "planner"}
	if c.Agent() != "planner" || !c.AgentVerified() {
		t.Fatal("landing agent wins")
	}
}
