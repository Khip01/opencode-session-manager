package tui

import (
	"strings"

	"github.com/Khip01/opencode-session-manager/internal/db"
)

type modeID int

const (
	modeNone modeID = iota
	modeRelinkChoice
	modeFilepicker
	modeConfirm
	modeRunningWarn
	modeResult
)

func (m modeID) Label() string {
	switch m {
	case modeNone:
		return "list"
	case modeRelinkChoice:
		return "relink-choice"
	case modeFilepicker:
		return "filepicker"
	case modeConfirm:
		return "confirm"
	case modeRunningWarn:
		return "running-warn"
	case modeResult:
		return "result"
	default:
		return "unknown"
	}
}

type relinkStrategy int

const (
	strategyPhase1 relinkStrategy = iota
	strategyManual
	strategyMigrate
)

func (s relinkStrategy) Label() string {
	switch s {
	case strategyPhase1:
		return "Phase 1 (project_id)"
	case strategyManual:
		return "Manual path remap"
	case strategyMigrate:
		return "Cross-project migrate"
	default:
		return "Unknown"
	}
}

type pendingRelink struct {
	sessionID    string
	sessionTitle string
	sessionKind  itemKind
	oldDirectory string

	strategy    relinkStrategy
	newDirectory string
	migrateCount int

	phase1Match *relinkerMatch

	runningProcs []RunningProcess
}

type relinkerMatch struct {
	sessionID string
	oldDir    string
	newDir    string
}

type resultKind int

const (
	resultOK resultKind = iota
	resultWarn
	resultErr
)

type modalState struct {
	prompt        string
	warnProcs     []RunningProcess
	resultMsg     string
	resultKind    resultKind

	choiceIdx     int
	choiceOptions []string

	pending *pendingRelink

	migratePreview []db.Session
}

func newModalState() modalState {
	return modalState{}
}

func (m modalState) hasPending() bool {
	return m.pending != nil
}

func (m *modalState) clear() {
	*m = newModalState()
}

func (m modalState) renderChoice() string {
	var b strings.Builder
	b.WriteString(m.prompt)
	b.WriteString("\n\n")
	for i, opt := range m.choiceOptions {
		cursor := "  "
		if i == m.choiceIdx {
			cursor = "▸ "
		}
		b.WriteString(cursor)
		b.WriteString(opt)
		b.WriteString("\n")
	}
	b.WriteString("\n[↑/↓ to choose, enter to confirm, esc to cancel]")
	return b.String()
}
