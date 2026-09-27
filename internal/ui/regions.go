package ui

import (
	"fmt"
	"strings"
)

// commonRegions is a curated reference of frequently used Alibaba Cloud
// regions, shown by the `:regions` palette command. It is intentionally
// not exhaustive — Alibaba Cloud adds regions over time — so the full,
// current list is always linked alongside it.
var commonRegions = []struct{ ID, Name string }{
	{"me-central-1", "Middle East — Riyadh, Saudi Arabia"},
	{"me-east-1", "Middle East — Dubai, UAE"},
	{"ap-southeast-1", "Asia Pacific — Singapore"},
	{"ap-southeast-3", "Asia Pacific — Kuala Lumpur, Malaysia"},
	{"ap-southeast-5", "Asia Pacific — Jakarta, Indonesia"},
	{"ap-south-1", "Asia Pacific — Mumbai, India"},
	{"ap-northeast-1", "Asia Pacific — Tokyo, Japan"},
	{"cn-hongkong", "China — Hong Kong"},
	{"cn-hangzhou", "China East 1 — Hangzhou"},
	{"cn-shanghai", "China East 2 — Shanghai"},
	{"cn-beijing", "China North 2 — Beijing"},
	{"cn-shenzhen", "China South 1 — Shenzhen"},
	{"eu-central-1", "Europe — Frankfurt, Germany"},
	{"eu-west-1", "Europe — London, UK"},
	{"us-east-1", "US East — Virginia"},
	{"us-west-1", "US West — Silicon Valley"},
}

// regionsText renders the common-regions reference list for the `:regions`
// palette command's result pane.
func regionsText() string {
	var b strings.Builder
	b.WriteString("[::b]Common Alibaba Cloud regions[-:-:-]\n\n")
	for _, r := range commonRegions {
		fmt.Fprintf(&b, "%-16s %s\n", r.ID, r.Name)
	}
	b.WriteString("\nNot listed? Any region ID works — see the full, current list at:\n")
	b.WriteString("https://www.alibabacloud.com/help/en/basics-for-beginners/regions-and-zones\n")
	b.WriteString("\nSwitch with: region <id>")
	return b.String()
}
