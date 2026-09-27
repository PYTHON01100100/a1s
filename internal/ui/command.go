package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/model"
)

// newCommandInput builds the "E" run-command input field. onSubmit is
// called with the entered command on Enter (and only then — unlike the
// filter box, a command shouldn't fire on every keystroke); onCancel on
// Esc or an empty submit.
func newCommandInput(instName string, onSubmit func(command string), onCancel func()) *tview.InputField {
	field := tview.NewInputField().
		SetLabel(fmt.Sprintf("Run command on %s (no SSH, via Cloud Assistant): ", instName)).
		SetFieldWidth(0)

	field.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := field.GetText()
			if text != "" {
				onSubmit(text)
				return
			}
		}
		onCancel()
	})

	return field
}

// commandResultText formats a RunCommand result for the output pane.
func commandResultText(instName, command string, r model.CommandResult) string {
	statusColor := "green"
	if r.ExitCode != 0 || r.Status != "Finished" {
		statusColor = "red"
	}
	text := fmt.Sprintf("[%s::b]%s[-:-:-] on [%s::b]%s[-:-:-]\n[::b]Status:[-:-:-] [%s]%s[-] (exit=%d)",
		orange, tview.Escape(command), accent, tview.Escape(instName), statusColor, r.Status, r.ExitCode)
	if r.InvokeID != "" {
		text += fmt.Sprintf("\n[::b]Invoke ID:[-:-:-] %s", r.InvokeID)
	}
	text += "\n\n[green::b]output[-:-:-]\n" + indentOutput(r.Output, "")
	return text
}

// commandErrorText formats a failure that prevented the command from
// running at all (e.g. Cloud Assistant unreachable, or --read-only).
func commandErrorText(instName, command string, err error) string {
	return fmt.Sprintf("[%s::b]%s[-:-:-] on [%s::b]%s[-:-:-]\n[::b]Status:[-:-:-] [red]Failed to run[-]\n\n%s",
		orange, tview.Escape(command), accent, tview.Escape(instName), tview.Escape(err.Error()))
}

// newResultView builds the scrollable output pane shared by the "E" run
// command and the ":" command palette.
func newResultView(title string) *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	view.SetBorder(true).SetTitle(title)
	return view
}
