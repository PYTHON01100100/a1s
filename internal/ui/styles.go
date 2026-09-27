package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Color palette, modeled on e1s/k9s/ec2s: a fixed dark background so a1s
// looks the same regardless of the terminal's own theme, with Alibaba
// Cloud's brand orange standing in for e1s/ec2s's teal borders and cyan
// standing in for their fuchsia accents. Status colors (green/yellow/red)
// are intentionally NOT part of the swappable theme below — they never
// change, so instance state always reads the same way regardless of theme.
var (
	colorBackground = tcell.NewRGBColor(0x1a, 0x1a, 0x1a)
	colorForeground = tcell.NewRGBColor(0xeb, 0xee, 0xf2)

	colorGreen   = tcell.NewRGBColor(0x48, 0xc7, 0x74)
	colorYellow  = tcell.NewRGBColor(0xff, 0xc4, 0x3d)
	colorRed     = tcell.NewRGBColor(0xff, 0x5c, 0x5c)
	colorGray    = tcell.NewRGBColor(0x96, 0x9e, 0xa8)
	colorMagenta = tcell.NewRGBColor(0xba, 0x55, 0xff) // GPU compute flag

	// The theme-swappable accents. "alibaba" (default) uses Alibaba Cloud's
	// orange; "mono" is a low-color, accessible theme. Both are tag strings
	// (usable directly in tview's "[color]" markup) as well as valid input
	// to tcell.GetColor for primitives that need a tcell.Color directly
	// (borders, selection highlight, ...).
	orange    = "#ff6a00"
	orange2   = "#ff8c00"
	accent    = "#42d3ff"
	themeName = "alibaba"

	// Format strings, rebuilt by setTheme() whenever the accents above
	// change.
	infoTitleFmt        string
	infoItemFmt         string
	infoKeyFmt          string
	tableTitleFmt       string
	footerChipFmt       string
	footerChipActiveFmt string
	footerAppFmt        string
)

func init() {
	rebuildFormats()
}

// rebuildFormats regenerates every tview markup format string from the
// current orange/orange2/accent theme vars. Called once at init and again
// every time setTheme changes those vars.
func rebuildFormats() {
	infoTitleFmt = " [" + orange + "]info([" + accent + "::b]%s[" + orange + ":-:-]) "
	infoItemFmt = " %s:[" + orange + "::b] %s "
	infoKeyFmt = " [" + accent + "::b]<%s> [green:-:-]%s "
	tableTitleFmt = " [" + orange + "::-]<[" + accent + "::b]%s[" + orange + "::-]>[" + orange + "::b]%s[" + orange + "::-]([" + accent + "::b]%d[" + orange + "::-]) "
	footerChipFmt = "[black:gray:] %s [-:-:-]"
	footerChipActiveFmt = "[black:" + orange + ":b] %s [-:-:-]"
	footerAppFmt = "[black:" + accent + ":bi] %s:%s [-:-:-]"
}

// setTheme swaps the decorative accent colors (borders, table headers,
// selection highlight, footer chips). Status colors (green/yellow/red)
// intentionally never change, so instance state always reads the same way
// regardless of theme.
func setTheme(name string) error {
	switch name {
	case "alibaba", "default", "orange", "":
		orange, orange2, accent = "#ff6a00", "#ff8c00", "#42d3ff"
		themeName = "alibaba"
	case "mono", "grayscale", "accessible":
		orange, orange2, accent = "white", "gray", "white"
		themeName = "mono"
	default:
		return fmt.Errorf("unknown theme %q; available: alibaba, mono", name)
	}
	rebuildFormats()
	return nil
}

// applyTheme overrides tview's default styles with a1s's fixed dark theme
// plus the current accent colors, so the app looks the same regardless of
// the user's terminal color scheme. Safe to call again after setTheme to
// re-apply a newly chosen theme to any primitive that reads tview.Styles at
// draw time (borders/titles set without an explicit override).
func applyTheme() {
	tview.Styles.PrimitiveBackgroundColor = colorBackground
	tview.Styles.ContrastBackgroundColor = colorBackground
	tview.Styles.MoreContrastBackgroundColor = colorBackground
	tview.Styles.BorderColor = tcell.GetColor(orange)
	tview.Styles.TitleColor = colorForeground
	tview.Styles.PrimaryTextColor = colorForeground
	tview.Styles.SecondaryTextColor = colorForeground
	tview.Styles.TertiaryTextColor = colorForeground
	tview.Styles.InverseTextColor = colorForeground
	tview.Styles.ContrastSecondaryTextColor = colorForeground
	tview.Styles.GraphicsColor = colorForeground
}

// stateColor returns the display color for an ECS instance lifecycle
// status, mirroring e1s/k9s/ec2s's convention of color-coding resource
// state.
func stateColor(status string) tcell.Color {
	switch {
	case strings.EqualFold(status, "Running"):
		return colorGreen
	case strings.EqualFold(status, "Stopped"), strings.EqualFold(status, "Deleted"):
		return colorRed
	case strings.EqualFold(status, "Starting"), strings.EqualFold(status, "Stopping"), strings.EqualFold(status, "Pending"):
		return colorYellow
	default:
		return colorForeground
	}
}
