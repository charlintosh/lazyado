package models

import "time"

type PRStatus string

const (
	PRStatusActive    PRStatus = "active"
	PRStatusCompleted PRStatus = "completed"
	PRStatusAbandoned PRStatus = "abandoned"
	PRStatusAll       PRStatus = "all"
)

type PRVote int

const (
	PRVoteApproved            PRVote = 10
	PRVoteApprovedSuggestions PRVote = 5
	PRVoteNoVote              PRVote = 0
	PRVoteWaitingForAuthor    PRVote = -5
	PRVoteRejected            PRVote = -10
)

func (v PRVote) Label() string {
	switch v {
	case PRVoteApproved:
		return "Approved"
	case PRVoteApprovedSuggestions:
		return "Approved w/ suggestions"
	case PRVoteNoVote:
		return "No vote"
	case PRVoteWaitingForAuthor:
		return "Waiting for author"
	case PRVoteRejected:
		return "Rejected"
	default:
		return "Unknown"
	}
}

func (v PRVote) Short() string {
	switch v {
	case PRVoteApproved:
		return "OK"
	case PRVoteApprovedSuggestions:
		return "OK*"
	case PRVoteNoVote:
		return "--"
	case PRVoteWaitingForAuthor:
		return "Wait"
	case PRVoteRejected:
		return "Rej"
	default:
		return "?"
	}
}

type PRMergeStatus string

const (
	PRMergeNotSet           PRMergeStatus = "notSet"
	PRMergeQueued           PRMergeStatus = "queued"
	PRMergeConflicts        PRMergeStatus = "conflicts"
	PRMergeSucceeded        PRMergeStatus = "succeeded"
	PRMergeRejectedByPolicy PRMergeStatus = "rejectedByPolicy"
	PRMergeFailure          PRMergeStatus = "failure"
)

type PRReviewer struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	Vote        PRVote `json:"vote"`
	IsRequired  bool   `json:"isRequired"`
	HasDeclined bool   `json:"hasDeclined"`
	IsFlagged   bool   `json:"isFlagged"`
	IsContainer bool   `json:"isContainer"`
}

type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PullRequest struct {
	ID           int           `json:"pullRequestId"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Status       PRStatus      `json:"status"`
	CreatedBy    string        `json:"createdBy"`
	CreatedByID  string        `json:"createdByID"`
	CreationDate time.Time     `json:"creationDate"`
	ClosedDate   time.Time     `json:"closedDate,omitempty"`
	SourceBranch string        `json:"sourceRefName"`
	TargetBranch string        `json:"targetRefName"`
	MergeStatus  PRMergeStatus `json:"mergeStatus"`
	IsDraft      bool          `json:"isDraft"`
	Repository   Repository    `json:"repository"`
	Reviewers    []PRReviewer  `json:"reviewers"`
	URL          string        `json:"url"`
	WebURL       string        `json:"webUrl"`
}

func (pr *PullRequest) ShortSourceBranch() string {
	return trimRefPrefix(pr.SourceBranch)
}

func (pr *PullRequest) ShortTargetBranch() string {
	return trimRefPrefix(pr.TargetBranch)
}

func trimRefPrefix(ref string) string {
	const prefix = "refs/heads/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ref
}

func (pr *PullRequest) StatusLabel() string {
	switch pr.Status {
	case PRStatusActive:
		return "Active"
	case PRStatusCompleted:
		return "Completed"
	case PRStatusAbandoned:
		return "Abandoned"
	default:
		return string(pr.Status)
	}
}

func AllPRVotes() []PRVote {
	return []PRVote{
		PRVoteApproved,
		PRVoteApprovedSuggestions,
		PRVoteNoVote,
		PRVoteWaitingForAuthor,
		PRVoteRejected,
	}
}
