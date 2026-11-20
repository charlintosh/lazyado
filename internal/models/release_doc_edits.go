package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type DocPath []any

func (p DocPath) String() string {
	parts := make([]string, len(p))
	for i, s := range p {
		parts[i] = fmt.Sprint(s)
	}
	return strings.Join(parts, "/")
}

func (p DocPath) Has(prefix DocPath) bool {
	return len(p) >= len(prefix) && reflect.DeepEqual(p[:len(prefix)], prefix)
}

func (p DocPath) Child(keys ...any) DocPath {
	out := make(DocPath, 0, len(p)+len(keys))
	return append(append(out, p...), keys...)
}

type DocEdit struct {
	Path       DocPath
	Where      string
	Field      string
	Before     any
	After      any
	Secret     bool
	Kind       InputKind
	BeforeText string
	AfterText  string
	Guard      DocPath
	GuardValue any
}

type DocEdits []DocEdit

func (es DocEdits) Find(path DocPath) (DocEdit, bool) {
	key := path.String()
	for _, e := range es {
		if e.Path.String() == key {
			return e, true
		}
	}
	return DocEdit{}, false
}

func (es DocEdits) Value(path DocPath, original any) any {
	if e, ok := es.Find(path); ok {
		return e.After
	}
	return original
}

func (es DocEdits) With(e DocEdit) DocEdits {
	out := es.Without(e.Path)
	if sameValue(e.Before, e.After) {
		return out
	}
	return append(out, e)
}

func (es DocEdits) Without(path DocPath) DocEdits {
	key := path.String()
	out := make(DocEdits, 0, len(es))
	for _, e := range es {
		if e.Path.String() != key {
			out = append(out, e)
		}
	}
	return out
}

func (es DocEdits) WithoutUnder(prefix DocPath) DocEdits {
	out := make(DocEdits, 0, len(es))
	for _, e := range es {
		if !e.Path.Has(prefix) {
			out = append(out, e)
		}
	}
	return out
}

func (es DocEdits) CountUnder(prefix DocPath) int {
	n := 0
	for _, e := range es {
		if e.Path.Has(prefix) {
			n++
		}
	}
	return n
}

func sameValue(a, b any) bool {
	return reflect.DeepEqual(normalize(a), normalize(b))
}

func normalize(v any) any {
	if n, ok := v.(json.Number); ok {
		return n.String()
	}
	return v
}

func (d *ReleaseDoc) TaskPath(stage DocStage, phase DocPhase, task DocTask) DocPath {
	return DocPath{"environments", stage.Index, d.phasesKey(), phase.Index, "workflowTasks", task.Index}
}

func (d *ReleaseDoc) VariablePath(v DocVariable) DocPath {
	if v.Stage < 0 {
		return DocPath{"variables", v.Name, "value"}
	}
	return DocPath{"environments", v.Stage, "variables", v.Name, "value"}
}

func (d *ReleaseDoc) StageOfIndex(index int) (DocStage, bool) {
	for _, s := range d.Stages {
		if s.Index == index {
			return s, true
		}
	}
	return DocStage{}, false
}

func (d *ReleaseDoc) ValueAt(path DocPath) any {
	var cur any = d.raw
	for _, step := range path {
		switch node := cur.(type) {
		case map[string]any:
			key, _ := step.(string)
			cur = node[key]
		case []any:
			i, ok := step.(int)
			if !ok || i < 0 || i >= len(node) {
				return nil
			}
			cur = node[i]
		default:
			return nil
		}
	}
	return cur
}

func setAt(root map[string]any, path DocPath, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	var cur any = root
	for i, step := range path {
		last := i == len(path)-1
		switch node := cur.(type) {
		case map[string]any:
			key, ok := step.(string)
			if !ok {
				return fmt.Errorf("%s: expected a name at %v", path, step)
			}
			if last {
				node[key] = value
				return nil
			}
			next, exists := node[key]
			if !exists || next == nil {
				next = map[string]any{}
				node[key] = next
			}
			cur = next
		case []any:
			idx, ok := step.(int)
			if !ok || idx < 0 || idx >= len(node) {
				return fmt.Errorf("%s: no item %v", path, step)
			}
			if last {
				node[idx] = value
				return nil
			}
			cur = node[idx]
		default:
			return fmt.Errorf("%s: can't set inside %T", path, cur)
		}
	}
	return nil
}

func (d *ReleaseDoc) Payload(edits DocEdits, comment string) ([]byte, error) {
	body, err := json.Marshal(d.raw)
	if err != nil {
		return nil, fmt.Errorf("encoding the release pipeline: %w", err)
	}
	root, err := decodeTree(body)
	if err != nil {
		return nil, err
	}
	for _, e := range edits {
		if err := setAt(root, e.Path, e.After); err != nil {
			return nil, fmt.Errorf("applying %s: %w", e.Field, err)
		}
	}
	if comment != "" {
		root["comment"] = comment
	}
	return json.Marshal(root)
}

func ReapplyEdits(latest *ReleaseDoc, edits DocEdits) (DocEdits, DocEdits) {
	var kept, dropped DocEdits
	for _, e := range edits {
		if e.Guard != nil && !sameValue(latest.ValueAt(e.Guard), e.GuardValue) {
			dropped = append(dropped, e)
			continue
		}
		current := latest.ValueAt(e.Path)
		switch {
		case sameValue(current, e.Before):
			kept = append(kept, e)
		case !sameValue(current, e.After):
			dropped = append(dropped, e)
		}
	}
	return kept, dropped
}

func FormatDocValue(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	case string:
		return v
	case json.Number:
		return v.String()
	}
	return fmt.Sprint(v)
}

func Toggled(v any) any {
	switch v := v.(type) {
	case bool:
		return !v
	case string:
		b, _ := strconv.ParseBool(v)
		return strconv.FormatBool(!b)
	}
	return true
}

func Truthy(v any) bool {
	switch v := v.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	}
	return false
}
