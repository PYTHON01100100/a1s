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

var version = "0.1.0-mvp"

func main() {
	cfg := config.FromEnv()
	region := flag.String("region", cfg.Region, "Alibaba Cloud region, e.g. me-central-1")
	profile := flag.String("profile", cfg.Profile, "aliyun CLI profile")
	cur := flag.String("currency", cfg.Currency, "display currency: USD or SAR")
	readonly := flag.Bool("read-only", false, "disable mutating actions such as RunCommand")
	demo := flag.Bool("demo", false, "run with realistic local demo data; no cloud calls")
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
	cfg.Demo = *demo
	if cfg.Currency != "USD" && cfg.Currency != "SAR" {
		fmt.Fprintln(os.Stderr, "currency must be USD or SAR")
		os.Exit(2)
	}
	cloud := aliyun.New(cfg, nil)
	if !cfg.Demo {
		if err := cloud.CheckCLI(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "aliyun CLI check failed:", err, "\nInstall/configure aliyun CLI or try: a1s --demo")
			os.Exit(1)
		}
	}
	app := ui.New(cfg, cloud, ai.New(cfg))
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
