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

type App struct {
	cfg   config.Config
	cloud *aliyun.Client
	ai    *ai.Client
}

func New(cfg config.Config, cloud *aliyun.Client, aic *ai.Client) *App {
	return &App{cfg: cfg, cloud: cloud, ai: aic}
}

func (a *App) Run(ctx context.Context) error {
	a.header()
	a.help()
	s := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\na1s> ")
		if !s.Scan() {
			return s.Err()
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if line == "q" || line == "quit" || line == "exit" {
			return nil
		}
		if err := a.handle(ctx, line); err != nil {
			fmt.Println("ERROR:", err)
		}
	}
}
func (a *App) header() {
	mode := "LIVE"
	if a.cfg.Demo {
		mode = "DEMO"
	}
	ro := "RW"
	if a.cfg.ReadOnly {
		ro = "READ-ONLY"
	}
	fmt.Printf("\033[2J\033[H a1s — AI-powered Alibaba Cloud terminal operations [%s/%s]\n Region: %s | Profile: %s | Currency: %s\n", mode, ro, blank(a.cfg.Region, "profile default"), blank(a.cfg.Profile, "default"), a.cfg.Currency)
}
func (a *App) help() {
	fmt.Println(" Commands: ecs | metrics <instance-id> [minutes] | run <instance-id> <shell> | quick <instance-id> | doctor [instance-id] | bill [YYYY-MM] | price <instance-type> [Hour|Month|Year] | report [file.md] | ai <question> | currency USD|SAR | help | quit")
}
func (a *App) handle(ctx context.Context, line string) error {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))
	switch cmd {
	case "help", "?":
		a.help()
		return nil
	case "ecs":
		return a.ecs(ctx)
	case "metrics":
		if len(parts) < 2 {
			return fmt.Errorf("usage: metrics <instance-id> [minutes]")
		}
		mins := 60
		if len(parts) > 2 {
			if n, e := strconv.Atoi(parts[2]); e == nil {
				mins = n
			}
		}
		return a.metrics(ctx, parts[1], mins)
	case "run":
		if len(parts) < 3 {
			return fmt.Errorf("usage: run <instance-id> <command>")
		}
		shell := strings.TrimSpace(strings.TrimPrefix(rest, parts[1]))
		return a.runCmd(ctx, parts[1], shell)
	case "quick":
		if len(parts) < 2 {
			return fmt.Errorf("usage: quick <instance-id>")
		}
		return a.quick(ctx, parts[1])
	case "doctor":
		id := ""
		if len(parts) > 1 {
			id = parts[1]
		}
		return a.doctor(ctx, id)
	case "bill":
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
	case "ai":
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
		fmt.Println("Display currency set to", c)
		return nil
	default:
		return fmt.Errorf("unknown command %q; type help", cmd)
	}
}
func (a *App) ecs(ctx context.Context) error {
	xs, e := a.cloud.ListInstances(ctx)
	if e != nil {
		return e
	}
	fmt.Printf("%-18s %-22s %-10s %-18s %-15s %-15s\n", "NAME", "ID", "STATUS", "TYPE", "ZONE", "PRIVATE IP")
	for _, x := range xs {
		fmt.Printf("%-18s %-22s %-10s %-18s %-15s %-15s\n", clip(x.Name, 18), clip(x.ID, 22), x.Status, clip(x.Type, 18), x.Zone, x.PrivateIP)
	}
	return nil
}
func (a *App) metrics(ctx context.Context, id string, mins int) error {
	ps, e := a.cloud.CPU(ctx, id, mins)
	if e != nil {
		return e
	}
	if len(ps) == 0 {
		fmt.Println("No metric points returned.")
		return nil
	}
	fmt.Println("CPUUtilization")
	fmt.Println("TIME                  AVG      MAX      GRAPH")
	for _, p := range ps {
		fmt.Printf("%-20s %6.1f%% %6.1f%%  %s\n", time.UnixMilli(p.Timestamp).Format("15:04:05"), p.Average, p.Maximum, bar(p.Average))
	}
	return nil
}
func (a *App) runCmd(ctx context.Context, id, cmd string) error {
	fmt.Printf("Running on %s via Cloud Assistant: %s\n", id, cmd)
	r, e := a.cloud.RunCommand(ctx, id, cmd)
	if e != nil {
		return e
	}
	fmt.Printf("Status: %s | Exit: %d | InvokeId: %s\n--- output ---\n%s\n--------------\n", r.Status, r.ExitCode, r.InvokeID, r.Output)
	return nil
}
func (a *App) quick(ctx context.Context, id string) error {
	for _, q := range []string{"uptime", "df -h /", "free -h 2>/dev/null || true", "systemctl --failed --no-pager 2>/dev/null || true", "ss -tulpn 2>/dev/null | head -30 || true"} {
		fmt.Println("\n$", q)
		if e := a.runCmd(ctx, id, q); e != nil {
			fmt.Println("  ", e)
		}
	}
	return nil
}
func (a *App) doctor(ctx context.Context, id string) error {
	xs, e := a.cloud.ListInstances(ctx)
	if e != nil {
		return e
	}
	var m []model.MetricPoint
	if id != "" {
		m, _ = a.cloud.CPU(ctx, id, 60)
	} else if len(xs) > 0 {
		m, _ = a.cloud.CPU(ctx, xs[0].ID, 60)
	}
	text := report.Doctor(xs, m)
	fmt.Print(text)
	if a.ai.Enabled() {
		ans, err := a.ai.Ask(ctx, "You are an Alibaba Cloud SRE. Analyze only the supplied operational data. Be concise, separate evidence from suggestions, and never claim a command was run unless output is present.", text)
		if err == nil {
			fmt.Println("\nAI Analysis\n-----------\n" + ans)
		}
	}
	return nil
}
func (a *App) bill(ctx context.Context, cycle string) error {
	b, e := a.cloud.Bill(ctx, cycle)
	if e != nil {
		return e
	}
	c := currency.Converter{SARPerUSD: a.cfg.SARPerUSD}
	shown := c.Convert(b.PretaxAmount, b.Currency, a.cfg.Currency)
	fmt.Printf("Billing cycle: %s\nSettlement/API amount: %s %.2f\nDisplay amount: %s %.2f\n", b.BillingCycle, b.Currency, b.PretaxAmount, currency.Symbol(a.cfg.Currency), shown)
	if b.Currency != a.cfg.Currency {
		fmt.Printf("Display FX: 1 USD = %.4f SAR (configurable via A1S_SAR_PER_USD)\n", a.cfg.SARPerUSD)
	}
	return nil
}
func (a *App) makeReport(ctx context.Context, path string) error {
	xs, e := a.cloud.ListInstances(ctx)
	if e != nil {
		return e
	}
	b, e := a.cloud.Bill(ctx, time.Now().Format("2006-01"))
	if e != nil {
		return e
	}
	md := report.Markdown(xs, b, a.cfg.Currency, a.cfg.SARPerUSD)
	if a.ai.Enabled() {
		if extra, err := a.ai.Ask(ctx, "You are an Alibaba Cloud operations analyst. Produce a short executive analysis of the following report. Do not invent metrics.", md); err == nil {
			md += "\n## AI Analysis\n\n" + extra + "\n"
		}
	}
	if e := os.WriteFile(path, []byte(md), 0644); e != nil {
		return e
	}
	fmt.Println("Report written to", path)
	return nil
}
func (a *App) askAI(ctx context.Context, q string) error {
	xs, e := a.cloud.ListInstances(ctx)
	if e != nil {
		return e
	}
	ctxText := fmt.Sprintf("Region=%s Currency=%s ECS=%v\nQuestion=%s", a.cfg.Region, a.cfg.Currency, xs, q)
	ans, e := a.ai.Ask(ctx, "You are a read-only Alibaba Cloud operations copilot inside a1s. Use supplied context. Give safe diagnostic guidance. Commands are suggestions only and require explicit user execution.", ctxText)
	if e != nil {
		return e
	}
	fmt.Println(ans)
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
	return strings.Repeat("█", n) + strings.Repeat("░", 20-n)
}
