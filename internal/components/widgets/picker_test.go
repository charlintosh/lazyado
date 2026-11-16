package widgets

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charlintosh/lazyado/internal/keys"
)

func typeText(p Picker, text string) Picker {
	for _, r := range text {
		p, _ = p.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return p
}

func people() []Option {
	return []Option{
		{Label: "Dana Reyes", Value: "dana"},
		{Label: "Marco Lima", Value: "marco"},
		{Label: "Priya Nair", Value: "priya"},
	}
}

func TestSearchPickerFiltersFuzzily(t *testing.T) {
	p := NewPicker(keys.DefaultKeyMap(), people(), PickerConfig{Rows: 5, Search: true})
	p.SetWidth(40)
	p.Focus()
	p = typeText(p, "mlm")
	if p.Len() != 1 {
		t.Fatalf("fuzzy query matched %d options, want 1", p.Len())
	}
	if o, ok := p.Selected(); !ok || o.Value != "marco" {
		t.Fatalf("selected %+v, want marco", o)
	}
	p.SetQuery("")
	if p.Len() != 3 {
		t.Fatalf("clearing the query left %d options, want 3", p.Len())
	}
}

func TestSearchPickerSendsLettersToQuery(t *testing.T) {
	p := NewPicker(keys.DefaultKeyMap(), people(), PickerConfig{Rows: 5, Search: true})
	p.SetWidth(40)
	p.Focus()
	p = typeText(p, "j")
	if p.Query() != "j" {
		t.Fatalf("query = %q, want j", p.Query())
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if p.Query() != "j" {
		t.Fatal("arrow keys must navigate, not type")
	}
}

func TestStaticPickerNavigatesWithVimKeys(t *testing.T) {
	p := NewPicker(keys.DefaultKeyMap(), people(), PickerConfig{Rows: 5})
	p.SetWidth(40)
	p.Select("priya")
	if o, _ := p.Selected(); o.Value != "priya" {
		t.Fatalf("Select did not move the cursor, got %+v", o)
	}
	p = typeText(p, "kk")
	if o, _ := p.Selected(); o.Value != "dana" {
		t.Fatalf("k should move up, got %+v", o)
	}
	p = typeText(p, "G")
	if o, _ := p.Selected(); o.Value != "priya" {
		t.Fatalf("G should jump to the end, got %+v", o)
	}
}

func TestPickerPaginatesLongLists(t *testing.T) {
	var options []Option
	for _, l := range "abcdefghijkl" {
		options = append(options, Option{Label: string(l), Value: string(l)})
	}
	p := NewPicker(keys.DefaultKeyMap(), options, PickerConfig{Rows: 4})
	p.SetWidth(20)
	if p.Height() != 5 {
		t.Fatalf("height = %d, want 4 rows plus pagination", p.Height())
	}
	for range 5 {
		p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if o, _ := p.Selected(); o.Value != "f" {
		t.Fatalf("cursor should cross into the next page, got %+v", o)
	}
}
