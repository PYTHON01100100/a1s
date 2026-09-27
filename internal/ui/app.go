// Package ui implements the a1s tview terminal UI: a k9s/e1s/ec2s-style
// full-screen ECS inventory table with an info panel, a live-refreshing
// footer status bar, and overlays for filtering, confirmation, running
// commands, and the long tail of a1s operations (bill/doctor/report/...)
// via a ":" command palette.
package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/model"
)

const (
	pageMain     = "main"
	pageFilter   = "filter"
	pageProfiles = "profiles"
	pageHelp     = "help"
	pageConfirm  = "confirm"
	pageCommand  = "command"
	pageResult   = "result"
)

const (
	noticeDuration      = 4 * time.Second
	autoRefreshInterval = 20 * time.Second
)

// infoPanelHeight is the info panel's fixed height: enough rows for the
// taller of the two columns (instance fields vs. keybindings), plus the top
// and bottom border.
var infoPanelHeight = max(infoItemRows, len(headerKeys)) + 2

// App is the a1s terminal UI.
type App struct {
	cfg     config.Config
	cloud   *aliyun.Client
	version string
	ctx     context.Context

	tapp  *tview.Application
	pages *tview.Pages
	root  *tview.Flex

	header *Header
	table  *Table
	footer *Footer

	all     []model.ECSInstance // last fetch, unfiltered
	filter  string
	loaded  bool // false until the first fetch completes
	warning string

	resultView *tview.TextView // the output pane for the most recent E/palette command, if any
}

// New builds the a1s UI shell around an already-configured aliyun client.
func New(cfg config.Config, cloud *aliyun.Client, version string) *App {
	setTheme("alibaba")
	applyTheme()

	a := &App{
		cfg:     cfg,
		cloud:   cloud,
		version: version,
		tapp:    tview.NewApplication(),
		pages:   tview.NewPages(),
		header:  newHeader(),
		table:   newTable(),
		footer:  newFooter(version),
	}
	a.table.SetOnSelect(a.header.SetInstance)

	a.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header.view, infoPanelHeight, 0, false).
		AddItem(a.table.view, 0, 1, true).
		AddItem(a.footer.view, 2, 0, false)

	a.pages.AddPage(pageMain, a.root, true, true)
	a.tapp.SetRoot(a.pages, true).SetFocus(a.table.view)
	a.tapp.SetInputCapture(a.handleKey)

	a.refreshFooter()

	return a
}

// Run performs the initial cloud/account check, kicks off the first fetch
// and the auto-refresh loop, and starts the tview event loop. It blocks
// until the user quits or ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.ctx = ctx

	a.checkAccount(ctx)
	// refresh() ends in a.tapp.QueueUpdateDraw, which blocks until Run()'s
	// event loop below is draining that channel. Calling it synchronously
	// here — before that loop exists — would deadlock the whole program
	// (see tview's Application.QueueUpdate); it must run on its own
	// goroutine so Run() can reach a.tapp.Run() first.
	go a.refresh()
	go a.autoRefresh(ctx)

	go func() {
		<-ctx.Done()
		a.tapp.Stop()
	}()

	return a.tapp.Run()
}

func (a *App) autoRefresh(ctx context.Context) {
	ticker := time.NewTicker(autoRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
		}
	}
}

// requireCloudData reports why cloud data/actions aren't available, or nil
// when they are.
func (a *App) requireCloudData() error {
	if a.cfg.Demo && !a.cfg.SampleData {
		return fmt.Errorf("UI-only mode has no cloud/sample data; restart without --demo, or with --sample-data for explicit samples")
	}
	if !a.cfg.SampleData && a.warning != "" {
		return fmt.Errorf("no Alibaba Cloud account is configured; use the : configure palette command (or `aliyun configure`), then Ctrl-R to retry")
	}
	return nil
}

// checkAccount verifies the aliyun CLI can be reached with a usable
// account, recording the result in a.warning so the footer/table can
// explain why data isn't available instead of just looking empty.
func (a *App) checkAccount(ctx context.Context) {
	if a.cfg.SampleData {
		a.warning = ""
		return
	}
	if a.cfg.Demo {
		a.warning = "UI-only mode: no cloud calls are made. Restart without --demo, or with --sample-data."
		return
	}
	if err := a.cloud.CheckCLI(ctx); err != nil {
		a.warning = "Alibaba Cloud account not ready: " + err.Error() + " — try the : configure palette command."
	} else {
		a.warning = ""
	}
}

// refresh re-fetches the ECS inventory (unless UI-only/not-ready) and
// updates the table/footer. Safe to call from any goroutine.
func (a *App) refresh() {
	if err := a.requireCloudData(); err != nil {
		a.tapp.QueueUpdateDraw(func() {
			a.warning = err.Error()
			a.loaded = true
			a.all = nil
			a.applyFilter()
			a.refreshFooter()
		})
		return
	}

	xs, err := a.cloud.ListInstances(a.ctx)
	a.tapp.QueueUpdateDraw(func() {
		if err != nil {
			a.warning = "Could not load ECS inventory: " + err.Error()
		} else {
			a.warning = ""
			a.all = xs
		}
		a.loaded = true
		a.applyFilter()
		a.refreshFooter()
	})
}

