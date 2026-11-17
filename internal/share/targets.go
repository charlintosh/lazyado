package share

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

func WorkItem(item models.WorkItem) Target {
	id := strconv.Itoa(item.ID)
	kind := string(item.Type)
	if kind == "" {
		kind = "Work item"
	}
	return newTarget("work item "+id, fmt.Sprintf("%s %s · %s", kind, id, item.Title)).
		add(Link, "Link", item.WebURL).
		add(ID, "ID", id).
		add(Mention, "Mention", "#"+id).
		add(GitHub, "GitHub commit link", "AB#"+id).
		add(Markdown, "Markdown link", markdownLink(id+" "+item.Title, item.WebURL)).
		add(Title, "Title and link", titled(fmt.Sprintf("%s %s: %s", kind, id, item.Title), item.WebURL)).
		build()
}

func PullRequest(pr models.PullRequest) Target {
	id := "!" + strconv.Itoa(pr.ID)
	return newTarget("pull request "+id, fmt.Sprintf("Pull request %s · %s", id, pr.Title)).
		add(Link, "Link", pr.WebURL).
		add(ID, "ID", id).
		add(Markdown, "Markdown link", markdownLink(id+" "+pr.Title, pr.WebURL)).
		add(Title, "Title and link", titled(fmt.Sprintf("Pull request %s: %s", id, pr.Title), pr.WebURL)).
		add(Name, "Source branch", pr.SourceBranch).
		build()
}

func PullRequestFile(pr models.PullRequest, commit, path string) Target {
	link := pullRequestFileURL(pr, path)
	permalink := commitFileURL(pullRequestRepoURL(pr), path, commit, Lines{})
	return newTarget(baseName(path), fmt.Sprintf("!%d · %s", pr.ID, trimSlash(path))).
		add(Link, "Link to file in pull request", link).
		add(Permalink, "Permalink ("+models.ShortCommit(commit)+")", permalink).
		add(Path, "Path", trimSlash(path)).
		add(Markdown, "Markdown link", markdownLink(fmt.Sprintf("%s in !%d", trimSlash(path), pr.ID), link)).
		build()
}

func PullRequestLines(pr models.PullRequest, commit, path string, lines Lines) Target {
	permalink := commitFileURL(pullRequestRepoURL(pr), path, commit, lines)
	where := trimSlash(path) + lines.Suffix()
	return newTarget(baseName(path)+lines.Suffix(), fmt.Sprintf("!%d · %s · %s", pr.ID, trimSlash(path), lines.Label())).
		add(Link, "Link to file in pull request", pullRequestFileURL(pr, path)).
		add(Permalink, "Permalink to "+lines.Label(), permalink).
		add(Path, "Path and lines", where).
		add(Snippet, "Snippet (Markdown)", lines.Snippet(path, permalink)).
		add(Markdown, "Markdown link", markdownLink(fmt.Sprintf("%s#%s in !%d", baseName(path), lines.Anchor(), pr.ID), permalink)).
		build()
}

type Code struct {
	RepoURL  string
	RepoName string
	Path     string
	Version  models.GitVersion
	Commit   string
	Folder   bool
}

func (c Code) link(lines Lines) string {
	return FileURL(c.RepoURL, c.Path, VersionParam(c.Version), lines)
}

func (c Code) permalink(lines Lines) string {
	if c.Commit == "" {
		return ""
	}
	return commitFileURL(c.RepoURL, c.Path, c.Commit, lines)
}

func (c Code) subject(extra string) string {
	path := trimSlash(c.Path)
	if path == "" {
		path = "/"
	}
	parts := []string{c.RepoName, path}
	if extra != "" {
		parts = append(parts, extra)
	}
	if c.Version.Name != "" {
		parts[len(parts)-1] += " on " + c.Version.Name
	}
	return strings.Join(parts, " · ")
}

func File(c Code) Target {
	link := c.link(Lines{})
	path := trimSlash(c.Path)
	if path == "" {
		path = "/"
	}
	return newTarget(baseName(c.Path), c.subject("")).
		add(Link, linkLabel(c.Version), link).
		add(Permalink, "Permalink ("+models.ShortCommit(c.Commit)+")", c.permalink(Lines{})).
		add(Path, "Path", path).
		add(Markdown, "Markdown link", markdownLink(path, link)).
		build()
}

