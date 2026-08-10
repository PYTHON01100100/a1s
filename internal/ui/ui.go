package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"a1s/internal/ai"
	"a1s/internal/aliyun"
	"a1s/internal/config"
	"a1s/internal/currency"
	"a1s/internal/model"
	"a1s/internal/report"
)

const (
	reset   = "\x1b[0m"
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	orange  = "\x1b[38;2;255;106;0m" // Alibaba Cloud orange
	orange2 = "\x1b[38;2;255;140;0m"
	cyan    = "\x1b[38;2;66;211;255m"
	green   = "\x1b[38;2;72;199;116m"
	yellow  = "\x1b[38;2;255;196;61m"
	red     = "\x1b[38;2;255;92;92m"
	gray    = "\x1b[38;2;150;158;168m"
	white   = "\x1b[38;2;235;238;242m"
)

type App struct {
	cfg          config.Config
	cloud        *aliyun.Client
	ai           *ai.Client
	instances    []model.ECSInstance
	selectedID   string
	selectedName string
}

func New(cfg config.Config, cloud *aliyun.Client, aic *ai.Client) *App {
	return &App{cfg: cfg, cloud: cloud, ai: aic}
}

func (a *App) Run(ctx context.Context) error {
	a.header()
	if err := a.ecs(ctx); err != nil {
		fmt.Println(yellow + "⚠ Could not load ECS inventory: " + reset + err.Error())
	}
	a.help()

	s := bufio.NewScanner(os.Stdin)
	for {
		a.prompt()
		if !s.Scan() {
			return s.Err()
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if line == "q" || line == "quit" || line == "exit" {
			fmt.Println(dim + "bye 👋" + reset)
			return nil
		}
		if err := a.handle(ctx, line); err != nil {
			fmt.Println(red + bold + "ERROR " + reset + red + err.Error() + reset)
		}
	}
}

func (a *App) header() {
	mode := "LIVE"
	modeColor := green
	if a.cfg.Demo {
		mode = "DEMO"
		modeColor = yellow
	}
	access := "RW"
	if a.cfg.ReadOnly {
		access = "READ-ONLY"
	}

	fmt.Print("\033[2J\033[H")
	fmt.Println(orange + bold + "   ▄▀█  █ █▀" + reset + "   " + white + bold + "Alibaba Cloud Operations" + reset)
	fmt.Println(orange2 + "   █▀█  █ ▄█" + reset + "   " + dim + "Browse • Observe • Run • Analyze" + reset)
	fmt.Println(dim + "────────────────────────────────────────────────────────────────────────" + reset)
	fmt.Printf(" %s%s● %s%s  %s%s%s  %sprofile%s %s  %sregion%s %s  %scurrency%s %s\n",
		modeColor, bold, mode, reset,
		gray, access, reset,
		gray, reset, white+blank(a.cfg.Profile, "default")+reset,
		gray, reset, white+blank(a.cfg.Region, "profile default")+reset,
		gray, reset, orange+bold+a.cfg.Currency+reset)
	if a.cfg.ConfigSource == "aliyun-cli" {
		fmt.Printf(" %s✓ aliyun-cli profile auto-detected%s  %s%s%s\n", green, reset, dim, a.cfg.AliyunConfig, reset)
	}
	fmt.Println(dim + "────────────────────────────────────────────────────────────────────────" + reset)
}

func (a *App) prompt() {
	scope := blank(a.cfg.Profile, "default") + "/" + blank(a.cfg.Region, "auto")
	selected := ""
	if a.selectedID != "" {
		selected = " " + orange + "[" + blank(a.selectedName, a.selectedID) + "]" + reset
	}
	fmt.Printf("\n%s%s%s%s%s a1s%s%s › ", dim, "[", scope, "]", reset, selected, orange)
	fmt.Print(reset)
}

func (a *App) help() {
	fmt.Println()
	fmt.Println(orange + bold + " QUICK KEYS / COMMANDS" + reset)
	fmt.Printf("  %secs/ls%s inventory   %suse 1%s select ECS   %smetrics%s CPU   %srun <cmd>%s execute\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sdoctor%s health      %squick%s checks       %sbill%s costs    %sai <q>%s copilot\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sreport%s markdown    %scurrency%s display    %sclear%s home   %s?%s help   %sq%s quit\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Println(dim + "  Tip: select once with `use 1`, then `metrics`, `doctor`, `quick`, or `run uptime`." + reset)
}

func (a *App) handle(ctx context.Context, line string) error {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

	switch cmd {
	case "help", "?":
		a.help()
		return nil
	case "clear", "home":
		a.header()
		return a.ecs(ctx)
	case "ecs", "ls":
		return a.ecs(ctx)
	case "use", "select":
		if len(parts) < 2 {
			return fmt.Errorf("usage: use <row-number|instance-id>")
		}
		return a.selectInstance(parts[1])
	case "metrics", "m":
		id, mins, err := a.resolveMetricsArgs(parts[1:])
		if err != nil {
			return err
		}
		return a.metrics(ctx, id, mins)
	case "run", "r":
		id, shell, err := a.resolveRunArgs(rest)
		if err != nil {
			return err
		}
		return a.runCmd(ctx, id, shell)
	case "quick":
		id := a.selectedID
		if len(parts) > 1 {
			id = parts[1]
		}
		if id == "" {
			return fmt.Errorf("select an ECS first: use <row-number>, or quick <instance-id>")
		}
		return a.quick(ctx, id)
	case "doctor", "d":
		id := a.selectedID
		if len(parts) > 1 {
			id = parts[1]
		}
		return a.doctor(ctx, id)
	case "bill", "$":
		cycle := time.Now().Format("2006-01")
		if len(parts) > 1 {
			cycle = parts[1]
		}
		return a.bill(ctx, cycle)
	case "report":
		path := "a1s-report.md"
		if len(parts) > 1 {
			path = parts[1]
		}
		return a.makeReport(ctx, path)
	case "ai", "a":
		if rest == "" {
			return fmt.Errorf("usage: ai <question>")
		}
		return a.askAI(ctx, rest)
	case "currency":
		if len(parts) < 2 {
			return fmt.Errorf("usage: currency USD|SAR")
		}
		c := strings.ToUpper(parts[1])
		if c != "USD" && c != "SAR" {
			return fmt.Errorf("supported display currencies: USD, SAR")
		}
		a.cfg.Currency = c
		fmt.Println(green + "✓ Display currency set to " + c + reset)
		return nil
	default:
		return fmt.Errorf("unknown command %q; type ? for help", cmd)
	}
}

func (a *App) selectInstance(ref string) error {
	if len(a.instances) == 0 {
		return fmt.Errorf("ECS inventory is empty; run ecs first")
	}
	if n, err := strconv.Atoi(ref); err == nil {
		if n < 1 || n > len(a.instances) {
			return fmt.Errorf("row must be between 1 and %d", len(a.instances))
		}
		x := a.instances[n-1]
		a.selectedID, a.selectedName = x.ID, x.Name
		fmt.Printf("%s✓ Selected%s %s%s%s %s(%s)%s\n", green, reset, orange, blank(x.Name, x.ID), reset, dim, x.ID, reset)
		return nil
	}
	for _, x := range a.instances {
		if x.ID == ref {
			a.selectedID, a.selectedName = x.ID, x.Name
			fmt.Printf("%s✓ Selected%s %s%s%s\n", green, reset, orange, blank(x.Name, x.ID), reset)
			return nil
		}
	}
	return fmt.Errorf("instance %q not found in current ECS inventory", ref)
}

func (a *App) resolveMetricsArgs(args []string) (string, int, error) {
	id := a.selectedID
	mins := 60
	if len(args) == 0 {
		if id == "" {
			return "", 0, fmt.Errorf("select an ECS first: use <row-number>, or metrics <instance-id> [minutes]")
		}
		return id, mins, nil
	}
	if n, err := strconv.Atoi(args[0]); err == nil && id != "" {
		return id, n, nil
	}
	id = args[0]
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil {
			mins = n
		}
	}
	return id, mins, nil
}

func (a *App) resolveRunArgs(rest string) (string, string, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", fmt.Errorf("usage after selecting: run <command>; legacy: run <instance-id> <command>")
	}
	fields := strings.Fields(rest)
	if len(fields) >= 2 && strings.HasPrefix(fields[0], "i-") {
		id := fields[0]
		shell := strings.TrimSpace(strings.TrimPrefix(rest, id))
		return id, shell, nil
	}
	if a.selectedID == "" {
		return "", "", fmt.Errorf("select an ECS first: use <row-number>, or run <instance-id> <command>")
	}
	return a.selectedID, rest, nil
}

