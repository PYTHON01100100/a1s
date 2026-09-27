package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/model"
)

// doStart starts the selected instance immediately, with no confirmation —
// unlike stop/terminate, starting an instance doesn't interrupt anything or
// lose data, so asking "are you sure?" would just be friction.
func (a *App) doStart() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	go a.runLifecycle("start", inst, func() error { return a.cloud.StartInstance(a.ctx, inst.ID) })
}

func (a *App) confirmStop() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	text := fmt.Sprintf("Stop %s\n(%s)?", blank(inst.Name, inst.ID), inst.ID)
	a.showConfirm(text, func() {
		if err := a.requireCloudData(); err != nil {
			a.Notify(err.Error(), true)
			return
		}
		go a.runLifecycle("stop", inst, func() error { return a.cloud.StopInstance(a.ctx, inst.ID, aliyun.StopNormal, false) })
	})
}

func (a *App) confirmStopEco() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	text := fmt.Sprintf("Economic stop %s\n(%s)?\nPauses vCPU/memory billing while stopped.", blank(inst.Name, inst.ID), inst.ID)
	a.showConfirm(text, func() {
		if err := a.requireCloudData(); err != nil {
			a.Notify(err.Error(), true)
			return
		}
		go a.runLifecycle("eco-stop", inst, func() error { return a.cloud.StopInstance(a.ctx, inst.ID, aliyun.StopEco, false) })
	})
}

func (a *App) confirmReboot() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	text := fmt.Sprintf("Reboot %s\n(%s)?", blank(inst.Name, inst.ID), inst.ID)
	a.showConfirm(text, func() {
		if err := a.requireCloudData(); err != nil {
			a.Notify(err.Error(), true)
			return
		}
		go a.runLifecycle("reboot", inst, func() error { return a.cloud.RebootInstance(a.ctx, inst.ID, false) })
	})
}

func (a *App) confirmTerminate() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	text := fmt.Sprintf("Terminate %s\n(%s)?\nThis cannot be undone.", blank(inst.Name, inst.ID), inst.ID)
	a.showConfirm(text, func() {
		if err := a.requireCloudData(); err != nil {
			a.Notify(err.Error(), true)
			return
		}
		go a.runLifecycle("terminate", inst, func() error { return a.cloud.DeleteInstance(a.ctx, inst.ID, false) })
	})
}

// runLifecycle performs a single lifecycle action, reports the outcome via
// a footer notice, and triggers a refresh on success so the instance's
// updated status shows up promptly. Called on its own goroutine by the
// do*/confirm* handlers above.
func (a *App) runLifecycle(verb string, inst model.ECSInstance, fn func() error) {
	label := blank(inst.Name, inst.ID)
	if err := fn(); err != nil {
		a.Notify(fmt.Sprintf("failed to %s %s: %v", verb, label, err), true)
		return
	}
	a.Notify(fmt.Sprintf("%s requested for %s", verb, label), false)
	a.refresh()
}

// showCommandInput opens the "E" run-command input for the selected
// instance.
func (a *App) showCommandInput() {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		return
	}
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}

	input := newCommandInput(blank(inst.Name, inst.ID),
		func(command string) {
			a.pages.RemovePage(pageCommand)
			a.showCommandRunning(inst, command)
			go a.doRunCommand(inst, command)
		},
		func() {
			a.pages.RemovePage(pageCommand)
			a.tapp.SetFocus(a.table.view)
		},
	)
	input.SetBorder(true).SetTitle(" Run command (no SSH) ")

	a.pages.AddPage(pageCommand, centered(input, 90, 3), true, true)
	a.tapp.SetFocus(input)
}

// showCommandRunning opens the output pane with a "running" placeholder;
// doRunCommand fills in the real output once the command finishes.
func (a *App) showCommandRunning(inst model.ECSInstance, command string) {
	view := newResultView(" Command output (Esc to close) ")
	view.SetText(fmt.Sprintf("Running %q on %s…", command, blank(inst.Name, inst.ID)))
	a.closeResultOnEsc(view)

	a.resultView = view
	a.pages.AddPage(pageResult, centered(view, 100, 30), true, true)
	a.tapp.SetFocus(view)
}

// closeResultOnEsc wires Esc to dismiss a result pane opened by
// showCommandRunning or the ":" palette and return focus to the table.
func (a *App) closeResultOnEsc(view *tview.TextView) {
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage(pageResult)
			a.tapp.SetFocus(a.table.view)
			return nil
		}
		return event
	})
}

func (a *App) doRunCommand(inst model.ECSInstance, command string) {
	r, err := a.cloud.RunCommand(a.ctx, inst.ID, command)
	a.tapp.QueueUpdateDraw(func() {
		if a.resultView == nil {
			return
		}
		if err != nil {
			a.resultView.SetText(commandErrorText(blank(inst.Name, inst.ID), command, err))
			return
		}
		a.resultView.SetText(commandResultText(blank(inst.Name, inst.ID), command, r))
	})
}

// showProfileSelect opens the Ctrl-P profile switcher, listing the
// aliyun-cli profiles found on disk.
func (a *App) showProfileSelect() {
	profiles, path, err := config.ListProfiles()
	if err != nil {
		a.Notify(err.Error(), true)
		return
	}

	closeProfiles := func() {
		a.pages.RemovePage(pageProfiles)
		a.tapp.SetFocus(a.table.view)
	}

	list := newProfileSelectList(profiles, a.cfg.Profile,
		func(p config.Profile) {
			closeProfiles()
			a.switchProfile(p)
		},
		closeProfiles,
	)
	list.SetBorder(true).SetTitle(" Switch profile — " + path + " ")

	a.pages.AddPage(pageProfiles, centered(list, 56, profileListHeight(len(profiles))), true, true)
	a.tapp.SetFocus(list)
}

func (a *App) switchProfile(p config.Profile) {
	a.cfg.Profile = p.Name
	if p.RegionID != "" {
		a.cfg.Region = p.RegionID
	}
	a.cloud = aliyun.New(a.cfg, nil)
	a.filter = ""
	a.Notify(fmt.Sprintf("profile switched to %s (region %s)", p.Name, blank(a.cfg.Region, "auto")), false)
	go func() {
		a.checkAccount(a.ctx)
		a.refresh()
	}()
}

func (a *App) switchRegion(region string) {
	a.cfg.Region = region
	a.cloud = aliyun.New(a.cfg, nil)
	a.filter = ""
	a.Notify("region set to "+region, false)
	go func() {
		a.checkAccount(a.ctx)
		a.refresh()
	}()
}
