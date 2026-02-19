package styles

// ── Terminal size breakpoints ─────────────────────────────────────────────
// Below MinTerminalWidth / MinTerminalHeight the app degrades to minimal view
// (work-items only + resize hint). Analogous to Tailwind's "sm" breakpoint.
const (
	MinTerminalWidth  = 110
	MinTerminalHeight = 25
)

// FilterShowWidth is the minimum terminal width at which the filter panel is
// displayed. Below this threshold the filter is hidden to leave the work-items
// panel enough horizontal space (≥43 inner chars).
const FilterShowWidth = 75

// CommentsShowWidth is the minimum content-area width at which the comments
// panel is rendered alongside the details panel.
const CommentsShowWidth = 55

// ── Filter panel dimensions ───────────────────────────────────────────────
const (
	FilterMinWidth = 18
	FilterMaxWidth = 28
)

// ── Panel height minimums ─────────────────────────────────────────────────
const (
	MinWorkItemsHeight = 8
	MinBottomRowHeight = 7
	MinBottomRowWidth  = 40
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
	DetailLabelWidth          = 10  // field label column in sidebar
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

// ── Work items panel ──────────────────────────────────────────────────────
const (
	WorkItemsHeaderHeight  = 7  // rows consumed by title/blank/header/separator/borders
	WorkItemsContentOffset = 6  // horizontal space: border(2) + outer-pad(2) + inner-pad(2)
	WorkItemsMinFlexWidth  = 20 // minimum width for flex columns
	WorkItemsMinContent    = 10 // minimum separator/content width
)

// ── Filter panel ─────────────────────────────────────────────────────────
const (
	FilterGroupVisibleOptions = 6  // max visible options per filter group
	FilterSepMaxWidth         = 15 // max separator line length
	FilterSepPadOffset        = 4  // horizontal indent: f.width - FilterSepPadOffset
	FilterMinViewportWidth    = 10 // minimum viewport width guard
)

// ── App layout ────────────────────────────────────────────────────────────
const (
	AppHeaderFooterSize   = 2    // header(1) + statusBar(1)
	FilterPanelWidthRatio = 0.20 // filter panel = 20% of screen width
	WorkItemsHeightRatio  = 0.55 // work items = 55% of content height
)

// ── Help panel ────────────────────────────────────────────────────────────
const (
	HelpKeyColumnWidth = 12
	HelpPanelWidth     = 40
	HelpPanelHeight    = 25
)

// ── Details panel metadata columns ───────────────────────────────────────
const (
	DetailMetaLabelWidth = 10 // label column width in metadata rows
	DetailMetaValueWidth = 12 // value column width in metadata rows
)

// ── Assignee list in modals ───────────────────────────────────────────────
const AssigneeListVisibleItems = 4 // pbi_modal, parent_work_item_modal inline list
