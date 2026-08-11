package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
	ollamaModel  string
	ollamaURL    string
	editor       *lineEditor
}

func New(cfg config.Config, cloud *aliyun.Client, aic *ai.Client) *App {
	return &App{cfg: cfg, cloud: cloud, ai: aic, ollamaURL: "http://127.0.0.1:11434", editor: newLineEditor()}
}

func (a *App) Run(ctx context.Context) error {
	a.header()
	if a.cfg.Demo && !a.cfg.SampleData {
		a.demoNotice()
	} else if err := a.ecs(ctx); err != nil {
		fmt.Println(yellow + "⚠ Could not load ECS inventory: " + reset + err.Error())
	}
	a.help()

	for {
		line, err := a.editor.ReadLine(a.promptText(), a.completions)
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
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
	if a.cfg.SampleData {
		mode = "SAMPLE DATA"
		modeColor = yellow
	} else if a.cfg.Demo {
		mode = "UI ONLY"
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

func (a *App) promptText() string {
	scope := blank(a.cfg.Profile, "default") + "/" + blank(a.cfg.Region, "auto")
	selected := ""
	if a.selectedID != "" {
		selected = " " + orange + "[" + blank(a.selectedName, a.selectedID) + "]" + reset
	}
	return fmt.Sprintf("%s[%s]%s%s a1s%s%s › ", dim, scope, reset, selected, orange, reset)
}

func (a *App) help() {
	fmt.Println()
	fmt.Println(orange + bold + " QUICK COMMANDS" + reset)
	fmt.Printf("  %secs/ls%s inventory      %suse 1%s select ECS      %smetrics%s CPU       %srun <ecs> <cmd>%s execute\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %squick list%s shortcuts  %sdoctor%s health         %sbill%s costs        %sreport%s markdown\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %suptime%s selected ECS   %sdisk%s filesystem       %smemory%s RAM        %sports%s listeners\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sfailed%s services       %sai providers%s AI setup   %sai models%s       %sai ask <q>%s ask AI\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %scurrency%s USD|SAR      %sclear%s home            %shelp/?%s          %sq%s quit\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Println(dim + "  Tip: ↑/↓ history • ←/→ cursor • Tab autocomplete • <command> --help • help <command>" + reset)
}

func (a *App) fullHelp() {
	fmt.Println()
	fmt.Println(orange + bold + " A1S HELP" + reset)
	fmt.Println(dim + " Alibaba Cloud terminal operations • commands are case-insensitive" + reset)
	fmt.Println()
	fmt.Println(white + bold + " RESOURCE BROWSING" + reset)
	fmt.Printf("  %secs%s, %sls%s                    List ECS inventory and network details\n", cyan, reset, cyan, reset)
	fmt.Printf("  %suse <row|name|id>%s          Select an ECS for following commands\n", cyan, reset)
	fmt.Printf("  %smetrics [target] [minutes]%s Show CPU metrics (default 60 minutes)\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " REMOTE EXECUTION" + reset)
	fmt.Printf("  %srun <ecs-name> <command>%s   Execute on an ECS by name, row, or instance ID\n", cyan, reset)
	fmt.Printf("  %srun <command>%s              Execute on the currently selected ECS\n", cyan, reset)
	fmt.Printf("  %suptime | disk | memory%s     Safe shortcuts for selected ECS\n", cyan, reset)
	fmt.Printf("  %sfailed | ports | quick%s     Service/port checks or all quick checks\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " OPERATIONS & FINOPS" + reset)
	fmt.Printf("  %sdoctor [target]%s            Infrastructure health summary\n", cyan, reset)
	fmt.Printf("  %sbill [YYYY-MM]%s             Account billing summary\n", cyan, reset)
	fmt.Printf("  %sreport [file.md]%s           Write a Markdown operations report\n", cyan, reset)
	fmt.Printf("  %scurrency USD|SAR%s           Change display currency\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " AI" + reset)
	fmt.Printf("  %sai providers%s               Show supported/configured providers\n", cyan, reset)
	fmt.Printf("  %sai use ollama%s              Use local Ollama\n", cyan, reset)
	fmt.Printf("  %sai models%s                  List locally installed Ollama models\n", cyan, reset)
	fmt.Printf("  %sai model <number|name>%s     Select a local model\n", cyan, reset)
	fmt.Printf("  %sai ask <question>%s          Ask the active AI provider\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " TERMINAL" + reset)
	fmt.Printf("  %s↑ / ↓%s  command history     %s← / →%s  move cursor     %sTab%s  autocomplete\n", cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sclear%s home                 %shelp <command>%s detailed help     %sq%s quit\n", cyan, reset, cyan, reset, cyan, reset)
	fmt.Println()
	fmt.Println(dim + " Examples: run test pwd • run test 'df -h' • help run • ai --help • metrics --help" + reset)
}

func (a *App) commandHelp(cmd string) error {
	cmd = strings.ToLower(strings.TrimSpace(cmd))
	fmt.Println()
	fmt.Printf("%s%s%s HELP%s\n", orange, bold, strings.ToUpper(cmd), reset)
	switch cmd {
	case "ecs", "ls":
		fmt.Println("  List ECS resources from the current Alibaba Cloud profile/region.")
		fmt.Println("  Shows: name, instance ID, status, type, OS, internal/external IP, billing, VPC, vSwitch, and zone.")
		fmt.Println("\n  Usage:\n    ecs\n    ls")
	case "use", "select":
		fmt.Println("  Select an ECS once, then run commands without repeating its name.")
		fmt.Println("\n  Usage:\n    use <row|ecs-name|instance-id>")
		fmt.Println("\n  Examples:\n    use 1\n    use test\n    use i-xxxxxxxx")
	case "run", "r":
		fmt.Println("  Execute a shell command through Alibaba Cloud Cloud Assistant.")
		fmt.Println("  The target can be an ECS name, table row number, or instance ID.")
		fmt.Println("\n  Usage:\n    run <ecs-name|row|instance-id> <command>\n    run <command>                 # when an ECS is selected")
		fmt.Println("\n  Examples:\n    run test pwd\n    run test uptime\n    run test df -h\n    use test\n    run systemctl status nginx")
		if a.cfg.ReadOnly {
			fmt.Println("\n  Note: RunCommand is disabled because a1s is running with --read-only.")
		}
	case "metrics", "m":
		fmt.Println("  Display ECS CPU utilization for a time window.")
		fmt.Println("\n  Usage:\n    metrics <ecs-name|row|instance-id> [minutes]\n    metrics [minutes]             # selected ECS")
		fmt.Println("\n  Examples:\n    metrics test 60\n    use test\n    metrics 30")
	case "quick":
		fmt.Println("  Run the safe ECS diagnostic shortcuts: uptime, disk, memory, failed services, and listening ports.")
		fmt.Println("\n  Usage:\n    quick\n    quick list")
	case "uptime", "disk", "memory", "failed", "ports":
		fmt.Printf("  Run the %q quick diagnostic on the selected ECS.\n", cmd)
		fmt.Printf("\n  Usage:\n    use <ecs>\n    %s\n", cmd)
	case "doctor", "d":
		fmt.Println("  Summarize infrastructure health from ECS inventory and CPU metrics; adds AI analysis when AI is configured.")
		fmt.Println("\n  Usage:\n    doctor\n    doctor <instance-id>")
	case "bill", "$":
		fmt.Println("  Show account billing for a billing cycle in settlement currency and selected display currency.")
		fmt.Println("\n  Usage:\n    bill\n    bill YYYY-MM\n\n  Example:\n    bill 2026-08")
	case "report":
		fmt.Println("  Generate a Markdown report from current cloud inventory and billing data.")
		fmt.Println("\n  Usage:\n    report [output.md]\n\n  Example:\n    report ops-report.md")
	case "ai", "a":
		fmt.Println("  AI namespace. Ollama is supported locally; OpenAI-compatible endpoints can be configured through environment variables.")
		fmt.Println("\n  Usage:\n    ai providers\n    ai use ollama\n    ai models\n    ai model <number|name>\n    ai ask <question>")
		fmt.Println("\n  Example:\n    ai use ollama\n    ai models\n    ai model 1\n    ai ask explain the health of my ECS resources")
	case "ollama":
		fmt.Println("  Compatibility namespace for local Ollama. Prefer the unified `ai` commands for normal use.")
		fmt.Println("\n  Usage:\n    ollama status\n    ollama models\n    ollama start\n    ollama use <model>\n    ollama ask <question>")
	case "currency":
		fmt.Println("  Change display currency without changing Alibaba Cloud settlement currency.")
		fmt.Println("\n  Usage:\n    currency USD\n    currency SAR")
	case "clear", "home":
		fmt.Println("  Clear the screen, redraw the header, and reload ECS inventory when cloud access is enabled.")
		fmt.Println("\n  Usage:\n    clear\n    home")
	case "help", "?", "--help", "-h":
		a.fullHelp()
	default:
		return fmt.Errorf("no help topic for %q; use --help to list commands", cmd)
	}
	return nil
}

func (a *App) demoNotice() {
	fmt.Println()
	fmt.Println(yellow + bold + " UI-ONLY MODE" + reset)
	fmt.Println("  No Alibaba Cloud APIs are called and no fake ECS, billing, metrics, or health data are shown.")
	fmt.Println(dim + "  Use `a1s` without --demo for your real aliyun-cli account, or --sample-data for explicit samples." + reset)
}

func (a *App) handle(ctx context.Context, line string) error {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

	// Help is available globally and per command: --help, help, help run, run --help, ai --help, etc.
	if cmd == "--help" || cmd == "-h" {
		a.fullHelp()
		return nil
	}
	if cmd == "help" || cmd == "?" {
		if len(parts) > 1 {
			return a.commandHelp(strings.ToLower(parts[1]))
		}
		a.fullHelp()
		return nil
	}
	if len(parts) > 1 && (parts[1] == "--help" || parts[1] == "-h") {
		return a.commandHelp(cmd)
	}

	switch cmd {
	case "clear", "home":
		a.header()
		if a.cfg.Demo && !a.cfg.SampleData {
			a.demoNotice()
			return nil
		}
		return a.ecs(ctx)
	case "ecs", "ls":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		return a.ecs(ctx)
	case "use", "select":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		if len(parts) < 2 {
			return fmt.Errorf("usage: use <row-number|instance-id>")
		}
		return a.selectInstance(parts[1])
	case "metrics", "m":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		id, mins, err := a.resolveMetricsArgs(parts[1:])
		if err != nil {
			return err
		}
		return a.metrics(ctx, id, mins)
	case "run", "r":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		id, shell, err := a.resolveRunArgs(rest)
		if err != nil {
			return err
		}
		return a.runCmd(ctx, id, shell)
	case "quick":
		if len(parts) > 1 && strings.EqualFold(parts[1], "list") {
			a.quickHelp()
			return nil
		}
		if err := a.requireCloudData(); err != nil {
			return err
		}
		id := a.selectedID
		if len(parts) > 1 && strings.HasPrefix(parts[1], "i-") {
			id = parts[1]
		}
		if id == "" {
			return fmt.Errorf("select an ECS first: use <row-number>, or quick <instance-id>")
		}
		return a.quick(ctx, id)
	case "uptime", "disk", "memory", "failed", "ports":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		if a.selectedID == "" {
			return fmt.Errorf("select an ECS first with: use <row-number>")
		}
		return a.quickOne(ctx, a.selectedID, cmd)

	case "doctor", "d":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		id := a.selectedID
		if len(parts) > 1 {
			id = parts[1]
		}
		return a.doctor(ctx, id)
	case "bill", "$":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		cycle := time.Now().Format("2006-01")
		if len(parts) > 1 {
			cycle = parts[1]
		}
		return a.bill(ctx, cycle)
	case "report":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		path := "a1s-report.md"
		if len(parts) > 1 {
			path = parts[1]
		}
		return a.makeReport(ctx, path)
	case "ollama":
		return a.handleOllama(ctx, parts[1:], strings.TrimSpace(rest))
	case "ai", "a":
		return a.handleAI(ctx, parts[1:], rest)
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
		return fmt.Errorf("unknown command %q; use --help or help <command> (Tab autocompletes commands)", cmd)
	}
}

func (a *App) requireCloudData() error {
	if a.cfg.Demo && !a.cfg.SampleData {
		return fmt.Errorf("UI-only mode has no cloud/sample data; run `a1s` for your real account or `a1s --sample-data` for explicit samples")
	}
	return nil
}

func (a *App) quickHelp() {
	fmt.Println()
	fmt.Println(orange + bold + " QUICK ECS COMMANDS" + reset)
	fmt.Printf("  %suptime%s   uptime and load\n", cyan, reset)
	fmt.Printf("  %sdisk%s     df -h /\n", cyan, reset)
	fmt.Printf("  %smemory%s   free -h\n", cyan, reset)
	fmt.Printf("  %sfailed%s   failed systemd services\n", cyan, reset)
	fmt.Printf("  %sports%s    listening TCP/UDP ports\n", cyan, reset)
	fmt.Printf("  %squick%s    run all five checks\n", cyan, reset)
	fmt.Println(dim + "  Select once with `use 1`, then run any shortcut directly." + reset)
}

func (a *App) quickOne(ctx context.Context, id, name string) error {
	commands := map[string]string{
		"uptime": "uptime",
		"disk":   "df -h /",
		"memory": "free -h 2>/dev/null || true",
		"failed": "systemctl --failed --no-pager 2>/dev/null || true",
		"ports":  "ss -tulpn 2>/dev/null | head -30 || true",
	}
	cmd, ok := commands[name]
	if !ok {
		return fmt.Errorf("unknown quick command %q", name)
	}
	return a.runCmd(ctx, id, cmd)
}

func (a *App) handleOllama(ctx context.Context, args []string, rest string) error {
	if len(args) == 0 || strings.EqualFold(args[0], "status") {
		models, err := a.ollamaModels(ctx)
		if err != nil {
			fmt.Printf("%s○ Ollama%s not reachable at %s\n", yellow, reset, a.ollamaURL)
			fmt.Println(dim + "  Start it with: ollama start" + reset)
			return nil
		}
		fmt.Printf("%s● Ollama%s running at %s • %d local model(s)\n", green, reset, a.ollamaURL, len(models))
		if a.ollamaModel != "" {
			fmt.Printf("  active model: %s%s%s\n", orange, a.ollamaModel, reset)
		}
		return nil
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "models", "list", "ls":
		models, err := a.ollamaModels(ctx)
		if err != nil {
			return fmt.Errorf("Ollama is not reachable: %w", err)
		}
		fmt.Println()
		fmt.Println(orange + bold + " OLLAMA MODELS" + reset)
		if len(models) == 0 {
			fmt.Println(dim + "  No local models installed." + reset)
			return nil
		}
		for i, m := range models {
			fmt.Printf("  %s%2d%s  %s\n", orange, i+1, reset, m)
		}
		return nil
	case "use":
		if len(args) < 2 {
			return fmt.Errorf("usage: ollama use <model>")
		}
		model := args[1]
		a.ollamaModel = model
		a.ai.BaseURL = strings.TrimRight(a.ollamaURL, "/") + "/v1"
		a.ai.APIKey = "ollama"
		a.ai.Model = model
		fmt.Printf("%s✓ Ollama AI enabled%s  %s%s%s\n", green, reset, orange, model, reset)
		fmt.Println(dim + "  Now use: ai <question>" + reset)
		return nil
	case "ask":
		q := strings.TrimSpace(strings.TrimPrefix(rest, args[0]))
		if q == "" {
			return fmt.Errorf("usage: ollama ask <question>")
		}
		if a.ollamaModel == "" {
			return fmt.Errorf("choose a model first: ollama use <model>")
		}
		return a.askAI(ctx, q)
	case "start", "serve":
		if _, err := exec.LookPath("ollama"); err != nil {
			return fmt.Errorf("ollama executable not found in PATH")
		}
		if _, err := a.ollamaModels(ctx); err == nil {
			fmt.Println(green + "✓ Ollama is already running." + reset)
			return nil
		}
		cmd := exec.Command("ollama", "serve")
		logf, err := os.OpenFile("/tmp/a1s-ollama.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		cmd.Stdout, cmd.Stderr = logf, logf
		if err := cmd.Start(); err != nil {
			logf.Close()
			return err
		}
		_ = logf.Close()
		fmt.Printf("%s✓ Ollama starting%s  pid=%d  log=/tmp/a1s-ollama.log\n", green, reset, cmd.Process.Pid)
		return nil
	default:
		return fmt.Errorf("usage: ollama [status|models|start|use <model>|ask <question>]")
	}
}

func (a *App) ollamaModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.ollamaURL, "/")+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Ollama returned %s", resp.Status)
	}
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		if strings.TrimSpace(m.Name) != "" {
			models = append(models, m.Name)
		}
	}
	return models, nil
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
		if x.ID == ref || strings.EqualFold(x.Name, ref) {
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
		return "", "", fmt.Errorf("usage: run <ecs-name|instance-id> <command> OR select with use <ecs> then run <command>")
	}
	fields := strings.Fields(rest)
	if len(fields) >= 2 {
		target := fields[0]
		if id, ok := a.resolveInstanceRef(target); ok {
			shell := strings.TrimSpace(strings.TrimPrefix(rest, target))
			return id, shell, nil
		}
	}
	if a.selectedID == "" {
		return "", "", fmt.Errorf("ECS target not found. Example: run test pwd, or use test then run pwd")
	}
	return a.selectedID, rest, nil
}

func (a *App) resolveInstanceRef(ref string) (string, bool) {
	if n, err := strconv.Atoi(ref); err == nil && n >= 1 && n <= len(a.instances) {
		return a.instances[n-1].ID, true
	}
	for _, x := range a.instances {
		if x.ID == ref || strings.EqualFold(x.Name, ref) {
			return x.ID, true
		}
	}
	return "", false
}

func (a *App) ecs(ctx context.Context) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	a.instances = xs
	fmt.Println()
	fmt.Printf(" %s%sECS INSTANCES%s  %s%d resources%s\n", orange, bold, reset, dim, len(xs), reset)
	fmt.Println(dim + " ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────" + reset)
	fmt.Printf(" %s%-3s %-16s %-20s %-9s %-18s %-17s %-15s %-15s%s\n", gray, "#", "NAME", "INSTANCE ID", "STATUS", "TYPE", "OS", "INTERNAL IP", "EXTERNAL IP", reset)
	for i, x := range xs {
		sc := yellow
		if strings.EqualFold(x.Status, "Running") {
			sc = green
		} else if strings.EqualFold(x.Status, "Stopped") {
			sc = red
		}
		marker := " "
		if x.ID == a.selectedID {
			marker = "›"
		}
		fmt.Printf(" %s%s%-3d%s %-16s %s%-20s%s %s%-9s%s %-18s %-17s %-15s %-15s\n", orange, marker, i+1, reset, clip(blank(x.Name, "-"), 16), dim, clip(x.ID, 20), reset, sc, clip(x.Status, 9), reset, clip(x.Type, 18), clip(blank(x.OSName, x.OSType), 17), clip(blank(x.PrivateIP, "-"), 15), clip(blank(x.PublicIP, "-"), 15))
		fmt.Printf("     %sBilling:%s %-14s  %sVPC:%s %s (%s)  %svSwitch:%s %s (%s)  %sZone:%s %s\n", gray, reset, blank(x.ChargeType, "-"), gray, reset, blank(x.VPCName, "-"), blank(x.VPCID, "-"), gray, reset, blank(x.VSwitchName, "-"), blank(x.VSwitchID, "-"), gray, reset, blank(x.Zone, "-"))
	}
	fmt.Println(dim + " ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────" + reset)
	if len(xs) > 0 {
		fmt.Printf(" %sTip:%s run %s%s%s pwd  •  use %s%s%s  •  Tab completes ECS names.\n", dim, reset, cyan, xs[0].Name, reset, cyan, xs[0].Name, reset)
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
	for _, name := range []string{"uptime", "disk", "memory", "failed", "ports"} {
		if err := a.quickOne(ctx, id, name); err != nil {
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
	var xs []model.ECSInstance
	if !(a.cfg.Demo && !a.cfg.SampleData) {
		var err error
		xs, err = a.cloud.ListInstances(ctx)
		if err != nil {
			return err
		}
	}
	ctxText := fmt.Sprintf("Region=%s Currency=%s ECS=%v\nQuestion=%s", a.cfg.Region, a.cfg.Currency, xs, q)
	ans, err := a.ai.Ask(ctx, "You are a read-only Alibaba Cloud operations copilot inside a1s. Use supplied context. Give safe diagnostic guidance. Commands are suggestions only and require explicit user execution.", ctxText)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s%sAI COPILOT%s\n%s\n", orange, bold, reset, ans)
	return nil
}

func (a *App) handleAI(ctx context.Context, args []string, rest string) error {
	if len(args) == 0 || strings.EqualFold(args[0], "providers") {
		fmt.Println()
		fmt.Println(orange + bold + " AI PROVIDERS" + reset)
		fmt.Printf("  %sollama%s             local models on this machine\n", cyan, reset)
		fmt.Printf("  %sopenai-compatible%s  A1S_AI_BASE_URL / vLLM / LiteLLM / DashScope-compatible endpoint\n", cyan, reset)
		if a.ai.Enabled() {
			fmt.Printf("  %sactive%s             %s @ %s\n", green, reset, a.ai.Model, a.ai.BaseURL)
		} else {
			fmt.Printf("  %sactive%s             not configured\n", yellow, reset)
		}
		fmt.Println(dim + "  Use: ai use ollama  →  ai models  →  ai model <number|name>  →  ai ask <question>" + reset)
		return nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "use":
		if len(args) < 2 {
			return fmt.Errorf("usage: ai use ollama|openai-compatible")
		}
		if strings.EqualFold(args[1], "ollama") {
			models, err := a.ollamaModels(ctx)
			if err != nil {
				return fmt.Errorf("Ollama unavailable: %w", err)
			}
			a.ai.BaseURL = strings.TrimRight(a.ollamaURL, "/") + "/v1"
			a.ai.APIKey = "ollama"
			if len(models) > 0 && a.ai.Model == "" {
				a.ai.Model = models[0]
			}
			fmt.Printf("%s✓ AI provider: Ollama%s\n", green, reset)
			return a.showAIModels(ctx)
		}
		return fmt.Errorf("openai-compatible uses A1S_AI_BASE_URL, A1S_AI_MODEL and optional A1S_AI_API_KEY at startup")
	case "models":
		return a.showAIModels(ctx)
	case "model":
		if len(args) < 2 {
			return fmt.Errorf("usage: ai model <number|name>")
		}
		models, err := a.ollamaModels(ctx)
		if err != nil {
			return err
		}
		choice := args[1]
		if n, e := strconv.Atoi(choice); e == nil {
			if n < 1 || n > len(models) {
				return fmt.Errorf("model number must be 1-%d", len(models))
			}
			choice = models[n-1]
		}
		found := false
		for _, m := range models {
			if m == choice {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("model %q is not installed locally", choice)
		}
		a.ai.BaseURL = strings.TrimRight(a.ollamaURL, "/") + "/v1"
		a.ai.APIKey = "ollama"
		a.ai.Model = choice
		a.ollamaModel = choice
		fmt.Printf("%s✓ AI model:%s %s%s%s\n", green, reset, orange, choice, reset)
		return nil
	case "ask":
		q := strings.TrimSpace(strings.TrimPrefix(rest, args[0]))
		if q == "" {
			return fmt.Errorf("usage: ai ask <question>")
		}
		return a.askAI(ctx, q)
	default:
		return a.askAI(ctx, rest)
	}
}

func (a *App) showAIModels(ctx context.Context) error {
	models, err := a.ollamaModels(ctx)
	if err != nil {
		return fmt.Errorf("Ollama unavailable: %w", err)
	}
	fmt.Println()
	fmt.Println(orange + bold + " LOCAL AI MODELS" + reset)
	if len(models) == 0 {
		fmt.Println(dim + "  No Ollama models installed." + reset)
		return nil
	}
	for i, m := range models {
		mark := " "
		if m == a.ai.Model {
			mark = "›"
		}
		fmt.Printf(" %s%s%2d%s  %s\n", orange, mark, i+1, reset, m)
	}
	return nil
}

func (a *App) completions(line string) []string {
	base := []string{"ecs", "ls", "use ", "run ", "metrics ", "quick", "quick list", "uptime", "disk", "memory", "failed", "ports", "doctor", "bill", "report", "ai", "ai providers", "ai use ollama", "ai models", "ai model ", "ai ask ", "ollama status", "ollama models", "currency SAR", "currency USD", "clear", "help", "--help", "quit"}
	trim := strings.TrimSpace(line)
	out := []string{}
	if strings.HasPrefix(trim, "run ") || strings.HasPrefix(trim, "use ") || strings.HasPrefix(trim, "metrics ") {
		parts := strings.Fields(line)
		if len(parts) <= 2 {
			prefix := ""
			if len(parts) > 1 {
				prefix = parts[1]
			}
			cmd := parts[0] + " "
			for _, x := range a.instances {
				if strings.HasPrefix(strings.ToLower(x.Name), strings.ToLower(prefix)) {
					suffix := ""
					if parts[0] == "run" {
						suffix = " "
					}
					out = append(out, cmd+x.Name+suffix)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	for _, c := range base {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(trim)) {
			out = append(out, c)
		}
	}
	return out
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
