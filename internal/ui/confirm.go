package ui

import "github.com/rivo/tview"

// newConfirmModal builds a Yes/Cancel confirmation dialog. onYes is called
// when the user confirms; onCancel when they back out (Cancel, Esc, or the
// modal otherwise closing without confirmation).
func newConfirmModal(text string, onYes func(), onCancel func()) *tview.Modal {
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Yes", "Cancel"})

	modal.SetDoneFunc(func(_ int, label string) {
		if label == "Yes" {
			onYes()
			return
		}
		onCancel()
	})

	return modal
}
