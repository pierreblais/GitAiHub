package gitai

import (
	"bytes"
	"context"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/feder-cr/git-ai/proto/go/gitalypb"
	"github.com/feder-cr/git-ai/sdk"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// emptyTreeSHA1 is the object ID of the empty tree, the parent side of a
// root commit's diff.
const emptyTreeSHA1 = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

var _ source.Gateway = (*Client)(nil)

func (c *Client) DefaultBranch(ctx context.Context, rel string) (string, error) {
	resp, err := c.refs.FindDefaultBranchName(ctx, &gitalypb.FindDefaultBranchNameRequest{Repository: c.repo(rel)})
	if err != nil {
		return "", translate(err)
	}
	return strings.TrimPrefix(string(resp.GetName()), "refs/heads/"), nil
}

func (c *Client) ListBranches(ctx context.Context, rel string) ([]source.Ref, error) {
	stream, err := c.refs.FindLocalBranches(ctx, &gitalypb.FindLocalBranchesRequest{Repository: c.repo(rel), SortBy: gitalypb.FindLocalBranchesRequest_UPDATED_DESC})
	if err != nil {
		return nil, translate(err)
	}
	var out []source.Ref
	err = recvAll(stream.Recv, func(m *gitalypb.FindLocalBranchesResponse) {
		for _, b := range m.GetLocalBranches() {
			out = append(out, source.Ref{
				Name:   strings.TrimPrefix(string(b.GetName()), "refs/heads/"),
				Kind:   source.RefBranch,
				Target: toCommit(b.GetTargetCommit()),
			})
		}
	})
	return out, err
}

func (c *Client) ListTags(ctx context.Context, rel string) ([]source.Ref, error) {
	stream, err := c.refs.FindAllTags(ctx, &gitalypb.FindAllTagsRequest{Repository: c.repo(rel)})
	if err != nil {
		return nil, translate(err)
	}
	var out []source.Ref
	err = recvAll(stream.Recv, func(m *gitalypb.FindAllTagsResponse) {
		for _, t := range m.GetTags() {
			out = append(out, source.Ref{
				Name:   strings.TrimPrefix(string(t.GetName()), "refs/tags/"),
				Kind:   source.RefTag,
				Target: toCommit(t.GetTargetCommit()),
			})
		}
	})
	return out, err
}

func (c *Client) ResolveRevision(ctx context.Context, rel, rev string) (string, error) {
	id, err := c.sdk.Repo(rel).ResolveRevision(ctx, rev)
	return id, translate(err)
}

func (c *Client) ListTree(ctx context.Context, rel, rev, dir string) ([]source.TreeEntry, error) {
	entries, err := c.sdk.Repo(rel).ListTree(ctx, rev, dir, false)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]source.TreeEntry, 0, len(entries))
	idx := map[string]int{}
	for _, e := range entries {
		full := e.Path
		if dir != "" && !strings.HasPrefix(full, dir+"/") {
			full = path.Join(dir, e.Path)
		}
		t := source.EntryBlob
		switch e.Type {
		case sdk.ObjectTree:
			t = source.EntryTree
		case sdk.ObjectCommit:
			t = source.EntrySubmodule
		}
		idx[full] = len(out)
		out = append(out, source.TreeEntry{Name: path.Base(full), Path: full, Type: t, Mode: e.Mode, OID: e.OID, Size: e.Size})
	}
	// Last commit per entry, best effort.
	p := dir
	if p != "" {
		p += "/"
	}
	stream, err := c.commits.ListLastCommitsForTree(ctx, &gitalypb.ListLastCommitsForTreeRequest{
		Repository: c.repo(rel), Revision: rev, Path: []byte(p), Limit: int32(len(out) + 1),
	})
	if err == nil {
		_ = recvAll(stream.Recv, func(m *gitalypb.ListLastCommitsForTreeResponse) {
			for _, ct := range m.GetCommits() {
				if i, ok := idx[strings.TrimSuffix(string(ct.GetPathBytes()), "/")]; ok {
					out[i].LastCommit = toCommit(ct.GetCommit())
				}
			}
		})
	}
	return out, nil
}

