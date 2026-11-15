package styles

const (
	TopBarHeight        = 1
	StatusBarHeight     = 1
	StatusBarGaps       = 3
	HeaderFullWidth     = 90
	HeaderCompactWidth  = 70
	PageHeaderHeight    = 3
	RailWidth           = 28
	RailShowWidth       = 100
	RightPaneShowWidth  = 140
	RightPaneMinWidth   = 40
	RightPaneMaxWidth   = 64
	RightPaneWidthRatio = 0.32
	ShellMinWidth       = 50
	ShellMinHeight      = 14
	BoxPaddedSize       = 4
	PaneChromeHeight    = 4
)

const (
	GridColTree        = 1
	GridColType        = 8
	GridCellPadding    = 2
	GridColID          = 6
	GridColState       = 14
	GridColAssigned    = 16
	GridColEffort      = 6
	GridColIteration   = 12
	GridColTags        = 14
	GridColTitleMin    = 36
	GridColTitleFloor  = 20
	GridTitleRatio     = 0.4
	RailFilterOptions  = 8
	RailFieldMinWidth  = 12
	RailFieldInset     = 2
	RailDropdownChrome = 5
	RailHubInset       = 3
	RailHubListHeight  = 5
	DetailFieldLabelW  = 11
	SectionBodyOffset  = 5
	DetailProgressBarW = 24
	ChildListChrome    = 3
)

const (
	BoardCardHeight      = 4
	CardChromeWidth      = 2
	BoardCardGap         = 1
	BoardColumnMinWidth  = 24
	BoardColumnGap       = 1
	TaskCardHeight       = 2
	TaskParentHeight     = 2
	TaskCellPadding      = 2
	TaskTableHeaderLines = 3
	TaskStateMinWidth    = 18
	TaskLaneMinWidth     = 22
	TaskLaneMaxWidth     = 44
	TaskLaneWidthRatio   = 0.30
)

// ── Panel height minimums ─────────────────────────────────────────────────
const (
	MinBottomRowWidth = 40
)

// ── Modal width tiers ─────────────────────────────────────────────────────
// Use these instead of raw integers in every modal View() method.
const (
	ModalWidthSM  = 40 // info, delete, simple confirmations
	ModalWidthMD  = 50 // assign, branch, quick search
	ModalWidthLG  = 60 // task
	ModalWidthXL  = 70 // pbi, parent work item, comment
	ModalWidthXXL = 80 // error modal
)

// ── Modal height tiers ────────────────────────────────────────────────────
const (
	ModalHeightSM  = 8  // delete
	ModalHeightMD  = 10 // info, branch
	ModalHeightLG  = 12 // quick search
	ModalHeightXL  = 20 // search, error, task (initial)
	ModalHeightXXL = 35 // parent_work_item_modal (many fields)
)

// ── Modal padding ─────────────────────────────────────────────────────────
const (
	ModalPaddingV = 1 // Padding(1, ...)
	ModalPaddingH = 2 // Padding(..., 2)
)

// ── Modal content offset ──────────────────────────────────────────────────
// Total horizontal space consumed by border + Padding(ModalPaddingV, ModalPaddingH):
// 2 border + 2*ModalPaddingH padding = 6. Used for inner width calculations.
const ModalContentOffset = 6

// ── Modal dynamic base heights ────────────────────────────────────────────
// Added to dynamic content counts (e.g. len(states) + StateModalBaseHeight).
const (
	StateModalBaseHeight  = 6  // state_modal: title + instructions + borders
	AssignModalBaseHeight = 10 // assign_modal: title + search + instructions + borders
)

// ── Modal scrollable list ─────────────────────────────────────────────────
const (
	ModalListVisibleItems = 8  // assign_modal, search_modal scroll window
	ModalListItemOffset   = 10 // name truncation offset: border(2)+padding(4)+cursor(2)+margin(2)
)

// ── Textarea dimensions ───────────────────────────────────────────────────
const (
	TextareaWidthMD  = 50 // task_modal, pbi_modal
	TextareaWidthLG  = 60 // comment_modal, parent_work_item_modal
	TextareaHeightSM = 3  // most modal textareas
	TextareaHeightMD = 4  // parent_work_item description
	TextareaHeightLG = 6  // comment_modal
)

// ── Text input dimensions ─────────────────────────────────────────────────
const (
	TextInputWidthSM = 15 // small number inputs (priority, effort in parent_work_item_modal)
	TextInputWidthMD = 30 // assign_modal
	TextInputWidthLG = 35 // branch_modal
)

// ── Input character limits ────────────────────────────────────────────────
const (
	InputCharLimitSM = 10  // quick_search_modal (numeric ID)
	InputCharLimitMD = 50  // search_modal, assign_modal
	InputCharLimitLG = 100 // branch_modal
	TitleCharLimit   = 255 // work item title field
)

// ── Content character limits ──────────────────────────────────────────────
const (
	ContentCharLimitMD = 1000 // single-field description (task)
	ContentCharLimitLG = 2000 // comment / rich text fields
)

// ── Form field specifics (pbi_modal) ─────────────────────────────────────
const (
	PriorityCharLimit = 1   // priority field: single digit
	EffortCharLimit   = 5   // effort field: e.g. "99.99"
	TagsCharLimit     = 500 // tags field
	SmallInputWidth   = 10  // small numeric inputs (priority, effort)
)

// ── Branch name limits ────────────────────────────────────────────────────
const (
	BranchNameMaxLen          = 40
	BranchNameWordBoundaryMin = 20
)

// ── Detail view ───────────────────────────────────────────────────────────
const (
	DetailSidebarWidthSM      = 30
	DetailSidebarWidthDefault = 36
	DetailSidebarWidthLG      = 44
	DetailBreakpointSM        = 100 // innerWidth < this → narrow sidebar
	DetailBreakpointLG        = 160 // innerWidth > this → wide sidebar
	DetailMinViewportWidth    = 20
	DetailMinViewportHeight   = 5
	DetailWrapMinWidth        = 10 // minimum text-wrap width in sections
)

// ── Panel border / padding offsets ────────────────────────────────────────
// PanelBorderOffset: 1px border each side = 2 total (one axis).
// PanelPaddedOffset: border(2) + Padding(0,1) both sides = 4 total.
const (
	PanelBorderOffset = 2
	PanelPaddedOffset = 4
)

// ── Filter panel ─────────────────────────────────────────────────────────
const (
	FilterGroupVisibleOptions = 6 // max visible options per filter group
)

// ── Assignee list in modals ───────────────────────────────────────────────
const AssigneeListVisibleItems = 4 // pbi_modal, parent_work_item_modal inline list
