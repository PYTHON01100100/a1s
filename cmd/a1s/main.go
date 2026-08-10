package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"a1s/internal/ai"
	"a1s/internal/aliyun"
	"a1s/internal/config"
	"a1s/internal/ui"
)

var version = "0.3.0-ui"

func main() {
	cfg := config.FromEnv()

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintln(out, "a1s — Alibaba Cloud terminal operations")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "USAGE")
		fmt.Fprintln(out, "  a1s [options]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "ACCOUNT")
		fmt.Fprintln(out, "  By default a1s auto-detects the current aliyun-cli profile and region.")
		fmt.Fprintln(out, "  Run without --demo to see ONLY resources returned by your Alibaba Cloud account.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "OPTIONS")
		flag.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "MODES")
		fmt.Fprintln(out, "  --demo          UI-only mode. No cloud calls and NO fake resources.")
		fmt.Fprintln(out, "  --sample-data   Explicit local sample resources for screenshots/testing.")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "EXAMPLES")
		fmt.Fprintln(out, "  a1s --currency SAR")
		fmt.Fprintln(out, "  a1s --read-only --currency SAR")
		fmt.Fprintln(out, "  a1s --demo")
		fmt.Fprintln(out, "  a1s --sample-data --currency SAR")
		fmt.Fprintln(out, "  a1s --version")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "INTERACTIVE")
		fmt.Fprintln(out, "  run <ecs-name> <command>      target ECS by name")
		fmt.Fprintln(out, "  use <row|name|instance-id>    select ECS")
		fmt.Fprintln(out, "  ai providers                  list AI providers")
		fmt.Fprintln(out, "  ai use ollama                 use local Ollama")
		fmt.Fprintln(out, "  ai models                     list installed models")
		fmt.Fprintln(out, "  ai model <number|name>        choose model")
		fmt.Fprintln(out, "  ↑/↓ history, ←/→ cursor, Tab autocomplete (Linux terminals)")
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

	cloud := aliyun.New(cfg, nil)
	if !cfg.Demo {
		if err := cloud.CheckCLI(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "aliyun CLI check failed:", err)
			fmt.Fprintln(os.Stderr, "Install/configure aliyun CLI, or use `a1s --demo` for UI-only mode.")
			os.Exit(1)
		}
	}

	app := ui.New(cfg, cloud, ai.New(cfg))
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
