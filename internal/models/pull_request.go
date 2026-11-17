package models

import (
	"fmt"
	"strings"
	"time"
)

type PullRequestStatus string

const (
	PullRequestActive    PullRequestStatus = "active"
	PullRequestCompleted PullRequestStatus = "completed"
	PullRequestAbandoned PullRequestStatus = "abandoned"
)

type Vote int

const (
	VoteApproved                Vote = 10
	VoteApprovedWithSuggestions Vote = 5
	VoteNone                    Vote = 0
	VoteWaitingForAuthor        Vote = -5
	VoteRejected                Vote = -10
)

var Votes = []Vote{VoteApproved, VoteApprovedWithSuggestions, VoteWaitingForAuthor, VoteRejected, VoteNone}

func (v Vote) Label() string {
	switch v {
	case VoteApproved:
		return "Approved"
	case VoteApprovedWithSuggestions:
		return "Approved with suggestions"
	case VoteWaitingForAuthor:
		return "Waiting for author"
	case VoteRejected:
		return "Rejected"
	default:
		return "No vote"
	}
}

func (v Vote) Action() string {
	switch v {
	case VoteApproved:
		return "Approve"
	case VoteApprovedWithSuggestions:
		return "Approve with suggestions"
	case VoteWaitingForAuthor:
		return "Wait for author"
	case VoteRejected:
		return "Reject"
	default:
		return "Reset vote"
	}
}

type Reviewer struct {
	ID          string
	DisplayName string
	UniqueName  string
	Vote        Vote
	Required    bool
	Group       bool
}

const AllRepositories = "*"

type Repository struct {
	ID   string
	Name string
}

func (r Reviewer) Name() string {
	name := strings.TrimSpace(r.DisplayName)
	if !strings.HasPrefix(name, "[") {
		return name
	}
	if end := strings.Index(name, "]"); end >= 0 {
		if short := strings.TrimSpace(strings.TrimLeft(name[end+1:], `\/`)); short != "" {
			return short
		}
	}
	return name
}

type PullRequest struct {
	ID             int
	Title          string
	Description    string
	Status         PullRequestStatus
	IsDraft        bool
	Author         Identity
	SourceBranch   string
	TargetBranch   string
	RepositoryID   string
	RepositoryName string
	ProjectID      string
	MergeStatus    string
	CreatedDate    time.Time
	ClosedDate     time.Time
	Reviewers      []Reviewer
	Labels         []string
	WebURL         string
}

func (p PullRequest) ReviewerFor(me Identity) (Reviewer, bool) {
	for _, r := range p.Reviewers {
		if me.Is(r.ID, r.UniqueName) {
			return r, true
		}
	}
	return Reviewer{}, false
}

func (p PullRequest) AuthoredBy(me Identity) bool {
	return me.Is(p.Author.ID, p.Author.UniqueName)
}

func (p PullRequest) Involves(me Identity) bool {
	_, reviewing := p.ReviewerFor(me)
	return reviewing || p.AuthoredBy(me)
}

func (p PullRequest) HasConflicts() bool {
	return strings.EqualFold(p.MergeStatus, "conflicts")
}

func (i Identity) Is(id, uniqueName string) bool {
	switch {
	case i.ID != "" && id != "":
		return strings.EqualFold(i.ID, id)
	case i.UniqueName != "" && uniqueName != "":
		return strings.EqualFold(i.UniqueName, uniqueName)
	}
	return false
}

func BranchName(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

const (
	ThreadActive  = "active"
	ThreadPending = "pending"
	ThreadFixed   = "fixed"
)

type PullRequestComment struct {
	ID         int
	ParentID   int
	AuthorID   string
	AuthorName string
	AuthorUPN  string
	Content    string
	Date       time.Time
	System     bool
	Likes      []Identity
}

type PullRequestThread struct {
	ID       int
	Status   string
	FilePath string
	Line     int
	EndLine  int
	LeftSide bool
	Date     time.Time
	Comments []PullRequestComment
}

func (t PullRequestThread) LastLine() int {
	return max(t.Line, t.EndLine)
}

func (t PullRequestThread) MultiLine() bool {
	return t.Line > 0 && t.EndLine > t.Line
}

func LineRange(first, last int) string {
	if last > first {
		return fmt.Sprintf("%d-%d", first, last)
	}
	return fmt.Sprintf("%d", first)
}

func (c PullRequestComment) WrittenBy(me Identity) bool {
	return !c.System && me.Is(c.AuthorID, c.AuthorUPN)
}

func (c PullRequestComment) LikedBy(me Identity) bool {
	for _, l := range c.Likes {
		if me.Is(l.ID, l.UniqueName) {
			return true
		}
	}
	return false
}

func (t PullRequestThread) Collapsed(expanded map[int]bool) bool {
	return t.Resolvable() && !t.Active() && !expanded[t.ID]
}

func (t PullRequestThread) Comment(id int) (PullRequestComment, bool) {
	for _, c := range t.Comments {
		if c.ID == id {
			return c, true
		}
	}
	return PullRequestComment{}, false
}

func (t PullRequestThread) System() bool {
	for _, c := range t.Comments {
		if !c.System {
			return false
		}
	}
	return true
}

func (t PullRequestThread) Resolvable() bool {
	return t.Status != "" && !t.System()
}

func (t PullRequestThread) Active() bool {
	return t.Resolvable() && (t.Status == ThreadActive || t.Status == ThreadPending)
}

func ActiveThreads(threads []PullRequestThread) int {
	n := 0
	for _, t := range threads {
		if t.Active() {
			n++
		}
	}
	return n
}

type CheckState int

const (
	CheckPending CheckState = iota
	CheckRunning
	CheckPassed
	CheckFailed
)

func (s CheckState) Label() string {
	switch s {
	case CheckPassed:
		return "PASS"
	case CheckFailed:
		return "FAIL"
	case CheckRunning:
		return "RUN"
	default:
		return "WAIT"
	}
}

type PullRequestCheck struct {
	Name     string
	Detail   string
	State    CheckState
	Required bool
	BuildID  int
}

func BuildChecks(checks []PullRequestCheck) []PullRequestCheck {
	var out []PullRequestCheck
	seen := map[int]bool{}
	for _, c := range checks {
		if c.BuildID > 0 && !seen[c.BuildID] {
			seen[c.BuildID] = true
			out = append(out, c)
		}
	}
	return out
}

func BlockingChecks(checks []PullRequestCheck) int {
	n := 0
	for _, c := range checks {
		if c.Required && c.State != CheckPassed {
			n++
		}
	}
	return n
}

type ChangeKind int

const (
	ChangeEdit ChangeKind = iota
	ChangeAdd
	ChangeDelete
	ChangeRename
)

func (k ChangeKind) Letter() string {
	switch k {
	case ChangeAdd:
		return "A"
	case ChangeDelete:
		return "D"
	case ChangeRename:
		return "R"
	default:
		return "M"
	}
}

type PullRequestFile struct {
	Path         string
	OriginalPath string
	Kind         ChangeKind
}

type PullRequestChanges struct {
	Iteration  int
	BaseCommit string
	HeadCommit string
	Files      []PullRequestFile
	Attributes string
}

type ThreadAnchor struct {
	FilePath  string
	Line      int
	EndLine   int
	EndOffset int
	LeftSide  bool
}

func (a ThreadAnchor) Lines() string {
	return LineRange(a.Line, a.EndLine)
}
