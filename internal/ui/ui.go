package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/currency"
	"github.com/PYTHON01100100/a1s/internal/model"
	"github.com/PYTHON01100100/a1s/internal/report"
)

// Alibaba Cloud brand orange is the default, professional theme. Status
// colors (green/yellow/red) stay constant across themes so meaning never
// changes; only decorative accent colors are swappable via setTheme.
var (
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
	magenta = "\x1b[38;2;186;85;255m" // GPU compute flag; constant across themes like the other status colors

	themeName = "alibaba"
)

type App struct {
	cfg          config.Config
	cloud        *aliyun.Client
	instances    []model.ECSInstance
	selectedID   string
	selectedName string
	filter       string
	editor       *lineEditor
	cloudReady   bool
	accountErr   error
}

func New(cfg config.Config, cloud *aliyun.Client) *App {
	return &App{
		cfg:    cfg,
		cloud:  cloud,
		editor: newLineEditor(),
	}
}

func (a *App) Run(ctx context.Context) error {
	a.header()
	a.dashboard(ctx)
	switch {
	case a.cfg.Demo && !a.cfg.SampleData:
		a.demoNotice()
	case !a.checkAccount(ctx):
		a.accountNotice()
	default:
		if err := a.ecs(ctx); err != nil {
			fmt.Println(yellow + "⚠ Could not load ECS inventory: " + reset + err.Error())
		}
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
	fmt.Printf("  %secs/ls%s inventory      %suse 1%s select ECS      %sstart/stop/reboot%s lifecycle   %sterminate%s delete\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sstop eco%s eco stop     %sfilter%s search          %squery%s lookup            %srun <ecs> <cmd>%s execute\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %swatch [s]%s live refresh %sregions%s pick a region\n", cyan, reset, cyan, reset)
	fmt.Printf("  %squick list%s shortcuts  %sdoctor%s health          %sbill%s costs             %sreport%s markdown\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sconfigure%s add/switch keys %sprofile/region%s switch  %stheme%s alibaba|mono   %scurrency%s USD|SAR\n", cyan, reset, cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %shelp/?%s full help      %sq%s quit\n", cyan, reset, cyan, reset)
	fmt.Println(dim + "  Tip: ↑/↓ history • ←/→ cursor • Tab autocomplete • :commands like k9s" + reset)
}

func (a *App) fullHelp() {
	fmt.Println()
	fmt.Println(orange + bold + " A1S HELP" + reset)
	fmt.Println(dim + " Alibaba Cloud terminal operations • commands are case-insensitive" + reset)
	fmt.Println()
	fmt.Println(white + bold + " RESOURCE BROWSING" + reset)
	fmt.Printf("  %secs%s, %sls%s                    List ECS inventory and network details\n", cyan, reset, cyan, reset)
	fmt.Printf("  %suse <row|name|id>%s          Select an ECS for following commands\n", cyan, reset)
	fmt.Printf("  %sfilter <query>%s             Narrow the list, e.g. filter status=running, filter web\n", cyan, reset)
	fmt.Printf("  %sfilter clear%s               Remove the active filter\n", cyan, reset)
	fmt.Printf("  %squery <text|key=value>%s     Look up instances by IP/VPC/vSwitch/CIDR/ENI without narrowing the table\n", cyan, reset)
	fmt.Printf("  %swatch [seconds]%s            Live auto-refreshing ECS view (default 5s, Ctrl+C to stop)\n", cyan, reset)
	fmt.Printf("  %smetrics [target] [minutes]%s Show CPU metrics (default 60 minutes)\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " INSTANCE LIFECYCLE" + reset)
	fmt.Printf("  %sstart [target]%s             Start a stopped instance\n", cyan, reset)
	fmt.Printf("  %sstop [target]%s              Stop normally (keeps the instance billed and ready)\n", cyan, reset)
	fmt.Printf("  %sstop eco [target]%s          Economic stop; pauses vCPU/memory billing while stopped\n", cyan, reset)
	fmt.Printf("  %sreboot [target]%s            Reboot; add --force for a hard reboot\n", cyan, reset)
	fmt.Printf("  %sterminate [target]%s         Delete an instance permanently (asks for confirmation)\n", cyan, reset)
	fmt.Println(dim + "  target is a row number, ECS name, or instance ID; omit it to use the selected ECS." + reset)
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
	fmt.Println(white + bold + " ACCOUNT & APPEARANCE" + reset)
	fmt.Printf("  %sconfigure%s                  Add/update a profile: name, region (from a list), then keys\n", cyan, reset)
	fmt.Printf("  %sconfigure <profile>%s        Same, for a specific named profile (e.g. uat, client1)\n", cyan, reset)
	fmt.Printf("  %sregions%s                    Show a reference list of common region IDs\n", cyan, reset)
	fmt.Printf("  %sprofiles%s                   List aliyun-cli profiles on this machine\n", cyan, reset)
	fmt.Printf("  %sprofile <name>%s             Switch the active aliyun-cli profile/region\n", cyan, reset)
	fmt.Printf("  %sregion <region-id>%s         Switch the active region\n", cyan, reset)
	fmt.Printf("  %stheme <alibaba|mono>%s       Switch the color theme\n", cyan, reset)
	fmt.Println()
	fmt.Println(white + bold + " TERMINAL" + reset)
	fmt.Printf("  %s↑ / ↓%s  command history     %s← / →%s  move cursor     %sTab%s  autocomplete\n", cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %sdashboard%s redraw home      %shelp <command>%s detailed help     %sq%s quit\n", cyan, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %s:<command>%s k9s-style command palette, e.g. :ecs or :bill\n", cyan, reset)
	fmt.Println()
	fmt.Println(dim + " Examples: run test pwd • stop eco test • filter status=running • watch 10 • help stop" + reset)
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
		fmt.Println("  Summarize infrastructure health from ECS inventory and CPU metrics.")
		fmt.Println("\n  Usage:\n    doctor\n    doctor <instance-id>")
	case "bill", "$":
		fmt.Println("  Show account billing for a billing cycle in settlement currency and selected display currency.")
		fmt.Println("\n  Usage:\n    bill\n    bill YYYY-MM\n\n  Example:\n    bill 2026-08")
	case "report":
		fmt.Println("  Generate a Markdown report from current cloud inventory and billing data.")
		fmt.Println("\n  Usage:\n    report [output.md]\n\n  Example:\n    report ops-report.md")
	case "start":
		fmt.Println("  Start a stopped ECS instance.")
		fmt.Println("\n  Usage:\n    start [row|ecs-name|instance-id]\n\n  Example:\n    use test\n    start\n    start web-02")
	case "stop", "p":
		fmt.Println("  Stop an ECS instance. Normal stop keeps it billed and ready for a fast restart.")
		fmt.Println("  Economic stop (\"eco\") pauses vCPU/memory billing for the instance while it stays stopped;")
		fmt.Println("  this matches the Alibaba Cloud console's economical mode and applies to eligible VPC pay-as-you-go instances.")
		fmt.Println("\n  Usage:\n    stop [target]                normal stop\n    stop eco [target]            economic stop (billing paused)\n    stop [eco] [target] --force  force stop")
		fmt.Println("\n  Examples:\n    use test\n    stop\n    stop eco web-02\n    stop --force i-xxxxxxxx")
		if a.cfg.ReadOnly {
			fmt.Println("\n  Note: instance actions are disabled because a1s is running with --read-only.")
		}
	case "reboot":
		fmt.Println("  Reboot an ECS instance.")
		fmt.Println("\n  Usage:\n    reboot [target]\n    reboot [target] --force\n\n  Example:\n    use test\n    reboot")
	case "terminate", "delete":
		fmt.Println("  Permanently delete an ECS instance. This is irreversible and asks for confirmation.")
		fmt.Println("\n  Usage:\n    terminate [target]\n    terminate [target] --force\n\n  Example:\n    terminate web-02")
	case "filter", "find":
		fmt.Println("  Narrow the ECS list and working set to matching instances.")
		fmt.Println("  Plain text matches name, ID, status, type, zone, billing, IPs, VPC/vSwitch, CIDR, and ENI.")
		fmt.Println("  key=value matches one field: status, type, compute, zone, region, name, id, ip, internalip,")
		fmt.Println("  externalip, vpc, vpcname, vswitch, vswitchname, cidr, vpccidr, vswitchcidr, eni, billing.")
		fmt.Println("\n  Usage:\n    filter <text>\n    filter status=running\n    filter zone=me-central-1a\n    filter clear")
		fmt.Println(dim + "  Looking something up without narrowing the table? Use `query` instead." + reset)
	case "query":
		fmt.Println("  Look up ECS instances by any field without narrowing the table (unlike `filter`).")
		fmt.Println("  Prints a full network/billing card per match: IP, VPC/vSwitch (with CIDR), zone, ENI, billing.")
		fmt.Println("  Accepts the same plain-text or key=value syntax as `filter`.")
		fmt.Println("\n  Usage:\n    query <text>\n    query <key>=<value>")
		fmt.Println("\n  Examples:\n    query 10.0.0.11          # which instance has this IP\n    query vpc=vpc-demo01     # every instance in this VPC\n    query vswitch=vsw-demo01\n    query cidr=10.0.0.0/24\n    query eni=eni-demo-web01")
	case "watch", "w":
		fmt.Println("  Auto-refresh the ECS inventory at a fixed interval, like a live dashboard.")
		fmt.Println("\n  Usage:\n    watch\n    watch <seconds>\n\n  Example:\n    watch 10\n\n  Stop with Ctrl+C.")
	case "dashboard":
		fmt.Println("  Redraw the K9s-inspired context dashboard and reload ECS inventory.")
		fmt.Println("\n  Usage:\n    dashboard\n    :dashboard")
	case "configure", "keys":
		fmt.Println("  Add or update an aliyun-cli profile: pick a name (e.g. uat, client1, client2),")
		fmt.Println("  pick a region from a reference list (or type any region ID), then enter the")
		fmt.Println("  AccessKey ID/Secret via the official `aliyun configure` prompt (paste works normally).")
		fmt.Println("  a1s never stores, logs, or otherwise touches the keys themselves.")
		fmt.Println("\n  Usage:\n    configure              prompts for a profile name, then the region and keys\n    configure <profile>    configure/add that specific named profile")
	case "regions":
		fmt.Println("  Show a reference list of common Alibaba Cloud region IDs.")
		fmt.Println("\n  Usage:\n    regions")
	case "profile":
		fmt.Println("  Switch the active aliyun-cli profile (and its default region, if set).")
		fmt.Println("\n  Usage:\n    profiles              list available profiles\n    profile <name>        switch profile\n    configure <name>      add a profile that doesn't exist yet")
	case "region":
		fmt.Println("  Switch the active Alibaba Cloud region without changing profile.")
		fmt.Println("\n  Usage:\n    region <region-id>\n    regions               show a reference list of region IDs\n\n  Example:\n    region me-central-1")
	case "theme":
		fmt.Println("  Switch the a1s color theme.")
		fmt.Println("\n  Usage:\n    theme               show the active theme\n    theme alibaba        Alibaba Cloud orange (default)\n    theme mono           low-color, accessible theme")
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
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, ":") {
		line = strings.TrimSpace(strings.TrimPrefix(line, ":"))
	}
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

	// Help is available globally and per command: --help, help, help run, run --help, stop --help, etc.
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
	case "clear", "home", "dashboard":
		a.header()
		a.dashboard(ctx)
		if a.cfg.Demo && !a.cfg.SampleData {
			a.demoNotice()
			return nil
		}
		if !a.checkAccount(ctx) {
			a.accountNotice()
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
	case "filter", "find":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		if len(parts) < 2 || strings.EqualFold(parts[1], "clear") {
			a.filter = ""
			fmt.Println(green + "✓ filter cleared" + reset)
			return a.ecs(ctx)
		}
		a.filter = rest
		return a.ecs(ctx)
	case "query":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		if rest == "" {
			return fmt.Errorf("usage: query <text|key=value>  e.g. query 10.0.0.11, query vpc=vpc-demo01, query eni-demo-web01")
		}
		return a.query(rest)
	case "watch", "w":
		if err := a.requireCloudData(); err != nil {
			return err
		}
		seconds := 5
		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				seconds = n
			}
		}
		return a.watch(ctx, seconds)
	case "start":
		return a.lifecycleCommand(ctx, "start", parts[1:])
	case "stop":
		return a.lifecycleCommand(ctx, "stop", parts[1:])
	case "reboot":
		return a.lifecycleCommand(ctx, "reboot", parts[1:])
	case "terminate", "delete":
		return a.lifecycleCommand(ctx, "terminate", parts[1:])
	case "profiles":
		return a.showProfiles()
	case "profile":
		if len(parts) < 2 {
			return fmt.Errorf("usage: profile <name>; run `profiles` to list available profiles")
		}
		return a.switchProfile(ctx, parts[1])
	case "region":
		if len(parts) < 2 {
			return fmt.Errorf("usage: region <region-id>")
		}
		return a.switchRegion(ctx, parts[1])
	case "regions":
		a.showRegions()
		return nil
	case "configure", "keys":
		return a.configureProfile(ctx, parts[1:])
	case "theme":
		if len(parts) < 2 {
			fmt.Printf("%sactive theme%s %s%s%s  %savailable: alibaba, mono%s\n", gray, reset, orange, themeName, reset, dim, reset)
			return nil
		}
		if err := setTheme(parts[1]); err != nil {
			return err
		}
		fmt.Printf("%s✓ theme set to%s %s%s%s\n", green, reset, orange, themeName, reset)
		return nil
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
	if !a.cfg.SampleData && !a.cloudReady {
		return fmt.Errorf("no Alibaba Cloud account is configured; run `configure` (or `aliyun configure` in another terminal), then `dashboard` to retry")
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

// resolveTargetOrSelected consumes a leading target argument when it matches
// a known instance; otherwise it falls back to the currently selected ECS.
// It returns the remaining args so lifecycle commands can still parse flags
// like --force after the target.
func (a *App) resolveTargetOrSelected(args []string) (id, label string, remaining []string, err error) {
	if len(args) > 0 {
		if rid, ok := a.resolveInstanceRef(args[0]); ok {
			return rid, a.labelFor(rid), args[1:], nil
		}
	}
	if a.selectedID != "" {
		return a.selectedID, blank(a.selectedName, a.selectedID), args, nil
	}
	return "", "", nil, fmt.Errorf("select an ECS first with `use <row-number>`, or specify a target: <row|name|instance-id>")
}

func (a *App) labelFor(id string) string {
	for _, x := range a.instances {
		if x.ID == id {
			return blank(x.Name, x.ID)
		}
	}
	return id
}

func (a *App) ecs(ctx context.Context) error {
	xs, err := a.cloud.ListInstances(ctx)
	if err != nil {
		return err
	}
	if a.filter != "" {
		xs = filterInstances(xs, a.filter)
	}
	a.instances = xs
	fmt.Println()
	if a.filter != "" {
		fmt.Printf(" %s%sECS INSTANCES%s  %s%d resources%s  %sfilter:%s %s%q%s\n", orange, bold, reset, dim, len(xs), reset, gray, reset, orange, a.filter, reset)
	} else {
		fmt.Printf(" %s%sECS INSTANCES%s  %s%d resources%s\n", orange, bold, reset, dim, len(xs), reset)
	}
	header := fmt.Sprintf(" %-3s %-16s %-20s %-9s %-22s %-7s %-24s %-15s %-15s %-36s %-40s %-30s %-14s %-24s",
		"#", "NAME", "INSTANCE ID", "STATUS", "TYPE", "COMPUTE", "OS", "INTERNAL IP", "EXTERNAL IP", "VPC", "VSWITCH", "ZONE (REGION)", "BILLING", "EXPIRES")
	rule := dim + " " + strings.Repeat("─", len(header)-1) + reset
	fmt.Println(rule)
	fmt.Println(gray + header + reset)
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
		expireText, expireColor := billingExpiry(x)
		kindText, kindColor := instanceKind(x.Type)
		fmt.Printf(" %s%s%-3d%s %-16s %s%-20s%s %s%-9s%s %-22s %s %-24s %-15s %-15s %-36s %-40s %-30s %s %s\n",
			orange, marker, i+1, reset,
			clip(blank(x.Name, "-"), 16),
			dim, clip(x.ID, 20), reset,
			sc, clip(x.Status, 9), reset,
			clip(x.Type, 22),
			padColor(kindText, kindColor, 7),
			clip(blank(x.OSName, x.OSType), 24),
			clip(blank(x.PrivateIP, "-"), 15),
			clip(blank(x.PublicIP, "-"), 15),
			clip(pairLabelCIDR(x.VPCName, x.VPCID, x.VPCCIDR), 36),
			clip(pairLabelCIDR(x.VSwitchName, x.VSwitchID, x.VSwitchCIDR), 40),
			clip(zoneLabel(x.Zone), 30),
			padPlain(billingLabel(x.ChargeType), 14),
			padColor(expireText, expireColor, 24))
	}
	fmt.Println(rule)
	if len(xs) > 0 {
		fmt.Printf(" %sTip:%s run %s%s%s pwd  •  use %s%s%s  •  Tab completes ECS names.\n", dim, reset, cyan, xs[0].Name, reset, cyan, xs[0].Name, reset)
	}
	return nil
}

// billingLabel translates Alibaba Cloud's raw ChargeType into the wording
// used in the console: PostPaid is pay-as-you-go, PrePaid is a subscription
// that renews/expires.
func billingLabel(chargeType string) string {
	switch chargeType {
	case "PostPaid":
		return "Pay-As-You-Go"
	case "PrePaid":
		return "Subscription"
	case "":
		return "-"
	default:
		return chargeType
	}
}

// billingExpiry reports when a subscription instance will expire, colored by
// urgency. Pay-as-you-go instances never expire, so ExpiredTime is ignored
// for anything that isn't a PrePaid subscription.
func billingExpiry(x model.ECSInstance) (string, string) {
	if !strings.EqualFold(x.ChargeType, "PrePaid") {
		return "-", gray
	}
	t, err := parseAlibabaTime(x.ExpiredTime)
	if err != nil {
		return "unknown", yellow
	}
	days := int(time.Until(t).Hours() / 24)
	date := t.Format("2006-01-02")
	switch {
	case days < 0:
		return fmt.Sprintf("expired %s", date), red
	case days <= 7:
		return fmt.Sprintf("%s (%dd) ⚠ renew", date, days), red
	case days <= 30:
		return fmt.Sprintf("%s (%dd)", date, days), yellow
	default:
		return fmt.Sprintf("%s (%dd)", date, days), gray
	}
}

// instanceKind flags whether an ECS instance type is GPU-accelerated or a
// plain CPU instance, based on Alibaba Cloud's instance family naming
// convention embedded in the type string (e.g. ecs.gn7i.*, ecs.vgn6i.*,
// ecs.ebmgn7.*, ecs.ga1.* are GPU families; ecs.g8i.*, ecs.c8i.*, ecs.r8i.*
// and similar are CPU-only).
func instanceKind(instanceType string) (string, string) {
	family := strings.ToLower(instanceType)
	if parts := strings.SplitN(family, ".", 3); len(parts) >= 2 {
		family = parts[1]
	}
	if isGPUFamily(family) {
		return "GPU", magenta
	}
	return "CPU", gray
}

func isGPUFamily(family string) bool {
	switch {
	case strings.HasPrefix(family, "ebmgn"):
		return true
	case strings.HasPrefix(family, "vgn"):
		return true
	case strings.HasPrefix(family, "gn"):
		return true
	case len(family) >= 3 && strings.HasPrefix(family, "ga") && family[2] >= '0' && family[2] <= '9':
		return true
	}
	return false
}

// pairLabel formats a named resource as "name (id)", falling back to just
// the id (or "-") when the friendly name isn't available.
func pairLabel(name, id string) string {
	n := blank(name, "-")
	if id == "" {
		return n
	}
	return fmt.Sprintf("%s (%s)", n, id)
}

// pairLabelCIDR is pairLabel with the resource's CIDR block appended, e.g.
// "demo-vpc (vpc-demo01) 10.0.0.0/16".
func pairLabelCIDR(name, id, cidr string) string {
	base := pairLabel(name, id)
	if cidr == "" {
		return base
	}
	return base + " " + cidr
}

// zoneLabel shows a zone alongside its region, e.g. "me-central-1a
// (me-central-1)". Alibaba Cloud zone IDs are the region ID plus one
// trailing letter, so the region is derived rather than requiring a
// separate API call.
func zoneLabel(zone string) string {
	zone = strings.TrimSpace(zone)
	if zone == "" {
		return "-"
	}
	region := regionFromZone(zone)
	if region == "" || region == zone {
		return zone
	}
	return fmt.Sprintf("%s (%s)", zone, region)
}

func regionFromZone(zone string) string {
	if len(zone) < 2 {
		return ""
	}
	last := zone[len(zone)-1]
	if last < 'a' || last > 'z' {
		return ""
	}
	return zone[:len(zone)-1]
}

func parseAlibabaTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range []string{"2006-01-02T15:04Z", time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", s)
}

// padPlain right-pads plain (non-colored) text to a fixed width.
func padPlain(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padColor pads the plain text to width first, then wraps the whole
// fixed-width field in color. Coloring before padding would make %-Ns count
// the invisible ANSI escape bytes as part of the width and break alignment.
func padColor(s, color string, width int) string {
	return color + padPlain(s, width) + reset
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
	if err := os.WriteFile(path, []byte(md), 0644); err != nil {
		return err
	}
	fmt.Printf("%s✓ Report written to%s %s\n", green, reset, path)
	return nil
}

func (a *App) completions(line string) []string {
	base := []string{
		"ecs", "ls", "use ", "run ", "metrics ", "quick", "quick list",
		"uptime", "disk", "memory", "failed", "ports", "doctor", "bill", "report",
		"start ", "stop ", "stop eco ", "reboot ", "terminate ", "delete ",
		"filter ", "filter clear", "find ", "query ", "watch", "watch ",
		"profiles", "profile ", "region ", "regions", "configure", "configure ", "keys", "theme", "theme alibaba", "theme mono",
		"dashboard", "currency SAR", "currency USD",
		"clear", "help", "--help", "quit",
		":ecs", ":run ", ":metrics ", ":bill", ":stop ", ":dashboard", ":help",
	}
	trim := strings.TrimSpace(line)
	out := []string{}
	palette := strings.HasPrefix(trim, ":")
	matchTrim := strings.TrimPrefix(trim, ":")

	nameCompleted := []string{"run ", "use ", "metrics ", "start ", "stop ", "reboot ", "terminate ", "delete "}
	for _, p := range nameCompleted {
		if !strings.HasPrefix(matchTrim, p) {
			continue
		}
		work := strings.TrimPrefix(strings.TrimSpace(line), ":")
		parts := strings.Fields(work)
		if len(parts) == 0 || len(parts) > 2 {
			break
		}
		prefix := ""
		if len(parts) > 1 {
			prefix = parts[1]
		}
		cmd := parts[0] + " "
		if palette {
			cmd = ":" + cmd
		}
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
		break
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
