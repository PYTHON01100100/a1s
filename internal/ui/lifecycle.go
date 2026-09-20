package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/model"
)

// checkAccount verifies the aliyun CLI can be reached with a usable account.
// SampleData mode never needs it (it's always "ready"); Demo mode never has
// an account. The result is cached on the App so requireCloudData and the
// dashboard don't need to shell out on every command.
func (a *App) checkAccount(ctx context.Context) bool {
	if a.cfg.SampleData {
		a.cloudReady, a.accountErr = true, nil
		return true
	}
	if a.cfg.Demo {
		a.cloudReady, a.accountErr = false, nil
		return false
	}
	a.accountErr = a.cloud.CheckCLI(ctx)
	a.cloudReady = a.accountErr == nil
	return a.cloudReady
}

// accountNotice explains, inside the app, why cloud data isn't available and
// how to fix it live instead of a1s exiting before the UI ever renders.
func (a *App) accountNotice() {
	fmt.Println()
	fmt.Println(yellow + bold + " ALIBABA CLOUD ACCOUNT NOT CONFIGURED" + reset)
	if _, err := exec.LookPath("aliyun"); err != nil {
		fmt.Println("  The aliyun CLI was not found in PATH.")
		fmt.Println(dim + "  Install it: https://github.com/aliyun/aliyun-cli#installation" + reset)
	} else {
		fmt.Println("  The aliyun CLI is installed but has no usable account/profile configured.")
		if a.accountErr != nil {
			fmt.Println(dim + "  " + a.accountErr.Error() + reset)
		}
	}
	fmt.Println()
	fmt.Printf("  Fix it live:   %sconfigure%s          add or update AccessKey ID/Secret and region\n", cyan, reset)
	fmt.Printf("  Then retry:    %sdashboard%s          reload once your account is ready\n", cyan, reset)
	fmt.Println(dim + "  Prefer not to use real credentials right now? Restart with `a1s --sample-data`." + reset)
}

// commonRegions is a curated reference of frequently used Alibaba Cloud
// regions, shown as a pick list during `configure`. It is intentionally not
// exhaustive — Alibaba Cloud adds regions over time — so the full/current
// list is always linked alongside it.
var commonRegions = []struct{ ID, Name string }{
	{"me-central-1", "Middle East — Riyadh, Saudi Arabia"},
	{"me-east-1", "Middle East — Dubai, UAE"},
	{"ap-southeast-1", "Asia Pacific — Singapore"},
	{"ap-southeast-3", "Asia Pacific — Kuala Lumpur, Malaysia"},
	{"ap-southeast-5", "Asia Pacific — Jakarta, Indonesia"},
	{"ap-south-1", "Asia Pacific — Mumbai, India"},
	{"ap-northeast-1", "Asia Pacific — Tokyo, Japan"},
	{"cn-hongkong", "China — Hong Kong"},
	{"cn-hangzhou", "China East 1 — Hangzhou"},
	{"cn-shanghai", "China East 2 — Shanghai"},
	{"cn-beijing", "China North 2 — Beijing"},
	{"cn-shenzhen", "China South 1 — Shenzhen"},
	{"eu-central-1", "Europe — Frankfurt, Germany"},
	{"eu-west-1", "Europe — London, UK"},
	{"us-east-1", "US East — Virginia"},
	{"us-west-1", "US West — Silicon Valley"},
}

// showRegions prints the common-regions reference list, numbered so it can
// be used both standalone (`regions`) and as a pick list inside `configure`.
func (a *App) showRegions() {
	fmt.Println()
	fmt.Println(orange + bold + " COMMON ALIBABA CLOUD REGIONS" + reset)
	for i, r := range commonRegions {
		fmt.Printf("  %s%2d%s  %-16s %s%s%s\n", orange, i+1, reset, r.ID, dim, r.Name, reset)
	}
	fmt.Println(dim + "  Not listed? Any region ID works — see the full, current list at:" + reset)
	fmt.Println(dim + "  https://www.alibabacloud.com/help/en/basics-for-beginners/regions-and-zones" + reset)
}