func (c *Client) ReadBlob(ctx context.Context, rel, rev, file string, maxBytes int64) (*source.Blob, error) {
	stream, err := c.commits.TreeEntry(ctx, &gitalypb.TreeEntryRequest{
		Repository: c.repo(rel), Revision: []byte(rev), Path: []byte(file), Limit: maxBytes,
	})
	if err != nil {
		return nil, translate(err)
	}
	b := &source.Blob{Path: file}
	first := true
	var buf bytes.Buffer
	var notFile bool
	err = recvAll(stream.Recv, func(m *gitalypb.TreeEntryResponse) {
		if first {
			first = false
			if m.GetType() != gitalypb.TreeEntryResponse_BLOB {
				notFile = true
			}
			b.OID = m.GetOid()
			b.Size = m.GetSize()
		}
		buf.Write(m.GetData())
	})
	if err != nil {
		return nil, err
	}
	if first {
		return nil, translate(errNotFound("file " + file))
	}
	if notFile {
		return nil, translate(errInvalid(file + " is not a file"))
	}
	b.Content = buf.Bytes()
	b.Truncated = b.Size > int64(len(b.Content))
	b.Binary = isBinary(b.Content)
	if b.Binary {
		b.Content = nil
	}
	return b, nil
}

func isBinary(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	if bytes.IndexByte(data[:n], 0) >= 0 {
		return true
	}
	return !utf8.Valid(data[:n]) && !utf8.Valid(data[:max(0, n-3)])
}

func (c *Client) ListCommits(ctx context.Context, rel, rev, p string, offset, limit int) ([]*source.Commit, error) {
	req := &gitalypb.FindCommitsRequest{Repository: c.repo(rel), Revision: []byte(rev), Offset: int32(offset), Limit: int32(limit), Trailers: true}
	if p != "" {
		req.Paths = [][]byte{[]byte(p)}
	}
	stream, err := c.commits.FindCommits(ctx, req)
	if err != nil {
		return nil, translate(err)
	}
	var out []*source.Commit
	err = recvAll(stream.Recv, func(m *gitalypb.FindCommitsResponse) {
		for _, gc := range m.GetCommits() {
			out = append(out, toCommit(gc))
		}
	})
	return out, err
}

func (c *Client) CountCommits(ctx context.Context, rel, rev, p string) (int, error) {
	req := &gitalypb.CountCommitsRequest{Repository: c.repo(rel), Revisions: [][]byte{[]byte(rev)}}
	if p != "" {
		req.Path = []byte(p)
	}
	resp, err := c.commits.CountCommits(ctx, req)
	if err != nil {
		return 0, translate(err)
	}
	return int(resp.GetCount()), nil
}

func (c *Client) GetCommits(ctx context.Context, rel string, ids []string) (map[string]*source.Commit, error) {
	out := map[string]*source.Commit{}
	if len(ids) == 0 {
		return out, nil
	}
	stream, err := c.commits.ListCommitsByOid(ctx, &gitalypb.ListCommitsByOidRequest{Repository: c.repo(rel), Oid: ids})
	if err != nil {
		return nil, translate(err)
	}
	err = recvAll(stream.Recv, func(m *gitalypb.ListCommitsByOidResponse) {
		for _, gc := range m.GetCommits() {
			cm := toCommit(gc)
			out[cm.ID] = cm
			// Requests by short ID also resolve.
			for _, id := range ids {
				if id != cm.ID && strings.HasPrefix(cm.ID, id) {
					out[id] = cm
				}
			}
		}
	})
	return out, err
}

