package report

import (
	"fmt"
	"strings"
	"time"

	"a1s/internal/currency"
	"a1s/internal/model"
)

func Doctor(instances []model.ECSInstance, points []model.MetricPoint) string {
	running := 0
	stopped := 0
	for _, i := range instances {
		if strings.EqualFold(i.Status, "Running") {
			running++
		} else {
			stopped++
		}
	}
	peak := 0.0
	avg := 0.0
	for _, p := range points {
		avg += p.Average
		if p.Maximum > peak {
			peak = p.Maximum
		}
	}
	if len(points) > 0 {
		avg /= float64(len(points))
	}
	severity := "✓ Healthy"
	if peak >= 90 {
		severity = "⚠ Attention"
	}
	return fmt.Sprintf("Infrastructure Health\n=====================\nECS: %d running, %d non-running\nCPU average: %.1f%%\nCPU peak: %.1f%%\nOverall: %s\n", running, stopped, avg, peak, severity)
}

func Markdown(instances []model.ECSInstance, bill model.BillSummary, displayCurrency string, sarPerUSD float64) string {
	c := currency.Converter{SARPerUSD: sarPerUSD}
	shown := c.Convert(bill.PretaxAmount, bill.Currency, displayCurrency)
	return fmt.Sprintf(`# Alibaba Cloud Operations Report

Generated: %s

## Executive Summary
- ECS instances discovered: %d
- Billing cycle: %s
- Displayed cost: %s %.2f

## ECS Inventory
| Name | ID | Status | Type | Zone | Private IP |
|---|---|---|---|---|---|
%s

## Notes
- Currency conversion is a display preference. Your actual Alibaba Cloud settlement currency remains authoritative.
- Use a1s doctor and a1s metrics for operational checks before making changes.
`, time.Now().Format(time.RFC3339), len(instances), bill.BillingCycle, currency.Symbol(displayCurrency), shown, instanceRows(instances))
}

func instanceRows(xs []model.ECSInstance) string {
	var b strings.Builder
	for _, x := range xs {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", x.Name, x.ID, x.Status, x.Type, x.Zone, x.PrivateIP)
	}
	return b.String()
}
