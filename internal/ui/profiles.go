package ui

import (
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/config"
)

// newProfileSelectList builds the Ctrl-P profile-switch overlay from the
// aliyun-cli profiles found on disk. onSelect receives the chosen profile's
// name and region.
func newProfileSelectList(profiles []config.Profile, current string, onSelect func(p config.Profile), onCancel func()) *tview.List {
	list := tview.NewList().
		ShowSecondaryText(true)

	for _, p := range profiles {
		p := p
		marker := ""
		if p.Name == current {
			marker = "  (current)"
		}
		list.AddItem(p.Name+marker, "region: "+blank(p.RegionID, "auto"), 0, func() {
			onSelect(p)
		})
	}
	if len(profiles) == 0 {
		list.AddItem("No profiles found", "run `aliyun configure` or the : configure palette command", 0, nil)
	}

	list.SetDoneFunc(onCancel)

	return list
}

// profileListHeight sizes the Ctrl-P overlay to fit every profile plus the
// list's own border and title.
func profileListHeight(n int) int {
	if n < 1 {
		n = 1
	}
	return n + 3
}
