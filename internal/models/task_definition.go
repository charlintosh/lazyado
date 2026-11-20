package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type TaskRef struct {
	ID    string
	Major int
}

func (r TaskRef) Key() string { return strings.ToLower(r.ID) + "@" + strconv.Itoa(r.Major) }

type TaskDefinition struct {
	ID           string
	Name         string
	FriendlyName string
	Major        int
	Version      string
	Inputs       []TaskInput
	Groups       []TaskInputGroup
}

type TaskInputGroup struct {
	Name  string
	Label string
}

type TaskInput struct {
	Name        string
	Label       string
	Type        string
	Default     string
	VisibleRule string
	Group       string
	Help        string
	Required    bool
	Editable    bool
	Options     []TaskOption
	Source      *DataSource
}

type DataSource struct {
	Name           string
	EndpointInput  string
	EndpointURL    string
	Parameters     map[string]string
	ResultSelector string
	ResultTemplate string
}

var inputRef = regexp.MustCompile(`\$\(([A-Za-z0-9_.\-]+)\)`)

func (s *DataSource) Inputs() []string {
	if s == nil {
		return nil
	}
	var out []string
	add := func(text string) {
		for _, m := range inputRef.FindAllStringSubmatch(text, -1) {
			if name := strings.ToLower(m[1]); !slices.Contains(out, name) {
				out = append(out, name)
			}
		}
	}
	if s.EndpointInput != "" {
		add("$(" + s.EndpointInput + ")")
	}
	add(s.EndpointURL)
	for _, v := range s.Parameters {
		add(v)
	}
	slices.Sort(out)
	return out
}

type AzureListRequest struct {
	EndpointID string
	Source     DataSource
}

func (r AzureListRequest) Key() string {
	keys := make([]string, 0, len(r.Source.Parameters))
	for k, v := range r.Source.Parameters {
		keys = append(keys, k+"="+v)
	}
	slices.Sort(keys)
	return strings.ToLower(r.EndpointID) + "|" + r.Source.Name + "|" + r.Source.EndpointURL + "|" + strings.Join(keys, "&")
}

type AgentQueue struct {
	ID     int
	Name   string
	Hosted bool
}

type TaskOption struct {
	Value  string
	Label  string
	Detail string
}

type InputKind int

const (
	InputText InputKind = iota
	InputMultiLine
	InputBool
	InputChoice
	InputEndpoint
	InputNumber
	InputAzure
	InputQueue
	InputLines
)

func (i TaskInput) Kind() InputKind {
	t := strings.ToLower(i.Type)
	switch {
	case t == "boolean":
		return InputBool
	case t == "multiline":
		return InputMultiLine
	case t == "int":
		return InputNumber
	case strings.HasPrefix(t, "connectedservice:"):
		return InputEndpoint
	case i.Source != nil:
		return InputAzure
	case (t == "picklist" || t == "radio") && len(i.Options) > 0 && !i.Editable:
		return InputChoice
	}
	return InputText
}

func (i TaskInput) EndpointType() string {
	_, rest, ok := strings.Cut(i.Type, ":")
	if !ok {
		return ""
	}
	kind, _, _ := strings.Cut(rest, ":")
	kind, _, _ = strings.Cut(kind, ",")
	return strings.ToLower(strings.TrimSpace(kind))
}

func (i TaskInput) OptionLabel(value string) string {
	for _, o := range i.Options {
		if strings.EqualFold(o.Value, value) {
			return o.Label
		}
	}
	return value
}

type ServiceEndpoint struct {
	ID   string
	Name string
	Type string
}

type DefinitionRevision struct {
	Revision   int
	Changed    time.Time
	ChangedBy  string
	ChangeType string
	Comment    string
}

type VariableGroupInfo struct {
	Name      string
	Variables int
}

type ReleaseDocRefs struct {
	Tasks      map[string]*TaskDefinition
	Endpoints  map[string]string
	Queues     map[int]AgentQueue
	Groups     map[int]VariableGroupInfo
	AzureNames map[string]string
}

func (r ReleaseDocRefs) Task(t DocTask) *TaskDefinition {
	return r.Tasks[TaskRef{ID: t.TaskID, Major: t.Major()}.Key()]
}

const (
	FieldGroupTask    = "Task"
	FieldGroupControl = "Control options"
)

type TaskField struct {
	Key          string
	Label        string
	Group        string
	Help         string
	Kind         InputKind
	Options      []TaskOption
	EndpointType string
	Required     bool
	Path         DocPath
	Original     any
	Value        any
	Default      string
	Guard        DocPath
	GuardValue   any
	Secret       bool
	ReadOnly     bool
	Source       *DataSource
}

func (f TaskField) FromAzure() bool { return f.Kind == InputAzure || f.Kind == InputEndpoint }

func (f TaskField) DependsOn(key string) bool {
	return slices.Contains(f.Source.Inputs(), strings.ToLower(key))
}

func (f TaskField) Changed() bool { return !sameValue(f.Original, f.Value) }

