package source

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
)

// MaxBlobBytes bounds the file content returned to the browser.
const MaxBlobBytes = 1 << 20

// Service answers Source queries. It resolves repository names through the
// catalog, reads from gitai through the gateway and attributes commits to
// agents through the landing lookup.
type Service struct {
	Catalog  RepositoryCatalog
	Gateway  Gateway
	Landings LandingLookup
}

// ListRepositories returns every known repository.
func (s *Service) ListRepositories(ctx context.Context) ([]Repository, error) {
	return s.Catalog.ListRepositories(ctx)
}

// GetRepository returns one repository, with its default branch from gitai.
func (s *Service) GetRepository(ctx context.Context, name string) (*Repository, error) {
	repo, err := s.Catalog.GetRepository(ctx, name)
	if err != nil {
		return nil, err
	}
	if repo.DefaultBranch == "" {
		if b, err := s.Gateway.DefaultBranch(ctx, repo.RelativePath); err == nil {
			repo.DefaultBranch = b
		}
	}
	return repo, nil
}

func (s *Service) path(ctx context.Context, name string) (*Repository, error) {
	return s.GetRepository(ctx, name)
}

// revision returns rev, or the default branch when rev is empty.
func (s *Service) revision(repo *Repository, rev string) (string, error) {
	if rev != "" {
		return rev, nil
	}
	if repo.DefaultBranch == "" {
		return "", fmt.Errorf("%w: repository %q has no default branch", ErrNotFound, repo.Name)
	}
	return repo.DefaultBranch, nil
}

// ListRefs returns branches or tags. The default branch comes first.
func (s *Service) ListRefs(ctx context.Context, name string, kind RefKind) ([]Ref, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, err
	}
	var refs []Ref
	if kind == RefTag {
		refs, err = s.Gateway.ListTags(ctx, repo.RelativePath)
	} else {
		refs, err = s.Gateway.ListBranches(ctx, repo.RelativePath)
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if (refs[i].Name == repo.DefaultBranch) != (refs[j].Name == repo.DefaultBranch) {
			return refs[i].Name == repo.DefaultBranch
		}
		return refs[i].Name < refs[j].Name
	})
	var targets []*Commit
	for i := range refs {
		if refs[i].Target != nil {
			targets = append(targets, refs[i].Target)
		}
	}
	s.attribute(ctx, repo, targets)
	return refs, nil
}

// Tree is a directory listing at a revision.
type Tree struct {
	Revision string
	CommitID string
	Path     string
	Entries  []TreeEntry
	// Readme is the README of the directory, if there is one.
	Readme *Blob
}

// GetTree lists a directory, directories first, with the README rendered.
func (s *Service) GetTree(ctx context.Context, name, rev, dir string) (*Tree, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, err
	}
	if rev, err = s.revision(repo, rev); err != nil {
		return nil, err
	}
	dir = cleanPath(dir)
	commitID, err := s.Gateway.ResolveRevision(ctx, repo.RelativePath, rev)
	if err != nil {
		return nil, err
	}
	entries, err := s.Gateway.ListTree(ctx, repo.RelativePath, commitID, dir)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if (entries[i].Type == EntryTree) != (entries[j].Type == EntryTree) {
			return entries[i].Type == EntryTree
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	t := &Tree{Revision: rev, CommitID: commitID, Path: dir, Entries: entries}
	var lasts []*Commit
	for i := range entries {
		if entries[i].LastCommit != nil {
			lasts = append(lasts, entries[i].LastCommit)
		}
	}
	s.attribute(ctx, repo, lasts)
	for _, e := range entries {
		if e.Type == EntryBlob && isReadme(e.Name) {
			if b, err := s.Gateway.ReadBlob(ctx, repo.RelativePath, commitID, e.Path, MaxBlobBytes); err == nil {
				t.Readme = b
			}
			break
		}
	}
	return t, nil
}

