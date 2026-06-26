package tui

import "testing"

func TestScrollTarget_DefaultNoMouseReturnsNone(t *testing.T) {
	m := model{width: 120, height: 40}
	if got := m.scrollTarget(); got != routeNone {
		t.Errorf("with no mouse position set, scrollTarget() = %d, want routeNone", got)
	}
}

func TestScrollTarget_OverLeftListPanel(t *testing.T) {
	m := model{
		width:    120,
		height:   40,
		mouseX:   30,
		mouseY:   10,
	}
	if got := m.scrollTarget(); got != routeList {
		t.Errorf("scrollTarget() over left list area = %d, want routeList", got)
	}
}

func TestScrollTarget_OverLeftHintsPanel(t *testing.T) {
	m := model{
		width:    120,
		height:   40,
		mouseX:   30,
		mouseY:   38,
	}
	if got := m.scrollTarget(); got != routeNone {
		t.Errorf("scrollTarget() over hints area = %d, want routeNone (hints is not scrollable)", got)
	}
}

func TestScrollTarget_OverRightMetaPanel(t *testing.T) {
	m := model{
		width:    120,
		height:   40,
		mouseX:   90,
		mouseY:   3,
	}
	if got := m.scrollTarget(); got != routeNone {
		t.Errorf("scrollTarget() over right meta area = %d, want routeNone (meta is not scrollable)", got)
	}
}

func TestScrollTarget_OverRightChatPanel(t *testing.T) {
	m := model{
		width:    120,
		height:   40,
		mouseX:   90,
		mouseY:   30,
	}
	if got := m.scrollTarget(); got != routeChat {
		t.Errorf("scrollTarget() over right chat area = %d, want routeChat", got)
	}
}

func TestScrollTarget_BoundaryMouseY(t *testing.T) {
	// mouseY == 0 is in header (row 0)
	m := model{width: 120, height: 40, mouseX: 30, mouseY: 0}
	if got := m.scrollTarget(); got != routeNone {
		t.Errorf("mouseY=0 (header) should be routeNone, got %d", got)
	}

	// mouseY == height is past the terminal
	m = model{width: 120, height: 40, mouseX: 30, mouseY: 40}
	if got := m.scrollTarget(); got != routeNone {
		t.Errorf("mouseY=height should be routeNone, got %d", got)
	}
}

func TestScrollTarget_ColumnBoundaryAtMidpoint(t *testing.T) {
	// mouseX exactly at midpoint belongs to the right column
	m := model{
		width:    120,
		height:   40,
		mouseX:   60,
		mouseY:   30,
	}
	if got := m.scrollTarget(); got != routeChat {
		t.Errorf("mouseX at midpoint should be right column, got %d", got)
	}
}
