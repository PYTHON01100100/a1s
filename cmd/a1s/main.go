package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/ui"
)

var version = "0.6.0"

func main() {
	cfg := config.FromEnv()

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "a1s — Alibaba Cloud terminal operations")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "USAGE")
		fmt.Fprintln(out, "  a1s")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "ACCOUNT")
		fmt.Fprintln(out, "  a1s auto-detects the current aliyun-cli profile and region and shows only")
		fmt.Fprintln(out, "  what your Alibaba Cloud account returns. If no account is configured yet,")
		fmt.Fprintln(out, "  a1s starts anyway and shows how to fix it (or run `configure` inside a1s).")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "OPTIONS")
		flag.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "OTHER MODES")
		fmt.Fprintln(out, "  --demo          UI-only mode. No cloud calls and no data shown.")
		fmt.Fprintln(out, "  --sample-data   Local fake instances for trying a1s without a real account.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "INTERACTIVE")
		fmt.Fprintln(out, "  ecs                           list ECS inventory")
		fmt.Fprintln(out, "  use <row|name|instance-id>    select ECS")
		fmt.Fprintln(out, "  start / stop / reboot         instance lifecycle for the selected ECS")
		fmt.Fprintln(out, "  stop eco <target>             stop and pause vCPU/memory billing")
		fmt.Fprintln(out, "  terminate <target>            delete an instance (confirmation required)")
		fmt.Fprintln(out, "  filter <query>                narrow the ECS list, e.g. filter status=running")
		fmt.Fprintln(out, "  watch [seconds]                live auto-refreshing ECS view")
		fmt.Fprintln(out, "  run <ecs-name> <command>      target ECS by name")
		fmt.Fprintln(out, "  profile <name> / region <id>  switch aliyun-cli profile or region")
		fmt.Fprintln(out, "  configure [profile]           add/update AccessKey ID, Secret, and region")
		fmt.Fprintln(out, "  currency USD|SAR              change the display currency")
		fmt.Fprintln(out, "  theme <alibaba|mono>          switch the color theme")
		fmt.Fprintln(out, "  :ecs / :run / :bill ...       k9s-style command palette")
		fmt.Fprintln(out, "  ↑/↓ history, ←/→ cursor, Home/End, Delete, Tab autocomplete (Linux terminals)")
		fmt.Fprintln(out, "  --help or help             full interactive help")
		fmt.Fprintln(out, "  <command> --help           detailed help for any interactive command")
	}

	region := flag.String("region", cfg.Region, "Alibaba Cloud region, e.g. me-central-1")
	profile := flag.String("profile", cfg.Profile, "aliyun CLI profile")
	cur := flag.String("currency", cfg.Currency, "display currency: USD or SAR")
	readonly := flag.Bool("read-only", false, "disable mutating cloud actions such as RunCommand")
	demo := flag.Bool("demo", false, "UI-only mode; no cloud calls and no fake resources")
	sampleData := flag.Bool("sample-data", false, "use explicit local sample resources for testing")
	ver := flag.Bool("version", false, "print version")

	flag.Parse()

	if *ver {
		fmt.Println("a1s", version)
		return
	}

	cfg.Region = *region
	cfg.Profile = *profile
	cfg.Currency = strings.ToUpper(*cur)
	cfg.ReadOnly = *readonly
	cfg.Demo = *demo || *sampleData
	cfg.SampleData = *sampleData

	if cfg.Currency != "USD" && cfg.Currency != "SAR" {
		fmt.Fprintln(os.Stderr, "currency must be USD or SAR")
		os.Exit(2)
	}

	// Account/CLI availability is checked inside the app so a missing or
	// unconfigured aliyun CLI shows a helpful in-app notice (with a `configure`
	// command to fix it live) instead of a hard exit before the UI even renders.
	cloud := aliyun.New(cfg, nil)
	app := ui.New(cfg, cloud)
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