func (c *Client) CommitDiff(ctx context.Context, rel string, cm *source.Commit) ([]source.FileDiff, error) {
	left := emptyTreeSHA1
	if len(cm.ParentIDs) > 0 {
		left = cm.ParentIDs[0]
	}
	stream, err := c.diffs.CommitDiff(ctx, &gitalypb.CommitDiffRequest{
		Repository: c.repo(rel), LeftCommitId: left, RightCommitId: cm.ID,
		EnforceLimits: true, MaxFiles: 300, MaxLines: 20000, MaxBytes: 5 << 20,
		SafeMaxFiles: 300, SafeMaxLines: 20000, SafeMaxBytes: 5 << 20, MaxPatchBytes: 512 << 10,
	})
	if err != nil {
		return nil, translate(err)
	}
	var out []source.FileDiff
	var cur *source.FileDiff
	var patch strings.Builder
	err = recvAll(stream.Recv, func(m *gitalypb.CommitDiffResponse) {
		if m.GetOverflowMarker() {
			return
		}
		if cur == nil {
			cur = &source.FileDiff{
				OldPath: string(m.GetFromPath()), NewPath: string(m.GetToPath()),
				Binary: m.GetBinary(), TooLarge: m.GetTooLarge() || m.GetCollapsed(),
				Additions: int(m.GetLinesAdded()), Deletions: int(m.GetLinesRemoved()),
			}
			cur.Status = fileStatus(m.GetFromId(), m.GetToId(), cur.OldPath, cur.NewPath)
			patch.Reset()
		}
		patch.Write(m.GetRawPatchData())
		if m.GetEndOfPatch() {
			cur.Patch = patch.String()
			if cur.Additions == 0 && cur.Deletions == 0 {
				cur.Additions, cur.Deletions = countPatch(cur.Patch)
			}
			out = append(out, *cur)
			cur = nil
		}
	})
	return out, err
}

func fileStatus(fromID, toID, fromPath, toPath string) source.FileStatus {
	switch {
	case isZero(fromID):
		return source.FileAdded
	case isZero(toID):
		return source.FileDeleted
	case fromPath != toPath:
		return source.FileRenamed
	}
	return source.FileModified
}

func isZero(oid string) bool { return strings.Trim(oid, "0") == "" }

func countPatch(p string) (add, del int) {
	for _, line := range strings.Split(p, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			add++
		case strings.HasPrefix(line, "-"):
			del++
		}
	}
	return add, del
}

func (c *Client) Blame(ctx context.Context, rel, rev, file string) (*source.Blame, error) {
	res, err := c.sdk.Repo(rel).Blame(ctx, rev, file)
	if err != nil {
		return nil, translate(err)
	}
	out := &source.Blame{CommitID: res.CommitID, Lines: make([]source.BlameLine, 0, len(res.Lines))}
	for _, l := range res.Lines {
		out.Lines = append(out.Lines, source.BlameLine{LineNumber: l.LineNumber, CommitID: l.CommitID, Content: string(l.Content)})
	}
	return out, nil
}

func (c *Client) SearchCode(ctx context.Context, rel, rev, query string, maxResults int) (*source.SearchResult, error) {
	matches, commitID, truncated, err := c.sdk.Repo(rel).SearchCode(ctx, sdk.SearchOptions{Revision: rev, Query: query, MaxResults: maxResults, ContextLines: 1})
	if err != nil {
		return nil, translate(err)
	}
	out := &source.SearchResult{CommitID: commitID, Truncated: truncated}
	for _, m := range matches {
		out.Matches = append(out.Matches, source.SearchMatch{Path: m.Path, LineNumber: m.Line, Line: m.Text, Before: m.Before, After: m.After})
	}
	return out, nil
}

func toCommit(gc *gitalypb.GitCommit) *source.Commit {
	if gc == nil {
		return nil
	}
	c := &source.Commit{
		ID:        gc.GetId(),
		ParentIDs: gc.GetParentIds(),
		Subject:   string(gc.GetSubject()),
		Body:      string(gc.GetBody()),
		Author:    toSignature(gc.GetAuthor()),
		Committer: toSignature(gc.GetCommitter()),
	}
	// The body repeats the subject as its first paragraph.
	if strings.HasPrefix(c.Body, c.Subject) {
		c.Body = strings.TrimLeft(strings.TrimPrefix(c.Body, c.Subject), "\n")
	}
	for _, t := range gc.GetTrailers() {
		c.Trailers = append(c.Trailers, source.Trailer{Key: string(t.GetKey()), Value: string(t.GetValue())})
	}
	source.ApplyTrailers(c)
	return c
}

func toSignature(a *gitalypb.CommitAuthor) source.Signature {
	if a == nil {
		return source.Signature{}
	}
	s := source.Signature{Name: string(a.GetName()), Email: string(a.GetEmail())}
	if a.GetDate() != nil {
		s.Time = a.GetDate().AsTime()
	}
	return s
}