// pickRegion shows the common-regions list and reads one answer: a list
// number, a typed region ID, or a blank line to skip and decide later inside
// aliyun configure's own region prompt.
func (a *App) pickRegion() (string, error) {
	a.showRegions()
	fmt.Printf("\n%sRegion%s — number from the list above, a custom region ID, or Enter to skip: ", cyan, reset)
	line, err := a.editor.ReadLine("", nil)
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil
	}
	if n, convErr := strconv.Atoi(line); convErr == nil {
		if n < 1 || n > len(commonRegions) {
			return "", fmt.Errorf("region number must be 1-%d", len(commonRegions))
		}
		return commonRegions[n-1].ID, nil
	}
	return line, nil
}

// configureProfile walks through adding or updating an aliyun-cli profile:
// a friendly name (uat, client1, client2, ...), a region picked from a
// reference list, then the official, interactive `aliyun configure` for the
// AccessKey ID/Secret with the terminal handed over directly — so paste
// works exactly like any other terminal prompt, and a1s never reads, stores,
// or logs the secret itself. The line editor has already restored normal
// (cooked) terminal mode by the time a command handler runs, so running a
// foreground child process with inherited stdio here is safe.
func (a *App) configureProfile(ctx context.Context, args []string) error {
	if _, err := exec.LookPath("aliyun"); err != nil {
		return fmt.Errorf("aliyun CLI not found in PATH; install it first: https://github.com/aliyun/aliyun-cli#installation")
	}

	name := ""
	if len(args) > 0 {
		name = strings.TrimSpace(args[0])
	} else {
		fmt.Printf("\n%sProfile name%s (e.g. default, uat, client1, client2) [default]: ", cyan, reset)
		line, err := a.editor.ReadLine("", nil)
		if err != nil {
			return err
		}
		name = strings.TrimSpace(line)
	}
	if name == "" {
		name = "default"
	}

	region, err := a.pickRegion()
	if err != nil {
		return err
	}

	cliArgs := []string{"configure", "--profile", name}
	fmt.Printf("\n%s  Launching: aliyun %s%s\n", dim, strings.Join(cliArgs, " "), reset)
	fmt.Println(dim + "  Follow its prompts for AccessKey ID and AccessKey Secret (paste works normally)." + reset)
	if region != "" {
		fmt.Printf(dim+"  When it asks for the default region, you can use: %s%s\n", region, reset)
	}

	cmd := exec.CommandContext(ctx, "aliyun", cliArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aliyun configure exited with an error: %w", err)
	}

	if region != "" {
		// Re-apply the region non-interactively so the list-picked choice
		// always wins, even if the interactive prompt above was skipped or
		// answered differently. Only the (non-secret) region ID is passed
		// on the command line; AccessKey ID/Secret never are.
		if _, err := exec.CommandContext(ctx, "aliyun", "configure", "set", "--profile", name, "--region", region).CombinedOutput(); err != nil {
			fmt.Println(yellow + "⚠ Could not set the default region automatically: " + reset + err.Error())
		}
	}
	fmt.Println(green + "✓ aliyun-cli configuration updated." + reset)

	return a.switchProfile(ctx, name)
}

// dashboard is a compact, K9s-inspired status surface. The detailed resource
// view still lives in `ecs`; this gives users context before they start typing.
func (a *App) dashboard(ctx context.Context) {
	selected := "none"
	if a.selectedID != "" {
		selected = blank(a.selectedName, a.selectedID)
	}
	access := "read/write"
	accessColor := green
	if a.cfg.ReadOnly {
		access = "read-only"
		accessColor = yellow
	}

	fmt.Println()
	fmt.Println(dim + " ┌─ CONTEXT ───────────────────────────────┬─ SESSION ──────────────────────────────┐" + reset)
	fmt.Printf(" %s│%s profile  %-28s %s│%s selected  %-29s %s│%s\n", dim, reset, clip(blank(a.cfg.Profile, "default"), 28), dim, reset, clip(selected, 29), dim, reset)
	fmt.Printf(" %s│%s region   %-28s %s│%s access    %s%-29s%s %s│%s\n", dim, reset, clip(blank(a.cfg.Region, "auto"), 28), dim, reset, accessColor, access, reset, dim, reset)
	fmt.Println(dim + " └─────────────────────────────────────────┴───────────────────────────────────────┘" + reset)
	fmt.Printf(" %s:%s %secs%s  %s:%s %sstart|stop|reboot%s  %s:%s %sfilter%s  %s:%s %swatch%s  %s:%s %shelp%s\n",
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset,
		orange, reset, cyan, reset)
	_ = ctx
}

