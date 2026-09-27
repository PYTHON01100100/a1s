package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/model"
)

// blank returns d when s is empty, otherwise s.
func blank(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// orDash is blank(s, "-"), the placeholder ec2s/e1s/k9s use for an unset
// table cell.
func orDash(s string) string { return blank(s, "-") }

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

// billingExpiry reports when a subscription instance will expire, colored
// by urgency. Pay-as-you-go instances never expire, so ExpiredTime is
// ignored for anything that isn't a PrePaid subscription.
func billingExpiry(x model.ECSInstance) (string, string) {
	if !strings.EqualFold(x.ChargeType, "PrePaid") {
		return "-", "gray"
	}
	t, err := parseAlibabaTime(x.ExpiredTime)
	if err != nil {
		return "unknown", "yellow"
	}
	days := int(time.Until(t).Hours() / 24)
	date := t.Format("2006-01-02")
	switch {
	case days < 0:
		return fmt.Sprintf("expired %s", date), "red"
	case days <= 7:
		return fmt.Sprintf("%s (%dd) renew", date, days), "red"
	case days <= 30:
		return fmt.Sprintf("%s (%dd)", date, days), "yellow"
	default:
		return fmt.Sprintf("%s (%dd)", date, days), "gray"
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
		return "GPU", "magenta"
	}
	return "CPU", "gray"
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

// bar renders a CPU percentage as a fixed-width block gauge for the
// metrics palette command's output pane.
func bar(v float64) string {
	n := int(v / 5)
	if n > 20 {
		n = 20
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("█", n) + "[gray]" + strings.Repeat("░", 20-n) + "[-]"
}

// indentOutput prefixes every line of s (or a placeholder when empty) for
// display inside a boxed "output" section of a result pane.
func indentOutput(s, prefix string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return prefix + "[gray](no output)[-]"
	}
	return prefix + strings.ReplaceAll(tview.Escape(s), "\n", "\n"+prefix)
}
