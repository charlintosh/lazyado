package share

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/charlintosh/lazyado/internal/models"
)

var versionPrefix = map[models.VersionKind]string{
	models.VersionBranch: "GB",
	models.VersionTag:    "GT",
	models.VersionCommit: "GC",
}

func VersionParam(v models.GitVersion) string {
	if v.Name == "" {
		return ""
	}
	return versionPrefix[v.Kind] + v.Name
}

type Lines struct {
	From int
	To   int
	Code []string
}

func (l Lines) Valid() bool {
	return l.From > 0 && l.To >= l.From
}

func (l Lines) Label() string {
	if l.From == l.To {
		return fmt.Sprintf("line %d", l.From)
	}
	return fmt.Sprintf("lines %d–%d", l.From, l.To)
}

func (l Lines) Suffix() string {
	if !l.Valid() {
		return ""
	}
	if l.From == l.To {
		return fmt.Sprintf(":%d", l.From)
	}
	return fmt.Sprintf(":%d-%d", l.From, l.To)
}

func (l Lines) Anchor() string {
	if l.From == l.To {
		return fmt.Sprintf("L%d", l.From)
	}
	return fmt.Sprintf("L%d-%d", l.From, l.To)
}

func (l Lines) Snippet(filePath, link string) string {
	if len(l.Code) == 0 {
		return ""
	}
	code := strings.Join(l.Code, "\n")
	fence := codeFence(code)
	out := fence + fenceLanguage(filePath) + "\n" + code + "\n" + fence
	if link != "" {
		out += "\n" + link
	}
	return out
}

func codeFence(code string) string {
	longest, run := 0, 0
	for _, r := range code {
		if r == '`' {
			run++
			longest = max(longest, run)
			continue
		}
		run = 0
	}
	return strings.Repeat("`", max(3, longest+1))
}

var fenceAliases = map[string]string{
	"yml": "yaml",
	"md":  "markdown",
	"sh":  "bash",
	"cs":  "csharp",
	"ps1": "powershell",
}

func fenceLanguage(filePath string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(filePath), "."))
	if alias, ok := fenceAliases[ext]; ok {
		return alias
	}
	return ext
}

func FileURL(repoURL, filePath, version string, lines Lines) string {
	if repoURL == "" || filePath == "" {
		return ""
	}
	q := url.Values{}
	q.Set("path", filePath)
	if version != "" {
		q.Set("version", version)
	}
	if lines.Valid() {
		q.Set("line", strconv.Itoa(lines.From))
		q.Set("lineEnd", strconv.Itoa(lines.To+1))
		q.Set("lineStartColumn", "1")
		q.Set("lineEndColumn", "1")
		q.Set("lineStyle", "plain")
		q.Set("_a", "contents")
	}
	return repoURL + "?" + q.Encode()
}

func commitFileURL(repoURL, filePath, commit string, lines Lines) string {
	if commit == "" {
		return ""
	}
	return FileURL(repoURL, filePath, versionPrefix[models.VersionCommit]+commit, lines)
}

func pullRequestRepoURL(pr models.PullRequest) string {
	repo, _, found := strings.Cut(pr.WebURL, "/pullrequest/")
	if !found {
		return ""
	}
	return repo
}

func pullRequestFileURL(pr models.PullRequest, filePath string) string {
	if pr.WebURL == "" {
		return ""
	}
	return pr.WebURL + "?_a=files&path=" + url.QueryEscape(filePath)
}

func pipelineURL(run models.PipelineRun) string {
	base, _, found := strings.Cut(run.WebURL, "/_build/results")
	if !found || run.PipelineID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/_build?definitionId=%d", base, run.PipelineID)
}

func releaseDefinitionURL(r models.Release) string {
	base, _, found := strings.Cut(r.WebURL, "/_release")
	if !found || r.DefinitionID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/_release?definitionId=%d", base, r.DefinitionID)
}

func baseName(filePath string) string {
	trimmed := strings.TrimSuffix(filePath, "/")
	if trimmed == "" {
		return "/"
	}
	return path.Base(trimmed)
}

func trimSlash(filePath string) string {
	return strings.TrimPrefix(filePath, "/")
}