// watch redraws the ECS inventory on a fixed interval until the user presses
// Ctrl+C, giving a live-refreshing view similar to k9s/g1c.
func (a *App) watch(ctx context.Context, seconds int) error {
	fmt.Printf("\n%s%sLIVE WATCH%s  %severy %ds • press Ctrl+C to stop%s\n", orange, bold, reset, dim, seconds, reset)

	stop, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := a.ecs(ctx); err != nil {
		fmt.Println(red + err.Error() + reset)
	}

	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop.Done():
			fmt.Println(dim + "\nwatch stopped" + reset)
			return nil
		case <-ticker.C:
			fmt.Print("\033[2J\033[H")
			a.header()
			if err := a.ecs(ctx); err != nil {
				fmt.Println(red + err.Error() + reset)
			}
			fmt.Println(dim + "watching… press Ctrl+C to stop" + reset)
		}
	}
}

// lifecycleCommand parses the shared `<action> [target] [eco|--force]` grammar
// used by start/stop/reboot/terminate and dispatches to the matching aliyun
// client call. `target` is a row number, ECS name, or instance ID and may be
// omitted in favor of the currently selected instance.
func (a *App) lifecycleCommand(ctx context.Context, action string, args []string) error {
	if err := a.requireCloudData(); err != nil {
		return err
	}

	eco := false
	force := false
	rem := make([]string, 0, len(args))
	for _, x := range args {
		switch {
		case action == "stop" && (strings.EqualFold(x, "eco") || strings.EqualFold(x, "economic")):
			eco = true
		case strings.EqualFold(x, "--force") || strings.EqualFold(x, "-f"):
			force = true
		default:
			rem = append(rem, x)
		}
	}

	id, label, _, err := a.resolveTargetOrSelected(rem)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		return a.runLifecycle(ctx, "start", id, label, func() error { return a.cloud.StartInstance(ctx, id) })
	case "stop":
		mode := aliyun.StopNormal
		desc := "stop"
		if eco {
			mode = aliyun.StopEco
			desc = "stop (economic — billing paused)"
		}
		return a.runLifecycle(ctx, desc, id, label, func() error { return a.cloud.StopInstance(ctx, id, mode, force) })
	case "reboot":
		return a.runLifecycle(ctx, "reboot", id, label, func() error { return a.cloud.RebootInstance(ctx, id, force) })
	case "terminate":
		if !a.confirm(fmt.Sprintf("Terminate %s (%s)? This permanently deletes the instance and cannot be undone.", label, id)) {
			fmt.Println(dim + "cancelled" + reset)
			return nil
		}
		return a.runLifecycle(ctx, "terminate", id, label, func() error { return a.cloud.DeleteInstance(ctx, id, force) })
	default:
		return fmt.Errorf("unknown lifecycle action %q", action)
	}
}

func (a *App) runLifecycle(ctx context.Context, action, id, label string, fn func() error) error {
	fmt.Printf("\n %s%s%s%s  %s%s%s %s(%s)%s\n", orange, bold, strings.ToUpper(action), reset, dim, "target", reset, orange, label, reset)
	if err := fn(); err != nil {
		return err
	}
	fmt.Printf(" %s✓ %s requested for%s %s%s%s\n", green, action, reset, orange, label, reset)
	if xs, err := a.cloud.ListInstances(ctx); err == nil {
		if a.filter != "" {
			xs = filterInstances(xs, a.filter)
		}
		a.instances = xs
	}
	return nil
}