// applyFilter must run on the UI goroutine. It narrows a.all by a.filter
// and pushes the result into the table.
func (a *App) applyFilter() {
	filtered := filterInstances(a.all, a.filter)
	scope := blank(a.cfg.Profile, "default") + "/" + blank(a.cfg.Region, "auto")
	a.table.SetInstances(filtered, scope)
}

func (a *App) refreshFooter() {
	a.footer.SetStatus(Status{
		Profile:  a.cfg.Profile,
		Region:   a.cfg.Region,
		Currency: a.cfg.Currency,
		Mode:     a.modeLabel(),
		ReadOnly: a.cfg.ReadOnly,
		Filter:   a.filter,
		Loaded:   a.loaded,
		Total:    len(a.all),
		Warning:  a.warning,
	})
}

func (a *App) modeLabel() string {
	switch {
	case a.cfg.SampleData:
		return "SAMPLE DATA"
	case a.cfg.Demo:
		return "UI ONLY"
	default:
		return "LIVE"
	}
}

// Notify shows a transient action message (e.g. the result of a stop/
// terminate) in the footer, then restores the normal status line after a
// few seconds. Safe to call from ANY goroutine, including tview's own
// event-loop goroutine (i.e. directly from a key handler, button, or input
// field callback) — it always does its actual work, including the first
// QueueUpdateDraw, on a freshly spawned goroutine of its own. Calling
// a.tapp.QueueUpdateDraw synchronously from within a callback that's
// already running ON the event-loop goroutine deadlocks the whole program
// (that channel is only drained by that same goroutine's own select loop,
// which can't run while it's still inside the callback that's blocked
// sending to it), so Notify must never do that update inline.
func (a *App) Notify(text string, isError bool) {
	go func() {
		icon, color := "✓", "green"
		if isError {
			icon, color = "✗", "red"
		}
		a.tapp.QueueUpdateDraw(func() {
			a.footer.SetNotice(fmt.Sprintf("%s [%s]%s[-]", icon, color, tview.Escape(text)))
		})

		time.Sleep(noticeDuration)
		a.tapp.QueueUpdateDraw(a.refreshFooter)
	}()
}

func (a *App) handleKey(event *tcell.EventKey) *tcell.EventKey {
	front, _ := a.pages.GetFrontPage()

	// Ctrl-C always quits, everywhere, even while typing in the filter box
	// — the standard terminal "abort" convention. Plain q quits from any
	// page too, EXCEPT text-input overlays, where q is a normal character
	// to type (e.g. filtering for "web-01", or a run command containing a
	// "q").
	if event.Key() == tcell.KeyCtrlC {
		a.tapp.Stop()
		return nil
	}
	if event.Rune() == 'q' && front != pageFilter && front != pageCommand {
		a.tapp.Stop()
		return nil
	}

	// The remaining shortcuts only make sense on the main page; overlays
	// (filter/profiles/help/confirm/command/result) handle their own keys
	// otherwise.
	if front != pageMain {
		return event
	}

	switch event.Key() {
	case tcell.KeyCtrlR:
		go a.refresh()
		return nil
	case tcell.KeyCtrlP:
		a.showProfileSelect()
		return nil
	case tcell.KeyEsc:
		if a.filter != "" {
			a.filter = ""
			a.applyFilter()
			a.refreshFooter()
		}
		return nil
	}

	switch event.Rune() {
	case '/':
		a.showFilter()
		return nil
	case ':':
		a.showPalette()
		return nil
	case '?':
		a.showHelp()
		return nil
	case 'g':
		a.table.SelectTop()
		return nil
	case 'G':
		a.table.SelectBottom()
		return nil
	case 's':
		a.doStart()
		return nil
	case 'S':
		a.confirmStop()
		return nil
	case 'x':
		a.confirmStopEco()
		return nil
	case 'R':
		a.confirmReboot()
		return nil
	case 'D':
		a.confirmTerminate()
		return nil
	case 'E':
		a.showCommandInput()
		return nil
	}

	return event
}

// centered wraps p in a Flex that pins it to a fixed width/height in the
// middle of the screen, the standard tview idiom for modal-style overlays.
func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

func (a *App) showFilter() {
	input := newFilterInput(a.filter,
		func(query string) {
			a.filter = query
			a.applyFilter()
			a.refreshFooter()
		},
		func() {
			a.pages.RemovePage(pageFilter)
			a.tapp.SetFocus(a.table.view)
		},
	)
	input.SetBorder(true).SetTitle(" Filter ")

	a.pages.AddPage(pageFilter, centered(input, 76, 3), true, true)
	a.tapp.SetFocus(input)
}

func (a *App) showHelp() {
	view := newHelpView()

	closeHelp := func() {
		a.pages.RemovePage(pageHelp)
		a.tapp.SetFocus(a.table.view)
	}
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == '?' {
			closeHelp()
			return nil
		}
		return event
	})

	a.pages.AddPage(pageHelp, centered(view, 64, 24), true, true)
	a.tapp.SetFocus(view)
}

func (a *App) showConfirm(text string, onYes func()) {
	closeConfirm := func() {
		a.pages.RemovePage(pageConfirm)
		a.tapp.SetFocus(a.table.view)
	}

	modal := newConfirmModal(text,
		func() {
			closeConfirm()
			onYes()
		},
		closeConfirm,
	)

	a.pages.AddPage(pageConfirm, modal, true, true)
	a.tapp.SetFocus(modal)
}
