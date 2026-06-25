package tui

import (
	"charm.land/bubbles/v2/filepicker"
)

func newDirPicker() filepicker.Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{}
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.SetHeight(16)
	return fp
}