// confirm asks a yes/no question on the current line and returns true only
// for an explicit y/yes answer.
func (a *App) confirm(msg string) bool {
	fmt.Printf("%s%s⚠ %s%s %s[y/N]%s ", yellow, bold, msg, reset, dim, reset)
	line, err := a.editor.ReadLine("", nil)
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// filterInstances narrows an ECS list either by a `key=value` match against
// one field, or by a plain substring match across name/id/status/type/zone.
func filterInstances(xs []model.ECSInstance, q string) []model.ECSInstance {
	q = strings.TrimSpace(q)
	if q == "" {
		return xs
	}
	if k, v, ok := strings.Cut(q, "="); ok {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.ToLower(strings.TrimSpace(v))
		out := make([]model.ECSInstance, 0, len(xs))
		for _, x := range xs {
			var field string
			switch k {
			case "status":
				field = x.Status
			case "type":
				field = x.Type
			case "zone":
				field = x.Zone
			case "name":
				field = x.Name
			case "id":
				field = x.ID
			case "charge", "chargetype", "billing":
				field = x.ChargeType
			}
			if strings.Contains(strings.ToLower(field), v) {
				out = append(out, x)
			}
		}
		return out
	}
	ql := strings.ToLower(q)
	out := make([]model.ECSInstance, 0, len(xs))
	for _, x := range xs {
		hay := strings.ToLower(strings.Join([]string{x.Name, x.ID, x.Status, x.Type, x.Zone, x.ChargeType, x.PublicIP, x.PrivateIP}, " "))
		if strings.Contains(hay, ql) {
			out = append(out, x)
		}
	}
	return out
}

// showProfiles lists the aliyun-cli profiles found on disk so the user can
// switch accounts/regions without leaving a1s, mirroring g1c's multi-project
// support.
func (a *App) showProfiles() error {
	profiles, path, err := config.ListProfiles()
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(orange + bold + " ALIYUN CLI PROFILES" + reset)
	fmt.Println(dim + "  " + path + reset)
	if len(profiles) == 0 {
		fmt.Println(dim + "  No profiles found." + reset)
		return nil
	}
	for _, p := range profiles {
		mark := " "
		if strings.EqualFold(p.Name, a.cfg.Profile) {
			mark = "›"
		}
		fmt.Printf(" %s%s%s  %-20s %sregion%s %s\n", orange, mark, reset, p.Name, gray, reset, blank(p.RegionID, "-"))
	}
	fmt.Println(dim + "  Switch with: profile <name>" + reset)
	return nil
}

func (a *App) switchProfile(ctx context.Context, name string) error {
	profiles, _, err := config.ListProfiles()
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if !strings.EqualFold(p.Name, name) {
			continue
		}
		a.cfg.Profile = p.Name
		if p.RegionID != "" {
			a.cfg.Region = p.RegionID
		}
		a.cloud = aliyun.New(a.cfg, nil)
		a.selectedID, a.selectedName, a.filter = "", "", ""
		fmt.Printf("%s✓ profile switched to%s %s%s%s  %sregion%s %s\n", green, reset, orange, p.Name, reset, gray, reset, blank(a.cfg.Region, "auto"))
		if a.cfg.Demo && !a.cfg.SampleData {
			return nil
		}
		return a.ecs(ctx)
	}
	return fmt.Errorf("profile %q not found in aliyun-cli config; run `profiles` to list available profiles", name)
}

func (a *App) switchRegion(ctx context.Context, region string) error {
	a.cfg.Region = region
	a.cloud = aliyun.New(a.cfg, nil)
	a.selectedID, a.selectedName, a.filter = "", "", ""
	fmt.Printf("%s✓ region set to%s %s\n", green, reset, a.cfg.Region)
	if a.cfg.Demo && !a.cfg.SampleData {
		return nil
	}
	return a.ecs(ctx)
}

// setTheme swaps the decorative accent colors. Status colors (green/yellow/
// red) intentionally never change, so instance state always reads the same
// way regardless of theme.
func setTheme(name string) error {
	switch strings.ToLower(name) {
	case "alibaba", "default", "orange":
		orange, orange2 = "\x1b[38;2;255;106;0m", "\x1b[38;2;255;140;0m"
		cyan = "\x1b[38;2;66;211;255m"
		themeName = "alibaba"
	case "mono", "grayscale", "accessible":
		orange, orange2 = "\x1b[1m\x1b[38;2;235;238;242m", "\x1b[38;2;150;158;168m"
		cyan = "\x1b[4m"
		themeName = "mono"
	default:
		return fmt.Errorf("unknown theme %q; available: alibaba, mono", name)
	}
	return nil
}
