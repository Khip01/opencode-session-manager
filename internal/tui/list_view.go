package tui

import (
	"charm.land/bubbles/v2/list"
)

func newList(items []sessionItem, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(toItemList(items), delegate, width, height)
	l.Title = "Sessions"
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetShowFilter(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetStatusBarItemName("session", "sessions")

	if len(items) == 0 {
		l.Select(0)
	}
	return l
}

func refreshList(l list.Model, items []sessionItem) list.Model {
	l.SetItems(toItemList(items))
	return l
}

func toItemList(items []sessionItem) []list.Item {
	out := make([]list.Item, len(items))
	for i, it := range items {
		out[i] = it
	}
	return out
}

func selectedItem(l list.Model) (sessionItem, bool) {
	it := l.SelectedItem()
	if it == nil {
		return sessionItem{}, false
	}
	si, ok := it.(sessionItem)
	return si, ok
}
