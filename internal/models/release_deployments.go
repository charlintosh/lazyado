package models

import (
	"strings"
	"time"
)

type DefinitionDeployment struct {
	DefinitionID int
	Release      string
	Stage        string
	State        StageState
	When         time.Time
}

func DeploymentState(status, operation string) StageState {
	op := strings.ToLower(operation)
	switch strings.ToLower(status) {
	case "succeeded":
		return StageSucceeded
	case "partiallysucceeded":
		return StagePartial
	case "failed":
		if strings.Contains(op, "rejected") {
			return StageRejected
		}
		if strings.Contains(op, "cancel") {
			return StageCanceled
		}
		return StageFailed
	case "inprogress":
		switch {
		case strings.Contains(op, "pending"), strings.Contains(op, "gate"), strings.Contains(op, "deferred"):
			return StageWaitingApproval
		case strings.Contains(op, "queued"):
			return StageQueued
		case strings.Contains(op, "scheduled"):
			return StageScheduled
		}
		return StageDeploying
	}
	if strings.Contains(op, "cancel") {
		return StageCanceled
	}
	return StageNotDeployed
}

func (s ReleaseStage) deployedAt() time.Time {
	if !s.Finished.IsZero() {
		return s.Finished
	}
	return s.Started
}

func CurrentDeployments(releases []Release) map[int]map[string]bool {
	type best struct {
		release int
		at      time.Time
		created time.Time
	}
	winners := map[string]best{}
	for _, r := range releases {
		for _, s := range r.Stages {
			if s.State != StageSucceeded && s.State != StagePartial {
				continue
			}
			at := s.deployedAt()
			if at.IsZero() {
				at = r.Created
			}
			candidate := best{release: r.ID, at: at, created: r.Created}
			cur, ok := winners[s.Name]
			if !ok || candidate.at.After(cur.at) || candidate.at.Equal(cur.at) && candidate.created.After(cur.created) {
				winners[s.Name] = candidate
			}
		}
	}
	out := make(map[int]map[string]bool, len(winners))
	for stage, w := range winners {
		if out[w.release] == nil {
			out[w.release] = map[string]bool{}
		}
		out[w.release][stage] = true
	}
	return out
}
