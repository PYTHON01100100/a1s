package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

const keyHints = "/ filter   ctrl-p profile   ctrl-r refresh   s start   S stop   x eco-stop   R reboot   D terminate   E run cmd   : palette   ? help   q quit"

// Footer renders the status bar as a row of e1s/ec2s-style colored chips
// (profile/region scope, active filter, key hints, counts, currency, app
// version), plus a second line for transient notices or account warnings.
type Footer struct {
	view *tview.Flex

	chips    *tview.Flex
	scope    *tview.TextView
	filter   *tview.TextView
	hints    *tview.TextView
	counts   *tview.TextView
	currency *tview.TextView
	app      *tview.TextView
	notice   *tview.TextView

	appChipWidth int
}

func newFooter(version string) *Footer {
	f := &Footer{
		chips:    tview.NewFlex().SetDirection(tview.FlexColumn),
		scope:    tview.NewTextView().SetDynamicColors(true),
		filter:   tview.NewTextView().SetDynamicColors(true),
		hints:    tview.NewTextView().SetDynamicColors(true).SetText("[green]" + keyHints + "[-]"),
		counts:   tview.NewTextView().SetDynamicColors(true),
		currency: tview.NewTextView().SetDynamicColors(true),
		app:      tview.NewTextView().SetDynamicColors(true),
		notice:   tview.NewTextView().SetDynamicColors(true),
	}

	appText := fmt.Sprintf("a1s:%s", version)
	f.app.SetText(fmt.Sprintf(footerAppFmt, "a1s", version))
	f.appChipWidth = len(appText) + 3

	f.view = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(f.chips, 1, 0, false).
		AddItem(f.notice, 1, 0, false)

	return f
}

// Status is the current, non-transient state the footer's chip row is
// built from.
type Status struct {
	Profile  string
	Region   string
	Currency string
	Mode     string // "LIVE", "SAMPLE DATA", or "UI ONLY"
	ReadOnly bool
	Filter   string
	Loaded   bool // false until the first fetch completes
	Total    int
	Warning  string // account/CLI problem, if any
}

// SetStatus rebuilds the chip row and notice line from the current Status.
func (f *Footer) SetStatus(st Status) {
	scope := blank(st.Profile, "default") + "/" + blank(st.Region, "auto")
	f.scope.SetText(fmt.Sprintf(footerChipActiveFmt, scope))

	f.chips.Clear()
	f.chips.AddItem(f.scope, len(scope)+3, 0, false)

	if st.Filter != "" {
		filterLabel := "filter: " + tview.Escape(st.Filter)
		f.filter.SetText(fmt.Sprintf(footerChipFmt, filterLabel))
		f.chips.AddItem(f.filter, len(filterLabel)+3, 0, false)
	}

	f.chips.AddItem(f.hints, 0, 1, false)

	access := "rw"
	if st.ReadOnly {
		access = "read-only"
	}
	countsLabel := fmt.Sprintf("%s · %s · loading…", st.Mode, access)
	if st.Loaded {
		countsLabel = fmt.Sprintf("%s · %s · %d instances", st.Mode, access, st.Total)
	}
	f.counts.SetText(fmt.Sprintf(footerChipFmt, countsLabel))
	f.chips.AddItem(f.counts, len(countsLabel)+3, 0, false)

	curLabel := st.Currency
	f.currency.SetText(fmt.Sprintf(footerChipFmt, curLabel))
	f.chips.AddItem(f.currency, len(curLabel)+3, 0, false)

	f.chips.AddItem(f.app, f.appChipWidth, 0, false)

	if st.Warning != "" {
		f.notice.SetText("[" + orange + "]⚠ " + strings.TrimSpace(st.Warning) + "[-]")
	} else {
		f.notice.SetText("")
	}
}

// SetNotice overrides the notice line with a transient action message
// (e.g. the result of a stop/terminate). The caller is responsible for
// restoring the normal line afterwards, typically by calling SetStatus
// again once the notice has been shown for a while.
func (f *Footer) SetNotice(text string) {
	f.notice.SetText(text)
}
