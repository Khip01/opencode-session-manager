package tui

import "context"

func (m *model) openRelinkChoice() {
	si, ok := selectedItem(m.list)
	if !ok {
		return
	}

	m.modal.prompt = "How do you want to relink this session?\n\n" +
		"Session:    " + si.session.ID + "\n" +
		"Title:      " + si.session.Title + "\n" +
		"Old path:   " + si.session.Directory

	m.modal.choiceOptions = []string{
		"Phase 1 (auto, project_id match)",
		"Manual path remap",
		"Cancel",
	}
	m.modal.choiceIdx = 0

	m.modal.pending = &pendingRelink{
		sessionID:    si.session.ID,
		sessionTitle: si.session.Title,
		sessionKind:  si.kind,
		oldDirectory: si.session.Directory,
	}
	m.mode = modeRelinkChoice
}

func (m *model) confirmChoice() {
	if m.modal.pending == nil {
		m.mode = modeNone
		m.modal.clear()
		return
	}

	switch m.modal.choiceIdx {
	case 0:
		m.modal.pending.strategy = strategyPhase1
		m.startPhase1()
	case 1:
		m.modal.pending.strategy = strategyManual
		m.startManualPath()
	default:
		m.mode = modeNone
		m.modal.clear()
	}
}

func (m *model) startPhase1() {
	if m.relinker == nil {
		m.showResult("internal error: relinker not set", resultErr)
		return
	}

	ctx := context.Background()
	matches, err := m.relinker.FindPhase1Matches(ctx)
	if err != nil {
		m.showResult("Phase 1 lookup failed: "+err.Error(), resultErr)
		return
	}

	var found *relinkerMatch
	for i := range matches {
		if matches[i].SessionID == m.modal.pending.sessionID {
			found = &relinkerMatch{
				sessionID: matches[i].SessionID,
				oldDir:    matches[i].OldDirectory,
				newDir:    matches[i].NewDirectory,
			}
			break
		}
	}

	if found == nil {
		m.showResult(
			"Phase 1 could not match this session — its project_id is missing or unknown.\n\nUse Manual path remap instead.",
			resultWarn,
		)
		return
	}

	m.modal.pending.phase1Match = found
	m.modal.pending.newDirectory = found.newDir
	m.proceedWithRunningCheck()
}

func (m *model) startManualPath() {
	m.mode = modeFilepicker
	m.picker = newDirPicker()
	if m.modal.pending.oldDirectory != "" {
		m.picker.CurrentDirectory = m.modal.pending.oldDirectory
	}
}

func (m *model) proceedWithRunningCheck() {
	procs, err := DetectRunningInstances()
	if err != nil || len(procs) == 0 {
		m.proceedToConfirm()
		return
	}
	m.modal.warnProcs = procs
	m.modal.prompt = "OpenCode is running. Stop the listed instances before continuing?\n\n" +
		"Relinking a session while opencode is active may cause inconsistent state."
	m.mode = modeRunningWarn
}

func (m *model) proceedToConfirm() {
	p := m.modal.pending
	if p == nil {
		m.mode = modeNone
		m.modal.clear()
		return
	}
	m.modal.prompt = "Apply this relink?\n\n" +
		"Session:    " + truncateModal(p.sessionID, 30) + "\n" +
		"Old path:   " + truncateModal(p.oldDirectory, 60) + "\n" +
		"New path:   " + truncateModal(p.newDirectory, 60) + "\n" +
		"Strategy:   " + p.strategy.Label() + "\n\n" +
		"A backup of opencode.db will be created before any write."
	m.mode = modeConfirm
}

func (m *model) killRunningAndProceed() {
	procs := m.modal.warnProcs
	results := []string{}
	for _, p := range procs {
		if err := killProcess(p.PID); err != nil {
			results = append(results, "PID "+itoa(p.PID)+": "+err.Error())
		} else {
			results = append(results, "PID "+itoa(p.PID)+": killed")
		}
	}
	m.status = m.styles.statusWarn.Render("killed running instances: " + joinComma(results))
	m.proceedToConfirm()
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
