package models

import (
	"fmt"
	"strings"
)

const (
	QualifierState    = "state:"
	QualifierBranch   = "branch:"
	QualifierPipeline = "pipeline:"
	QualifierBy       = "by:"
)

var RunQualifiers = []string{QualifierState, QualifierBranch, QualifierPipeline, QualifierBy + MeAlias}

var RunStateFilters = []string{"running", "queued", "succeeded", "failed", "partial", "canceled"}

type RunCriteria struct {
	PipelineID   int
	Ref          string
	Status       string
	Result       string
	RequestedFor string
}

func (c RunCriteria) Empty() bool { return c == RunCriteria{} }

type RunQuery struct {
	Terms    []string
	State    string
	Branch   string
	Pipeline string
	By       string
}

func ParseRunQuery(input string) RunQuery {
	var q RunQuery
	for _, t := range searchTokens(input) {
		lower := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lower, QualifierState):
			q.State = strings.ToLower(unquote(t[len(QualifierState):]))
		case strings.HasPrefix(lower, QualifierBranch):
			q.Branch = unquote(t[len(QualifierBranch):])
		case strings.HasPrefix(lower, QualifierPipeline):
			q.Pipeline = unquote(t[len(QualifierPipeline):])
		case strings.HasPrefix(lower, QualifierBy):
			q.By = unquote(t[len(QualifierBy):])
		default:
			q.Terms = append(q.Terms, strings.ToLower(unquote(t)))
		}
	}
	return q
}

func (q RunQuery) Active() bool {
	return len(q.Terms) > 0 || q.State != "" || q.Branch != "" || q.Pipeline != "" || q.By != ""
}

func (q RunQuery) Matches(r PipelineRun) bool {
	haystack := strings.ToLower(strings.Join([]string{
		r.Number, r.Title(), r.PipelineName, r.Branch(), r.RequestedFor.DisplayName, r.ShortCommit(),
	}, "\n"))
	for _, term := range q.Terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func (q RunQuery) Criteria(pipelines []Pipeline, me Identity) (RunCriteria, error) {
	var c RunCriteria
	switch q.State {
	case "":
	case "running":
		c.Status = "inProgress"
	case "queued":
		c.Status = "notStarted"
	case "succeeded", "failed", "canceled":
		c.Result = q.State
	case "partial":
		c.Result = "partiallySucceeded"
	default:
		return c, fmt.Errorf("state: can be %s", strings.Join(RunStateFilters, ", "))
	}
	c.Ref = BranchRef(q.Branch)
	if q.By != "" {
		if !strings.EqualFold(q.By, MeAlias) {
			return c, fmt.Errorf("by: only supports %s", MeAlias)
		}
		if me.ID == "" {
			return c, fmt.Errorf("Couldn't confirm your identity from the PAT, so by:%s is unavailable", MeAlias)
		}
		c.RequestedFor = me.ID
	}
	if q.Pipeline != "" {
		p, err := pipelineNamed(q.Pipeline, pipelines)
		if err != nil {
			return c, err
		}
		c.PipelineID = p.ID
	}
	return c, nil
}

func pipelineNamed(name string, pipelines []Pipeline) (Pipeline, error) {
	var matches []Pipeline
	for _, p := range pipelines {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
		if containsFold(p.Name, name) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return Pipeline{}, fmt.Errorf("No pipeline named %q", name)
	case 1:
		return matches[0], nil
	}
	return Pipeline{}, fmt.Errorf("pipeline:%s matches %d pipelines, type more of the name", name, len(matches))
}

func PipelineMatches(p Pipeline, terms []string) bool {
	haystack := strings.ToLower(p.Name + "\n" + p.Folder)
	for _, t := range terms {
		if !strings.Contains(haystack, t) {
			return false
		}
	}
	return true
}

func SearchTerms(input string) []string {
	var out []string
	for _, t := range searchTokens(input) {
		out = append(out, strings.ToLower(unquote(t)))
	}
	return out
}

func NamePattern(input string) string {
	text := strings.TrimSpace(strings.Join(SearchTerms(input), " "))
	if text == "" {
		return ""
	}
	return "*" + strings.Trim(text, "*") + "*"
}