func (f TaskField) Text() string {
	text := FormatDocValue(f.Value)
	if f.Value == nil && f.Default != "" {
		text = f.Default
	}
	switch f.Kind {
	case InputBool:
		if Truthy(f.Value) || (f.Value == nil && Truthy(f.Default)) {
			return "Yes"
		}
		return "No"
	case InputChoice:
		for _, o := range f.Options {
			if strings.EqualFold(o.Value, text) {
				return o.Label
			}
		}
	case InputLines:
		var parts []string
		for _, v := range asList(f.Value) {
			parts = append(parts, scalar(v))
		}
		return strings.Join(parts, "; ")
	}
	return text
}

func asList(v any) []any {
	list, _ := v.([]any)
	return list
}

func (f TaskField) EditText() string {
	if f.Kind == InputLines {
		var parts []string
		for _, v := range asList(f.Value) {
			parts = append(parts, scalar(v))
		}
		return strings.Join(parts, "\n")
	}
	if f.Value == nil {
		return f.Default
	}
	return FormatDocValue(f.Value)
}

func (f TaskField) Edit(where string, value any) DocEdit {
	after := f
	after.Value = value
	before := f
	before.Value = f.Original
	return DocEdit{
		Path: f.Path, Where: where, Field: f.Label, Before: f.Original, After: value, Secret: f.Secret, Kind: f.Kind,
		BeforeText: before.shown(), AfterText: after.shown(), Guard: f.Guard, GuardValue: f.GuardValue,
	}
}

func (f TaskField) shown() string {
	if f.Secret {
		if sameValue(f.Value, f.Original) {
			return "******** secret"
		}
		return "******** new value"
	}
	if text := f.Text(); text != "" {
		return text
	}
	return "empty"
}

func (f TaskField) ToggledValue() any {
	switch {
	case f.Value != nil:
		return Toggled(f.Value)
	case strings.HasPrefix(f.Key, "$"):
		return true
	}
	return strconv.FormatBool(!Truthy(f.Default))
}

func (f TaskField) ruleValue() string {
	if f.Value == nil {
		return f.Default
	}
	return scalar(f.Value)
}

func (d *ReleaseDoc) TaskFields(stage DocStage, phase DocPhase, task DocTask, def *TaskDefinition, edits DocEdits) ([]TaskField, int) {
	base := d.TaskPath(stage, phase, task)
	guard := base.Child("taskId")
	field := func(key, label, group string, kind InputKind, path DocPath) TaskField {
		original := d.ValueAt(path)
		return TaskField{
			Key: key, Label: label, Group: group, Kind: kind, Path: path, Original: original,
			Value: edits.Value(path, original), Guard: guard, GuardValue: task.TaskID,
		}
	}
	fields := []TaskField{field("$name", "Display name", FieldGroupTask, InputText, base.Child("name"))}
	fields[0].Required = true
	hidden := 0
	if def != nil {
		values := map[string]string{}
		var inputs []TaskField
		for _, in := range def.Inputs {
			f := field(in.Name, in.Label, groupLabel(def, in.Group), in.Kind(), base.Child("inputs", in.Name))
			f.Help, f.Options, f.Required, f.Default = in.Help, in.Options, in.Required, in.Default
			f.EndpointType, f.Source = in.EndpointType(), in.Source
			if f.Label == "" {
				f.Label = in.Name
			}
			values[strings.ToLower(in.Name)] = f.ruleValue()
			inputs = append(inputs, f)
		}
		for i, in := range def.Inputs {
			if !InputVisible(in.VisibleRule, values) {
				hidden++
				continue
			}
			fields = append(fields, inputs[i])
		}
		fields = groupFields(fields, def)
	} else {
		names := make([]string, 0, len(task.Inputs))
		for name := range task.Inputs {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			fields = append(fields, field(name, name, FieldGroupTask, InputText, base.Child("inputs", name)))
		}
	}
	fields = append(fields,
		field("$enabled", "Enabled", FieldGroupControl, InputBool, base.Child("enabled")),
		field("$continueOnError", "Continue on error", FieldGroupControl, InputBool, base.Child("continueOnError")),
		field("$alwaysRun", "Always run", FieldGroupControl, InputBool, base.Child("alwaysRun")),
		withDefault(field("$timeout", "Timeout (minutes)", FieldGroupControl, InputNumber, base.Child("timeoutInMinutes")), "0"),
		withDefault(field("$condition", "Run this task when", FieldGroupControl, InputText, base.Child("condition")), "succeeded()"),
	)
	return fields, hidden
}

func withDefault(f TaskField, value string) TaskField {
	f.Default = value
	return f
}

func groupLabel(def *TaskDefinition, name string) string {
	if name == "" {
		return FieldGroupTask
	}
	for _, g := range def.Groups {
		if strings.EqualFold(g.Name, name) {
			if g.Label != "" {
				return g.Label
			}
			return g.Name
		}
	}
	return name
}

