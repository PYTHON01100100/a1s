package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/model"
)

// headerKeys are the static keybinding hints shown in the info panel's
// right column, mirroring e1s/ec2s's header layout.
var headerKeys = []struct{ key, description string }{
	{"/", "Filter (Esc to clear)"},
	{"ctrl-p", "Switch profile"},
	{"ctrl-r", "Refresh"},
	{"g / G", "Top / bottom"},
	{"s", "Start instance"},
	{"S", "Stop instance"},
	{"x", "Economic stop"},
	{"R", "Reboot instance"},
	{"D", "Terminate instance"},
	{"E", "Run command (no SSH)"},
	{":", "Command palette"},
	{"?", "Help"},
	{"q", "Quit"},
}

// infoItemRows is the number of fields SetInstance renders in the left
// column, used by the caller to size the info panel tall enough for both
// columns.
const infoItemRows = 13

// Header renders the e1s-style "info" panel: details of the currently
// selected instance on the left, static keybinding hints on the right. It
// updates every time the table selection changes.
type Header struct {
	view     *tview.Flex
	itemsCol *tview.Flex
}

func newHeader() *Header {
	itemsCol := tview.NewFlex().SetDirection(tview.FlexRow)

	keysCol := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, k := range headerKeys {
		t := tview.NewTextView().
			SetDynamicColors(true).
			SetText(fmt.Sprintf(infoKeyFmt, k.key, k.description))
		keysCol.AddItem(t, 1, 1, false)
	}

	view := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(itemsCol, 0, 2, false).
		AddItem(keysCol, 0, 1, false)
	view.SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle(fmt.Sprintf(infoTitleFmt, "no selection"))

	h := &Header{view: view, itemsCol: itemsCol}
	h.SetInstance(nil)
	return h
}

// SetInstance updates the info panel to show inst's details, or a
// placeholder if inst is nil (no rows in the table).
func (h *Header) SetInstance(inst *model.ECSInstance) {
	h.itemsCol.Clear()

	if inst == nil {
		h.view.SetTitle(fmt.Sprintf(infoTitleFmt, "no selection"))
		t := tview.NewTextView().SetDynamicColors(true).SetText(" No instance selected ")
		h.itemsCol.AddItem(t, 1, 1, false)
		return
	}

	h.view.SetTitle(fmt.Sprintf(infoTitleFmt, blank(inst.Name, inst.ID)))

	kindText, _ := instanceKind(inst.Type)
	expireText, _ := billingExpiry(*inst)

	items := []struct{ name, value string }{
		{"Instance ID", inst.ID},
		{"Status", inst.Status},
		{"Type", inst.Type},
		{"Compute", kindText},
		{"OS", orDash(blank(inst.OSName, inst.OSType))},
		{"Internal IP", orDash(inst.PrivateIP)},
		{"External IP", orDash(inst.PublicIP)},
		{"VPC", pairLabelCIDR(inst.VPCName, inst.VPCID, inst.VPCCIDR)},
		{"VSwitch", pairLabelCIDR(inst.VSwitchName, inst.VSwitchID, inst.VSwitchCIDR)},
		{"Zone", zoneLabel(inst.Zone)},
		{"ENI ID", orDash(inst.ENIID)},
		{"Billing", billingLabel(inst.ChargeType)},
		{"Expires", expireText},
	}
	for _, item := range items {
		t := tview.NewTextView().
			SetDynamicColors(true).
			SetText(fmt.Sprintf(infoItemFmt, item.name, tview.Escape(item.value)))
		h.itemsCol.AddItem(t, 1, 1, false)
	}
}
