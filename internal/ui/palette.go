package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/currency"
	"github.com/PYTHON01100100/a1s/internal/model"
	"github.com/PYTHON01100100/a1s/internal/report"
)

// paletteHelp is shown as the palette input's placeholder-ish hint and in
// `:help`.
const paletteHelp = "bill [cycle] · doctor · report [file] · query <term> · metrics [minutes] · run <cmd> · uptime|disk|memory|failed|ports|quick · currency USD|SAR · theme alibaba|mono · profile <name> · region <id> · regions · profiles · configure [name]"

// newPaletteInput builds the ":" command palette input field, k9s-style.
func newPaletteInput(onSubmit func(line string), onCancel func()) *tview.InputField {
	field := tview.NewInputField().
		SetLabel(": ").
		SetFieldWidth(0)

	field.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := strings.TrimSpace(field.GetText())
			if text != "" {
				onSubmit(text)
				return
			}
		}
		onCancel()
	})

	return field
}

func (a *App) showPalette() {
	input := newPaletteInput(
		func(line string) {
			a.pages.RemovePage(pageCommand)
			a.tapp.SetFocus(a.table.view)
			a.runPalette(line)
		},
		func() {
			a.pages.RemovePage(pageCommand)
			a.tapp.SetFocus(a.table.view)
		},
	)
	input.SetBorder(true).SetTitle(" Command palette — " + paletteHelp + " ")

	a.pages.AddPage(pageCommand, centered(input, 100, 3), true, true)
	a.tapp.SetFocus(input)
}

// showPaletteResult opens the shared result pane with placeholder text,
// returning it so the caller can fill it in (synchronously, or later from a
// goroutine via QueueUpdateDraw).
func (a *App) showPaletteResult(title, placeholder string) *tview.TextView {
	view := newResultView(" " + title + " (Esc to close) ")
	view.SetText(placeholder)
	a.closeResultOnEsc(view)

	a.resultView = view
	a.pages.AddPage(pageResult, centered(view, 100, 30), true, true)
	a.tapp.SetFocus(view)
	return view
}