func FileLines(c Code, lines Lines) Target {
	link := c.link(lines)
	permalink := c.permalink(lines)
	stable := permalink
	if stable == "" {
		stable = link
	}
	return newTarget(baseName(c.Path)+lines.Suffix(), c.subject(lines.Label())).
		add(Link, "Link to "+lines.Label()+linkSuffix(c.Version), link).
		add(Permalink, "Permalink ("+models.ShortCommit(c.Commit)+")", permalink).
		add(Path, "Path and lines", trimSlash(c.Path)+lines.Suffix()).
		add(Snippet, "Snippet (Markdown)", lines.Snippet(c.Path, stable)).
		add(Markdown, "Markdown link", markdownLink(baseName(c.Path)+"#"+lines.Anchor(), stable)).
		build()
}

func Commit(repoURL string, c models.CommitRef) Target {
	link := ""
	if repoURL != "" && c.ID != "" {
		link = repoURL + "/commit/" + c.ID
	}
	short := models.ShortCommit(c.ID)
	label := strings.TrimSpace(short + " " + c.Subject())
	return newTarget("commit "+short, "Commit "+label).
		add(Link, "Link", link).
		add(ID, "Commit ID", c.ID).
		add(Markdown, "Markdown link", markdownLink(label, link)).
		add(Title, "Title and link", titled(label, link)).
		build()
}

func Ref(kind, name, link string) Target {
	return newTarget(strings.ToLower(kind)+" "+name, kind+" "+name).
		add(Link, "Link", link).
		add(Name, "Name", name).
		add(Markdown, "Markdown link", markdownLink(name, link)).
		build()
}

func Pipeline(p models.Pipeline) Target {
	return newTarget("pipeline "+p.Name, fmt.Sprintf("Pipeline %s · definition %d", p.Name, p.ID)).
		add(Link, "Link", p.WebURL).
		add(ID, "Definition ID", strconv.Itoa(p.ID)).
		add(Name, "Name", p.Name).
		add(Markdown, "Markdown link", markdownLink(p.Name, p.WebURL)).
		build()
}

func Run(run models.PipelineRun, logsURL, logsLabel string) Target {
	name := strings.TrimSpace(run.PipelineName + " " + run.Number)
	if logsURL == run.WebURL {
		logsURL = ""
	}
	subject := name
	if logsLabel != "" && logsURL != "" {
		subject += " · " + logsLabel
	}
	return newTarget("run "+run.Number, subject).
		add(Link, "Link", run.WebURL).
		add(ID, "Build number", run.Number).
		add(RunID, "Run ID", positive(run.ID)).
		add(Logs, "Link to job logs", logsURL).
		add(Markdown, "Markdown link", markdownLink(name, run.WebURL)).
		add(Title, "Title, result and link", titled(fmt.Sprintf("%s (%s)", name, run.State().Name()), run.WebURL)).
		add(PipelineLink, "Pipeline link", pipelineURL(run)).
		build()
}

func Release(r models.Release) Target {
	name := strings.TrimSpace(r.Definition + " · " + r.Name)
	if r.Definition == "" {
		name = r.Name
	}
	return newTarget(r.Name, name).
		add(Link, "Link", r.WebURL).
		add(Name, "Name", r.Name).
		add(Markdown, "Markdown link", markdownLink(name, r.WebURL)).
		add(Title, "Title, stages and link", titled(releaseSummary(r), r.WebURL)).
		add(PipelineLink, "Release pipeline link", releaseDefinitionURL(r)).
		build()
}

func ReleaseDefinition(d models.ReleaseDefinition) Target {
	return newTarget("release pipeline "+d.Name, fmt.Sprintf("Release pipeline %s · definition %d", d.Name, d.ID)).
		add(Link, "Link", d.WebURL).
		add(ID, "Definition ID", strconv.Itoa(d.ID)).
		add(Name, "Name", d.Name).
		add(Markdown, "Markdown link", markdownLink(d.Name, d.WebURL)).
		build()
}

func releaseSummary(r models.Release) string {
	title := strings.TrimSpace(r.Definition + " " + r.Name)
	var stages []string
	for _, s := range r.Stages {
		stages = append(stages, s.Name+": "+s.State.Name())
	}
	if len(stages) == 0 {
		return title
	}
	return title + " (" + strings.Join(stages, ", ") + ")"
}

func positive(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func linkLabel(v models.GitVersion) string {
	return "Link" + linkSuffix(v)
}

func linkSuffix(v models.GitVersion) string {
	if v.Name == "" {
		return ""
	}
	return " (" + v.Name + ")"
}