func (a *App) ecs(ctx context.Context) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	a.instances = xs

	fmt.Println()
	fmt.Printf(" %s%sECS INSTANCES%s  %s%d resources%s\n", orange, bold, reset, dim, len(xs), reset)
	fmt.Println(dim + " ───────────────────────────────────────────────────────────────────────────────────────────" + reset)
	fmt.Printf(" %s%-3s %-20s %-23s %-10s %-19s %-17s %-15s%s\n", gray, "#", "NAME", "INSTANCE ID", "STATUS", "TYPE", "ZONE", "PRIVATE IP", reset)
	for i, x := range xs {
		statusColor := yellow
		marker := " "
		if strings.EqualFold(x.Status, "Running") {
			statusColor = green
		} else if strings.EqualFold(x.Status, "Stopped") {
			statusColor = red
		}
		if x.ID == a.selectedID {
			marker = "›"
		}
		fmt.Printf(" %s%s%-3d%s %-20s %s%-23s%s %s%-10s%s %-19s %-17s %-15s\n",
			orange, marker, i+1, reset,
			clip(blank(x.Name, "-"), 20), dim, clip(x.ID, 23), reset,
			statusColor, clip(x.Status, 10), reset,
			clip(x.Type, 19), clip(x.Zone, 17), clip(x.PrivateIP, 15))
	}
	fmt.Println(dim + " ───────────────────────────────────────────────────────────────────────────────────────────" + reset)
	if len(xs) > 0 && a.selectedID == "" {
		fmt.Printf(" %sTip:%s %suse 1%s selects the first instance; no more copying instance IDs.\n", dim, reset, cyan, reset)
	}
	return nil
}

