package styles

const (
	TopBarHeight         = 1
	StatusBarHeight      = 1
	StatusBarGaps        = 3
	HeaderFullWidth      = 90
	HeaderCompactWidth   = 70
	PageHeaderHeight     = 3
	RailWidth            = 28
	RailShowWidth        = 100
	RailCompactWidth     = 5
	RailCompactShowWidth = 70
	RightPaneShowWidth   = 140
	RightPaneMinWidth    = 40
	RightPaneMaxWidth    = 64
	RightPaneWidthRatio  = 0.32
	ShellMinWidth        = 50
	ShellMinHeight       = 14
	BoxPaddedSize        = 4
	PaneChromeHeight     = 4
	PaneContentInsetX    = 2
	PaneContentInsetY    = 3
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
	DetailFieldLabelW  = 11
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
	TaskCardGap          = 1
	TaskParentHeight     = 2
	TaskCollapsedHeight  = 1
	TaskCellPadding      = 2
	TaskTableHeaderLines = 3
	TaskStateMinWidth    = 18
	TaskLaneMinWidth     = 22
	TaskLaneMaxWidth     = 44
	TaskLaneWidthRatio   = 0.30
	DragChipWidth        = 40
)

// ── Panel height minimums ─────────────────────────────────────────────────
const (
	MinBottomRowWidth = 40
)

const (
	ModalWidthSM  = 44
	ModalWidthMD  = 50
	ModalWidthLG  = 60
	ModalWidthXL  = 70
	ModalWidthXXL = 80
)

const (
	TextareaHeightSM = 3
	TextareaHeightMD = 4
	TextareaHeightLG = 6
)

const (
	InputCharLimitSM   = 10
	TitleCharLimit     = 255
	ContentCharLimitLG = 2000
	PriorityCharLimit  = 1
	EffortCharLimit    = 5
	TagsCharLimit      = 500
)

const (
	PickerRows            = 8
	FormPickerRows        = 5
	MentionPickerRows     = 5
	FormCompactFieldWidth = 22
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
	DetailReadingWidth        = 100
	DetailBandValueW          = 36
	DetailFormLabelW          = 13
	DetailColumnGap           = 3
	DetailHeaderLines         = 8
	DetailHeaderLinesStacked  = 10
	CommentIndent             = 3
)

const (
	KeywordInputWidth  = 24
	FilterPickerWidth  = 44
	FilterPickerRows   = 12
	FilterPickerChrome = 7
)