// GetBlob returns a file with the last commit that changed it.
func (s *Service) GetBlob(ctx context.Context, name, rev, file string) (*Blob, *Commit, string, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, nil, "", err
	}
	if rev, err = s.revision(repo, rev); err != nil {
		return nil, nil, "", err
	}
	file = cleanPath(file)
	if file == "" {
		return nil, nil, "", fmt.Errorf("%w: path is required", ErrInvalidArgument)
	}
	commitID, err := s.Gateway.ResolveRevision(ctx, repo.RelativePath, rev)
	if err != nil {
		return nil, nil, "", err
	}
	b, err := s.Gateway.ReadBlob(ctx, repo.RelativePath, commitID, file, MaxBlobBytes)
	if err != nil {
		return nil, nil, "", err
	}
	var last *Commit
	if cs, err := s.Gateway.ListCommits(ctx, repo.RelativePath, commitID, file, 0, 1); err == nil && len(cs) > 0 {
		last = cs[0]
		s.attribute(ctx, repo, cs)
	}
	return b, last, commitID, nil
}

// ListCommits returns commits newest first, optionally limited to a path.
func (s *Service) ListCommits(ctx context.Context, name, rev, p string, offset, limit int) ([]*Commit, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, err
	}
	if rev, err = s.revision(repo, rev); err != nil {
		return nil, err
	}
	cs, err := s.Gateway.ListCommits(ctx, repo.RelativePath, rev, cleanPath(p), offset, limit)
	if err != nil {
		return nil, err
	}
	s.attribute(ctx, repo, cs)
	return cs, nil
}

// GetCommit returns one commit with its diff.
func (s *Service) GetCommit(ctx context.Context, name, id string) (*Commit, []FileDiff, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	cs, err := s.Gateway.GetCommits(ctx, repo.RelativePath, []string{id})
	if err != nil {
		return nil, nil, err
	}
	c := cs[id]
	if c == nil {
		return nil, nil, fmt.Errorf("%w: commit %q", ErrNotFound, id)
	}
	s.attribute(ctx, repo, []*Commit{c})
	diff, err := s.Gateway.CommitDiff(ctx, repo.RelativePath, c)
	if err != nil {
		return nil, nil, err
	}
	return c, diff, nil
}

// GetBlame returns the blame of a file with each line's commit attributed.
func (s *Service) GetBlame(ctx context.Context, name, rev, file string) (*Blame, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, err
	}
	if rev, err = s.revision(repo, rev); err != nil {
		return nil, err
	}
	b, err := s.Gateway.Blame(ctx, repo.RelativePath, rev, cleanPath(file))
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	seen := map[string]bool{}
	for _, l := range b.Lines {
		if !seen[l.CommitID] {
			seen[l.CommitID] = true
			ids = append(ids, l.CommitID)
		}
	}
	commits, err := s.Gateway.GetCommits(ctx, repo.RelativePath, ids)
	if err != nil {
		return nil, err
	}
	list := make([]*Commit, 0, len(commits))
	for _, c := range commits {
		list = append(list, c)
	}
	s.attribute(ctx, repo, list)
	b.Commits = commits
	return b, nil
}

// SearchCode searches the files of a revision.
func (s *Service) SearchCode(ctx context.Context, name, rev, query string, max int) (*SearchResult, error) {
	repo, err := s.path(ctx, name)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidArgument)
	}
	if rev, err = s.revision(repo, rev); err != nil {
		return nil, err
	}
	return s.Gateway.SearchCode(ctx, repo.RelativePath, rev, query, max)
}

// attribute sets Landing on commits the changeset log knows, and derives
// trailer fields. Lookup failures leave the author fallback in place.
func (s *Service) attribute(ctx context.Context, repo *Repository, commits []*Commit) {
	if len(commits) == 0 {
		return
	}
	ids := make([]string, 0, len(commits))
	for _, c := range commits {
		ApplyTrailers(c)
		ids = append(ids, c.ID)
	}
	if s.Landings == nil {
		return
	}
	landings, err := s.Landings.Landings(ctx, repo.Name, ids)
	if err != nil {
		return
	}
	for _, c := range commits {
		if l, ok := landings[c.ID]; ok {
			l := l
			c.Landing = &l
		}
	}
}

func cleanPath(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	p = path.Clean(p)
	if p == "." || strings.HasPrefix(p, "../") || p == ".." {
		return ""
	}
	return p
}

func isReadme(name string) bool {
	n := strings.ToLower(name)
	return n == "readme" || n == "readme.md" || n == "readme.markdown" || n == "readme.txt" || n == "readme.rst"
}