func (a *App) metrics(ctx context.Context, id string, mins int) error {
	ps, err := a.cloud.CPU(ctx, id, mins)
	if err != nil {
		return err
	}
	if len(ps) == 0 {
		fmt.Println(yellow + "No metric points returned." + reset)
		return nil
	}
	fmt.Printf("\n %s%sCPUUtilization%s  %s%s • last %dm%s\n", orange, bold, reset, dim, id, mins, reset)
	fmt.Printf(" %s%-12s %8s %8s  %s%s\n", gray, "TIME", "AVG", "MAX", "GRAPH", reset)
	for _, p := range ps {
		fmt.Printf(" %-12s %7.1f%% %7.1f%%  %s%s%s\n", time.UnixMilli(p.Timestamp).Format("15:04:05"), p.Average, p.Maximum, cyan, bar(p.Average), reset)
	}
	return nil
}

func (a *App) runCmd(ctx context.Context, id, cmd string) error {
	fmt.Printf("\n %s%sRUN COMMAND%s  %s%s%s\n", orange, bold, reset, dim, id, reset)
	fmt.Printf(" %s$%s %s\n", orange, reset, cmd)
	r, err := a.cloud.RunCommand(ctx, id, cmd)
	if err != nil {
		return err
	}
	statusColor := green
	if r.ExitCode != 0 || !strings.EqualFold(r.Status, "Finished") {
		statusColor = red
	}
	fmt.Printf(" %s%s%s%s  %sexit=%d%s  %sinvoke=%s%s\n", statusColor, bold, r.Status, reset, dim, r.ExitCode, reset, dim, r.InvokeID, reset)
	fmt.Println(dim + " ┌─ output ──────────────────────────────────────────────────────────────" + reset)
	fmt.Println(indentOutput(r.Output, " │ "))
	fmt.Println(dim + " └──────────────────────────────────────────────────────────────────────" + reset)
	return nil
}

