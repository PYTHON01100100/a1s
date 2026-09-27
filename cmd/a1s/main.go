package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"flag"

	"github.com/PYTHON01100100/a1s/internal/aliyun"
	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/ui"
)

var version = "0.7.0"

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
		fmt.Fprintln(out, "  a1s starts anyway and shows how to fix it (or the :configure palette command).")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "OPTIONS")
		flag.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "OTHER MODES")
		fmt.Fprintln(out, "  --demo          UI-only mode. No cloud calls and no data shown.")
		fmt.Fprintln(out, "  --sample-data   Local fake instances for trying a1s without a real account.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "KEYBINDINGS (full-screen UI, k9s/e1s/ec2s-style)")
		fmt.Fprintln(out, "  /                filter the ECS table (text or field=value)")
		fmt.Fprintln(out, "  s / S / x / R / D  start / stop / economic stop / reboot / terminate")
		fmt.Fprintln(out, "  E                run a shell command on the selected instance (no SSH)")
		fmt.Fprintln(out, "  :                command palette: bill, doctor, report, query, metrics, ...")
		fmt.Fprintln(out, "  ctrl-p           switch aliyun-cli profile")
		fmt.Fprintln(out, "  ctrl-r           refresh now (also auto-refreshes on its own)")
		fmt.Fprintln(out, "  ?                help        q / ctrl-c  quit")
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
	// unconfigured aliyun CLI shows a helpful in-app notice (with a
	// :configure palette command to fix it live) instead of a hard exit
	// before the UI even renders.
	cloud := aliyun.New(cfg, nil)
	app := ui.New(cfg, cloud, version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
