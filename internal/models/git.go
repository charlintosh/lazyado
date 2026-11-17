package models

import (
	"errors"
	"path"
	"slices"
	"strings"
	"time"
)

type VersionKind int

const (
	VersionBranch VersionKind = iota
	VersionTag
	VersionCommit
)

const shortCommitLength = 7

type GitVersion struct {
	Kind VersionKind
	Name string
}

func BranchVersion(name string) GitVersion { return GitVersion{Kind: VersionBranch, Name: name} }

func (v GitVersion) Type() string {
	switch v.Kind {
	case VersionTag:
		return "tag"
	case VersionCommit:
		return "commit"
	}
	return "branch"
}

func (v GitVersion) Label() string {
	if v.Kind == VersionCommit {
		return ShortCommit(v.Name)
	}
	return v.Name
}

func (v GitVersion) IsZero() bool { return v.Name == "" }

func ShortCommit(id string) string {
	if len(id) > shortCommitLength {
		return id[:shortCommitLength]
	}
	return id
}

type CommitRef struct {
	ID      string
	Comment string
	Author  string
	Date    time.Time
}

func (c CommitRef) Subject() string {
	subject, _, _ := strings.Cut(strings.TrimSpace(c.Comment), "\n")
	return strings.TrimSpace(subject)
}

type GitItem struct {
	Path   string
	Folder bool
	Last   CommitRef
}

func (i GitItem) Name() string {
	if i.Path == "/" || i.Path == "" {
		return "/"
	}
	return path.Base(i.Path)
}

func (i GitItem) IsMarkdown() bool {
	return strings.EqualFold(path.Ext(i.Path), ".md")
}

func ParentPath(p string) string {
	if p == "/" || p == "" {
		return ""
	}
	return path.Dir(p)
}

func SortGitItems(items []GitItem) []GitItem {
	out := slices.Clone(items)
	slices.SortStableFunc(out, func(a, b GitItem) int {
		if a.Folder != b.Folder {
			if a.Folder {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
	return out
}

func FolderReadme(items []GitItem) (GitItem, bool) {
	for _, it := range items {
		if !it.Folder && strings.EqualFold(it.Name(), "README.md") {
			return it, true
		}
	}
	return GitItem{}, false
}

func IsBinary(text string) bool {
	sample := text
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	return strings.IndexByte(sample, 0) >= 0
}

type Commit struct {
	CommitRef
	Committer string
	Parents   []string
	Adds      int
	Edits     int
	Deletes   int
}

func (c Commit) IsMerge() bool { return len(c.Parents) > 1 }

func (c Commit) Body() string {
	_, body, _ := strings.Cut(strings.TrimSpace(c.Comment), "\n")
	return strings.TrimSpace(body)
}

type CommitFile struct {
	Path string
	Kind ChangeKind
}

type CommitDetail struct {
	Commit Commit
	Files  []CommitFile
}

type CommitCriteria struct {
	Author string
	Path   string
}

func (c CommitCriteria) IsZero() bool { return c == CommitCriteria{} }

const (
	authorQualifier = "author:"
	pathQualifier   = "path:"
)

func ParseCommitQuery(query string) (CommitCriteria, string) {
	var criteria CommitCriteria
	var text []string
	for _, word := range strings.Fields(query) {
		lower := strings.ToLower(word)
		switch {
		case strings.HasPrefix(lower, authorQualifier) && len(word) > len(authorQualifier):
			criteria.Author = word[len(authorQualifier):]
		case strings.HasPrefix(lower, pathQualifier) && len(word) > len(pathQualifier):
			criteria.Path = "/" + strings.TrimPrefix(word[len(pathQualifier):], "/")
		default:
			text = append(text, word)
		}
	}
	return criteria, strings.Join(text, " ")
}

func (c CommitCriteria) Query() string {
	var parts []string
	if c.Author != "" {
		parts = append(parts, authorQualifier+c.Author)
	}
	if c.Path != "" {
		parts = append(parts, pathQualifier+c.Path)
	}
	return strings.Join(parts, " ")
}

func CommitMatches(c Commit, text string) bool {
	for _, term := range strings.Fields(strings.ToLower(text)) {
		if !strings.Contains(strings.ToLower(c.Comment), term) && !strings.Contains(strings.ToLower(c.Author), term) && !strings.HasPrefix(strings.ToLower(c.ID), term) {
			return false
		}
	}
	return true
}

const StaleAfter = 90 * 24 * time.Hour

type Branch struct {
	Name          string
	ObjectID      string
	Last          CommitRef
	Ahead         int
	Behind        int
	Default       bool
	Locked        bool
	Mine          bool
	Creator       string
	PullRequestID int
}

func (b Branch) Stale(now time.Time) bool {
	return !b.Last.Date.IsZero() && now.Sub(b.Last.Date) > StaleAfter
}

func ValidateRefName(name string) error {
	switch {
	case name == "":
		return errors.New("enter a name")
	case strings.ContainsAny(name, " ~^:?*[\\\t"):
		return errors.New("a name can't contain spaces or any of ~ ^ : ? * [ \\")
	case strings.Contains(name, "..") || strings.Contains(name, "//") || strings.Contains(name, "@{"):
		return errors.New("a name can't contain .., // or @{")
	case strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || strings.HasSuffix(name, ".") || strings.HasSuffix(name, ".lock"):
		return errors.New("a name can't start or end with /, or end with . or .lock")
	case strings.HasPrefix(name, "-"):
		return errors.New("a name can't start with -")
	}
	return nil
}

type Tag struct {
	Name     string
	ObjectID string
	CommitID string
}

func (t Tag) Annotated() bool { return t.ObjectID != "" && t.ObjectID != t.CommitID }

type TagDetail struct {
	Name     string
	Message  string
	Tagger   string
	Date     time.Time
	CommitID string
}