func (a *App) quick(ctx context.Context, id string) error {
	for _, q := range []string{"uptime", "df -h /", "free -h 2>/dev/null || true", "systemctl --failed --no-pager 2>/dev/null || true", "ss -tulpn 2>/dev/null | head -30 || true"} {
		if err := a.runCmd(ctx, id, q); err != nil {
			fmt.Println(red + "  " + err.Error() + reset)
		}
	}
	return nil
}

func (a *App) doctor(ctx context.Context, id string) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	var m []model.MetricPoint
	if id != "" {
		m, _ = a.cloud.CPU(ctx, id, 60)
	} else if len(xs) > 0 {
		m, _ = a.cloud.CPU(ctx, xs[0].ID, 60)
	}
	text := report.Doctor(xs, m)
	fmt.Printf("\n%s%sDOCTOR%s\n%s", orange, bold, reset, text)
	if a.ai.Enabled() {
		ans, err := a.ai.Ask(ctx, "You are an Alibaba Cloud SRE. Analyze only the supplied operational data. Be concise, separate evidence from suggestions, and never claim a command was run unless output is present.", text)
		if err == nil {
			fmt.Println("\n" + orange + bold + "AI ANALYSIS" + reset + "\n" + ans)
		}
	}
	return nil
}

func (a *App) bill(ctx context.Context, cycle string) error {
	b, err := a.cloud.Bill(ctx, cycle)
	if err != nil {
		return err
	}
	c := currency.Converter{SARPerUSD: a.cfg.SARPerUSD}
	shown := c.Convert(b.PretaxAmount, b.Currency, a.cfg.Currency)
	fmt.Printf("\n %s%sBILLING%s  %s%s%s\n", orange, bold, reset, dim, b.BillingCycle, reset)
	fmt.Printf(" Settlement   %s %.2f\n", b.Currency, b.PretaxAmount)
	fmt.Printf(" Display      %s%s %.2f%s\n", orange, currency.Symbol(a.cfg.Currency), shown, reset)
	if b.Currency != a.cfg.Currency {
		fmt.Printf(" %sDisplay-only FX: 1 USD = %.4f SAR%s\n", dim, a.cfg.SARPerUSD, reset)
	}
	return nil
}

func (a *App) makeReport(ctx context.Context, path string) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	b, err := a.cloud.Bill(ctx, time.Now().Format("2006-01"))
	if err != nil {
		return err
	}
	md := report.Markdown(xs, b, a.cfg.Currency, a.cfg.SARPerUSD)
	if a.ai.Enabled() {
		if extra, err := a.ai.Ask(ctx, "You are an Alibaba Cloud operations analyst. Produce a short executive analysis of the following report. Do not invent metrics.", md); err == nil {
			md += "\n## AI Analysis\n\n" + extra + "\n"
		}
	}
	if err := os.WriteFile(path, []byte(md), 0644); err != nil {
		return err
	}
	fmt.Printf("%s✓ Report written to%s %s\n", green, reset, path)
	return nil
}

func (a *App) askAI(ctx context.Context, q string) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	ctxText := fmt.Sprintf("Region=%s Currency=%s ECS=%v\nQuestion=%s", a.cfg.Region, a.cfg.Currency, xs, q)
	ans, err := a.ai.Ask(ctx, "You are a read-only Alibaba Cloud operations copilot inside a1s. Use supplied context. Give safe diagnostic guidance. Commands are suggestions only and require explicit user execution.", ctxText)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s%sAI COPILOT%s\n%s\n", orange, bold, reset, ans)
	return nil
}

func blank(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func bar(v float64) string {
	n := int(v / 5)
	if n > 20 {
		n = 20
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("█", n) + dim + strings.Repeat("░", 20-n) + reset
}

func indentOutput(s, prefix string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return prefix + dim + "(no output)" + reset
	}
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
