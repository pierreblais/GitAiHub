package review

import "testing"

func TestGroupFiles(t *testing.T) {
	r := Review{Files: []File{
		{Path: "README.md", Class: ClassDocs},
		{Path: "main.go", Class: ClassCode},
		{Path: "main_test.go", Class: ClassTests},
		{Path: "util.go", Class: ClassCode},
	}}
	g := r.GroupFiles()
	if len(g) != 3 || g[0].Class != ClassCode || len(g[0].Files) != 2 || g[1].Class != ClassTests || g[2].Class != ClassDocs {
		t.Fatalf("unexpected groups: %+v", g)
	}
}

func TestCurrentVerdicts(t *testing.T) {
	r := Review{Verdicts: []Verdict{
		{Actor: "a", Kind: VerdictRequestChanges, Current: false},
		{Actor: "b", Kind: VerdictApprove, Current: true},
		{Actor: "a", Kind: VerdictApprove, Current: true},
		{Actor: "b", Kind: VerdictRequestChanges, Current: true},
	}}
	v := r.CurrentVerdicts()
	if len(v) != 2 || v[0].Actor != "b" || v[0].Kind != VerdictRequestChanges || v[1].Actor != "a" || v[1].Kind != VerdictApprove {
		t.Fatalf("unexpected: %+v", v)
	}
}