func groupFields(fields []TaskField, def *TaskDefinition) []TaskField {
	order := []string{FieldGroupTask}
	for _, g := range def.Groups {
		order = append(order, groupLabel(def, g.Name))
	}
	rank := func(group string) int {
		if i := slices.Index(order, group); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(fields, func(a, b TaskField) int { return rank(a.Group) - rank(b.Group) })
	return fields
}

func (d *ReleaseDoc) VariableField(v DocVariable, edits DocEdits) TaskField {
	path := d.VariablePath(v)
	original := d.ValueAt(path)
	f := TaskField{
		Key: v.Name, Label: v.Name, Group: v.Scope, Kind: InputText, Path: path, Original: original,
		Value: edits.Value(path, original), Secret: v.Secret,
	}
	if v.Stage >= 0 {
		f.Guard, f.GuardValue = DocPath{"environments", v.Stage, "name"}, v.Scope
	}
	return f
}

var ruleClause = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*(!=|=|\bnotContains\b|\bcontains\b|\bnotEndsWith\b|\bendsWith\b|\bstartsWith\b|>|<)\s*(.*?)\s*$`)

func InputVisible(rule string, values map[string]string) bool {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return true
	}
	for alt := range strings.SplitSeq(rule, "||") {
		all := true
		for clause := range strings.SplitSeq(alt, "&&") {
			if !clauseHolds(clause, values) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func clauseHolds(clause string, values map[string]string) bool {
	m := ruleClause.FindStringSubmatch(clause)
	if m == nil {
		return true
	}
	got := strings.ToLower(values[strings.ToLower(m[1])])
	want := strings.ToLower(strings.Trim(m[3], `"'`))
	switch m[2] {
	case "=":
		return got == want
	case "!=":
		return got != want
	case "contains":
		return strings.Contains(got, want)
	case "notContains":
		return !strings.Contains(got, want)
	case "endsWith":
		return strings.HasSuffix(got, want)
	case "notEndsWith":
		return !strings.HasSuffix(got, want)
	case "startsWith":
		return strings.HasPrefix(got, want)
	case ">", "<":
		a, errA := strconv.ParseFloat(got, 64)
		b, errB := strconv.ParseFloat(want, 64)
		if errA != nil || errB != nil {
			return true
		}
		return (m[2] == ">" && a > b) || (m[2] == "<" && a < b)
	}
	return true
}

func TaskYAML(task DocTask, def *TaskDefinition, fields []TaskField) string {
	ref := fmt.Sprintf("%s@%d", task.TaskID, task.Major())
	if def != nil && def.Name != "" {
		ref = fmt.Sprintf("%s@%d", def.Name, def.Major)
	}
	var b strings.Builder
	b.WriteString("- task: " + ref + "\n")
	var inputs []TaskField
	for _, f := range fields {
		switch f.Key {
		case "$name":
			b.WriteString("  displayName: " + yamlScalar(FormatDocValue(f.Value), "  ") + "\n")
		case "$enabled":
			if !Truthy(f.Value) {
				b.WriteString("  enabled: false\n")
			}
		case "$continueOnError", "$alwaysRun":
			if Truthy(f.Value) {
				b.WriteString("  " + strings.TrimPrefix(f.Key, "$") + ": true\n")
			}
		case "$timeout":
			if v := FormatDocValue(f.Value); v != "" && v != "0" {
				b.WriteString("  timeoutInMinutes: " + v + "\n")
			}
		case "$condition":
			if v := FormatDocValue(f.Value); v != "" && !strings.EqualFold(v, "succeeded()") {
				b.WriteString("  condition: " + yamlScalar(v, "  ") + "\n")
			}
		default:
			if FormatDocValue(f.Value) != "" {
				inputs = append(inputs, f)
			}
		}
	}
	if len(inputs) > 0 {
		b.WriteString("  inputs:\n")
		for _, f := range inputs {
			b.WriteString("    " + f.Key + ": " + yamlScalar(FormatDocValue(f.Value), "      ") + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

var yamlPlain = regexp.MustCompile(`^[A-Za-z0-9_./$()\-][A-Za-z0-9 _./$()\-]*$`)

func yamlScalar(v, indent string) string {
	if strings.Contains(v, "\n") {
		lines := strings.Split(strings.TrimRight(v, "\n"), "\n")
		return "|\n" + indent + strings.Join(lines, "\n"+indent)
	}
	lower := strings.ToLower(v)
	_, numErr := strconv.ParseFloat(v, 64)
	if yamlPlain.MatchString(v) && !strings.HasSuffix(v, " ") && numErr != nil &&
		!slices.Contains([]string{"true", "false", "yes", "no", "null", "on", "off", "~"}, lower) {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func FieldValue(kind InputKind, previous any, text string) (any, error) {
	switch kind {
	case InputLines:
		out := []any{}
		for line := range strings.SplitSeq(text, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				out = append(out, line)
			}
		}
		return out, nil
	case InputQueue:
		id, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("choose an agent pool")
		}
		return json.Number(strconv.Itoa(id)), nil
	case InputNumber:
		text = strings.TrimSpace(text)
		if text == "" {
			text = "0"
		}
		if _, err := strconv.Atoi(text); err != nil {
			return nil, fmt.Errorf("enter a whole number")
		}
		if _, isString := previous.(string); isString {
			return text, nil
		}
		return json.Number(text), nil
	}
	return text, nil
}
