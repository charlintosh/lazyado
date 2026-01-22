package modals

import (
	"github.com/charlintosh/lazyado/internal/keys"
	"github.com/charlintosh/lazyado/internal/models"
	"github.com/charlintosh/lazyado/internal/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// BugParentModal is a thin wrapper around ParentModal preconfigured for Bugs
type BugParentModal struct {
	inner ParentModal
}

func NewBugParentModal(st styles.Styles, k keys.KeyMap) BugParentModal {
	m := NewParentModal(st, k)
	m.workItemType = workItemTypeBug
	return BugParentModal{inner: m}
}

func (p BugParentModal) Init() tea.Cmd { return p.inner.Init() }
func (p BugParentModal) View() string  { return p.inner.View() }

func (p BugParentModal) Update(msg tea.Msg) (BugParentModal, tea.Cmd) {
	newInner, cmd := p.inner.Update(msg)
	return BugParentModal{inner: newInner}, cmd
}

func (p *BugParentModal) SetSize(w, h int)  { p.inner.SetSize(w, h) }
func (p *BugParentModal) SetVisible(v bool) { p.inner.SetVisible(v) }
func (p *BugParentModal) IsVisible() bool   { return p.inner.IsVisible() }
func (p *BugParentModal) SetData(iterations []models.Iteration, areas []models.Area) {
	p.inner.SetData(iterations, areas)
}
func (p *BugParentModal) SetDefaultArea(areaPath string)         { p.inner.SetDefaultArea(areaPath) }
func (p *BugParentModal) SetMembers(members []models.TeamMember) { p.inner.SetMembers(members) }