// runPalette parses and dispatches one ":" command line. State changes
// (currency/theme/profile/region) report through a footer notice; anything
// that produces a block of text (bill/doctor/report/query/metrics/regions/
// profiles/run/quick diagnostics) opens the shared result pane.
func (a *App) runPalette(line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	cmd := strings.ToLower(fields[0])
	args := fields[1:]

	switch cmd {
	case "currency":
		a.paletteCurrency(args)
	case "theme":
		a.paletteTheme(args)
	case "profile":
		a.paletteProfile(args)
	case "region":
		a.paletteRegion(args)
	case "regions":
		a.showPaletteResult("Regions", regionsText())
	case "profiles":
		a.paletteProfiles()
	case "configure":
		a.paletteConfigure(args)
	case "query", "find":
		a.paletteQuery(strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
	case "bill", "$":
		a.paletteBill(args)
	case "doctor", "d":
		a.paletteDoctor()
	case "report":
		a.paletteReport(args)
	case "metrics", "m":
		a.paletteMetrics(args)
	case "run", "r":
		a.paletteRun(strings.TrimSpace(strings.TrimPrefix(line, fields[0])))
	case "uptime", "disk", "memory", "failed", "ports", "quick":
		a.paletteQuick(cmd)
	case "help", "?":
		a.showPaletteResult("Palette help", "[::b]a1s command palette[-:-:-]\n\n"+strings.ReplaceAll(paletteHelp, " · ", "\n"))
	default:
		a.Notify(fmt.Sprintf("unknown palette command %q — try :help", cmd), true)
	}
}

func (a *App) paletteCurrency(args []string) {
	if len(args) == 0 {
		a.Notify("usage: currency USD|SAR", true)
		return
	}
	c := strings.ToUpper(args[0])
	if c != "USD" && c != "SAR" {
		a.Notify("supported display currencies: USD, SAR", true)
		return
	}
	a.cfg.Currency = c
	a.Notify("display currency set to "+c, false)
	a.refreshFooter()
}

func (a *App) paletteTheme(args []string) {
	if len(args) == 0 {
		a.Notify("active theme: "+themeName+" (available: alibaba, mono)", false)
		return
	}
	if err := setTheme(args[0]); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	applyTheme()
	a.header.SetInstance(selectedOrNil(a.table))
	a.applyFilter()
	a.refreshFooter()
	a.Notify("theme set to "+themeName, false)
}

func selectedOrNil(t *Table) *model.ECSInstance {
	inst, ok := t.SelectedInstance()
	if !ok {
		return nil
	}
	return &inst
}

func (a *App) paletteProfile(args []string) {
	if len(args) == 0 {
		a.Notify("usage: profile <name>; :profiles lists available profiles, or press Ctrl-P", true)
		return
	}
	profiles, _, err := config.ListProfiles()
	if err != nil {
		a.Notify(err.Error(), true)
		return
	}
	for _, p := range profiles {
		if strings.EqualFold(p.Name, args[0]) {
			a.switchProfile(p)
			return
		}
	}
	a.Notify(fmt.Sprintf("profile %q not found; :profiles lists available profiles", args[0]), true)
}

func (a *App) paletteRegion(args []string) {
	if len(args) == 0 {
		a.Notify("usage: region <region-id>; :regions shows a reference list", true)
		return
	}
	a.switchRegion(args[0])
}

func (a *App) paletteProfiles() {
	profiles, path, err := config.ListProfiles()
	if err != nil {
		a.Notify(err.Error(), true)
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[::b]aliyun-cli profiles[-:-:-]\n[gray]%s[-]\n\n", tview.Escape(path))
	if len(profiles) == 0 {
		b.WriteString("No profiles found.\n")
	}
	for _, p := range profiles {
		mark := " "
		if strings.EqualFold(p.Name, a.cfg.Profile) {
			mark = ">"
		}
		fmt.Fprintf(&b, "%s %-20s region %s\n", mark, p.Name, blank(p.RegionID, "-"))
	}
	b.WriteString("\nSwitch with: profile <name>  (or Ctrl-P)")
	a.showPaletteResult("Profiles", b.String())
}

// paletteConfigure suspends the tview screen (handing the real terminal
// back, cooked mode) so the official, interactive `aliyun configure` can
// run with inherited stdio exactly as it would outside a1s — pasting an
// AccessKey works normally, and a1s never reads, stores, or logs it.
func (a *App) paletteConfigure(args []string) {
	if _, err := exec.LookPath("aliyun"); err != nil {
		a.Notify("aliyun CLI not found in PATH; install it first: https://github.com/aliyun/aliyun-cli#installation", true)
		return
	}
	name := "default"
	if len(args) > 0 {
		name = args[0]
	}

	a.tapp.Suspend(func() {
		fmt.Printf("a1s: launching `aliyun configure --profile %s`\n", name)
		fmt.Println("Follow its prompts for AccessKey ID and AccessKey Secret (paste works normally).")
		cmd := exec.CommandContext(a.ctx, "aliyun", "configure", "--profile", name)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("aliyun configure exited with an error:", err)
			fmt.Println("Press Enter to return to a1s…")
			fmt.Scanln()
			return
		}
		fmt.Println("a1s: configuration updated. Press Enter to return…")
		fmt.Scanln()
	})

	profiles, _, err := config.ListProfiles()
	if err == nil {
		for _, p := range profiles {
			if p.Name == name {
				a.switchProfile(p)
				return
			}
		}
	}
	go a.refresh()
}

func (a *App) paletteQuery(term string) {
	if term == "" {
		a.Notify("usage: query <text|field=value>", true)
		return
	}
	matches := filterInstances(a.all, term)
	var b strings.Builder
	fmt.Fprintf(&b, "[::b]query[-:-:-] %q — %d match(es)\n", term, len(matches))
	if len(matches) == 0 {
		b.WriteString("\nTry: query <ip>  ·  query vpc=<id>  ·  query vswitch=<id>  ·  query cidr=<block>  ·  query eni=<id>")
	}
	for _, x := range matches {
		expireText, expireColor := billingExpiry(x)
		kindText, kindColor := instanceKind(x.Type)
		fmt.Fprintf(&b, "\n[gray]────────────────────────────────────────────────────────[-]\n")
		fmt.Fprintf(&b, "[%s::b]%s[-:-:-]  [gray]%s[-]  [%s]%s[-]\n", orange, tview.Escape(blank(x.Name, x.ID)), x.ID, kindColor, kindText)
		fmt.Fprintf(&b, "  status    %-10s  type      %s\n", x.Status, x.Type)
		fmt.Fprintf(&b, "  internal  %-15s external  %s\n", orDash(x.PrivateIP), orDash(x.PublicIP))
		fmt.Fprintf(&b, "  vpc       %s\n", pairLabelCIDR(x.VPCName, x.VPCID, x.VPCCIDR))
		fmt.Fprintf(&b, "  vswitch   %s\n", pairLabelCIDR(x.VSwitchName, x.VSwitchID, x.VSwitchCIDR))
		fmt.Fprintf(&b, "  zone      %s\n", zoneLabel(x.Zone))
		fmt.Fprintf(&b, "  eni       %s\n", orDash(x.ENIID))
		fmt.Fprintf(&b, "  billing   %-14s [%s]expires %s[-]\n", billingLabel(x.ChargeType), expireColor, expireText)
	}
	a.showPaletteResult("Query", b.String())
}

func (a *App) paletteBill(args []string) {
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	cycle := time.Now().Format("2006-01")
	if len(args) > 0 {
		cycle = args[0]
	}
	view := a.showPaletteResult("Billing", "Loading billing summary…")
	go func() {
		b, err := a.cloud.Bill(a.ctx, cycle)
		a.tapp.QueueUpdateDraw(func() {
			if a.resultView != view {
				return
			}
			if err != nil {
				view.SetText(tview.Escape(err.Error()))
				return
			}
			c := currency.Converter{SARPerUSD: a.cfg.SARPerUSD}
			shown := c.Convert(b.PretaxAmount, b.Currency, a.cfg.Currency)
			text := fmt.Sprintf("[::b]Billing[-:-:-]  [gray]%s[-]\n\nSettlement   %s %.2f\nDisplay      [%s]%s %.2f[-]\n",
				b.BillingCycle, b.Currency, b.PretaxAmount, orange, currency.Symbol(a.cfg.Currency), shown)
			if b.Currency != a.cfg.Currency {
				text += fmt.Sprintf("\n[gray]Display-only FX: 1 USD = %.4f SAR[-]\n", a.cfg.SARPerUSD)
			}
			view.SetText(text)
		})
	}()
}

func (a *App) paletteDoctor() {
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	view := a.showPaletteResult("Doctor", "Running health checks…")
	go func() {
		xs, err := a.cloud.ListInstances(a.ctx)
		var pts []model.MetricPoint
		if err == nil && len(xs) > 0 {
			pts, _ = a.cloud.CPU(a.ctx, xs[0].ID, 60)
		}
		a.tapp.QueueUpdateDraw(func() {
			if a.resultView != view {
				return
			}
			if err != nil {
				view.SetText(tview.Escape(err.Error()))
				return
			}
			view.SetText(tview.Escape(report.Doctor(xs, pts)))
		})
	}()
}

func (a *App) paletteReport(args []string) {
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	path := "a1s-report.md"
	if len(args) > 0 {
		path = args[0]
	}
	go func() {
		xs, err := a.cloud.ListInstances(a.ctx)
		if err != nil {
			a.Notify("report failed: "+err.Error(), true)
			return
		}
		b, err := a.cloud.Bill(a.ctx, time.Now().Format("2006-01"))
		if err != nil {
			a.Notify("report failed: "+err.Error(), true)
			return
		}
		md := report.Markdown(xs, b, a.cfg.Currency, a.cfg.SARPerUSD)
		if err := os.WriteFile(path, []byte(md), 0644); err != nil {
			a.Notify("report failed: "+err.Error(), true)
			return
		}
		a.Notify("report written to "+path, false)
	}()
}

func (a *App) paletteMetrics(args []string) {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		a.Notify("select an instance first", true)
		return
	}
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	mins := 60
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			mins = n
		}
	}
	view := a.showPaletteResult("CPU metrics", "Loading metrics…")
	go func() {
		ps, err := a.cloud.CPU(a.ctx, inst.ID, mins)
		a.tapp.QueueUpdateDraw(func() {
			if a.resultView != view {
				return
			}
			if err != nil {
				view.SetText(tview.Escape(err.Error()))
				return
			}
			if len(ps) == 0 {
				view.SetText("No metric points returned.")
				return
			}
			var b strings.Builder
			fmt.Fprintf(&b, "[::b]CPUUtilization[-:-:-]  [gray]%s • last %dm[-]\n\n", blank(inst.Name, inst.ID), mins)
			for _, p := range ps {
				fmt.Fprintf(&b, "%-9s %6.1f%%  %s\n", time.UnixMilli(p.Timestamp).Format("15:04:05"), p.Average, bar(p.Average))
			}
			view.SetText(b.String())
		})
	}()
}

