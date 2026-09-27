package ui

import "github.com/rivo/tview"

const helpText = `[::b]a1s keybindings[-:-:-]

[::b]Navigation[-:-:-]
  j / down       move down
  k / up         move up
  g              jump to top
  G              jump to bottom

[::b]Filtering[-:-:-]
  /              open filter (plain text, or field=value e.g. status=running)
  Esc            clear filter / close overlay

[::b]Profiles[-:-:-]
  Ctrl-P         switch aliyun-cli profile/region

[::b]Actions[-:-:-]
  s              start the selected instance
  S              stop the selected instance normally (asks to confirm)
  x              economic stop: pauses vCPU/memory billing while stopped (asks to confirm)
  R              reboot the selected instance (asks to confirm)
  D              terminate the selected instance (asks to confirm, irreversible)
  E              run a shell command on the selected instance, no SSH
                 (via Alibaba Cloud Cloud Assistant)

[::b]Command palette[-:-:-]
  :              open the command palette for the long tail of a1s commands:
                   bill [cycle], doctor, report [file], query <term>,
                   metrics [minutes], currency USD|SAR, theme alibaba|mono,
                   profile <name>, region <id>, regions, configure [name]

[::b]General[-:-:-]
  Ctrl-R         refresh now (re-fetch ECS inventory)
                 (also auto-refreshes on its own)
  ?              toggle this help
  q / Ctrl-C     quit

Press Esc or ? to close this screen.`

// newHelpView builds the "?" help modal.
func newHelpView() *tview.TextView {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText)
	view.SetBorder(true)
	return view
}