func (a *App) paletteRun(shell string) {
	if shell == "" {
		a.Notify("usage: run <command> (selected instance), or press E", true)
		return
	}
	inst, ok := a.table.SelectedInstance()
	if !ok {
		a.Notify("select an instance first", true)
		return
	}
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	a.showCommandRunning(inst, shell)
	go a.doRunCommand(inst, shell)
}

var quickCommands = map[string]string{
	"uptime": "uptime",
	"disk":   "df -h /",
	"memory": "free -h 2>/dev/null || true",
	"failed": "systemctl --failed --no-pager 2>/dev/null || true",
	"ports":  "ss -tulpn 2>/dev/null | head -30 || true",
}

func (a *App) paletteQuick(name string) {
	inst, ok := a.table.SelectedInstance()
	if !ok {
		a.Notify("select an instance first", true)
		return
	}
	if err := a.requireCloudData(); err != nil {
		a.Notify(err.Error(), true)
		return
	}
	if name == "quick" {
		view := a.showPaletteResult("Quick diagnostics", "Running uptime, disk, memory, failed services, and ports…")
		go func() {
			var b strings.Builder
			for _, n := range []string{"uptime", "disk", "memory", "failed", "ports"} {
				r, err := a.cloud.RunCommand(a.ctx, inst.ID, quickCommands[n])
				fmt.Fprintf(&b, "[::b]%s[-:-:-]\n", n)
				if err != nil {
					fmt.Fprintf(&b, "[red]%s[-]\n\n", tview.Escape(err.Error()))
					continue
				}
				b.WriteString(indentOutput(r.Output, "  "))
				b.WriteString("\n\n")
			}
			a.tapp.QueueUpdateDraw(func() {
				if a.resultView != view {
					return
				}
				view.SetText(b.String())
			})
		}()
		return
	}
	a.showCommandRunning(inst, quickCommands[name])
	go a.doRunCommand(inst, quickCommands[name])
}
