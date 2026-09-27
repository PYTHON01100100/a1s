package ui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/model"
)

var tableColumns = []string{"NAME", "INSTANCE ID", "STATUS", "TYPE", "COMPUTE", "OS", "INTERNAL IP", "EXTERNAL IP", "VPC", "VSWITCH", "ZONE", "BILLING", "EXPIRES"}

const (
	statusColumn  = 2
	computeColumn = 4
	expiresColumn = 12
)

// Table renders the ECS inventory, one row per instance.
type Table struct {
	view      *tview.Table
	instances []model.ECSInstance // currently displayed rows, in row order

	// onSelect is invoked whenever the selected row changes (interactively
	// or via SetInstances), with nil when there is no selectable row.
	onSelect func(inst *model.ECSInstance)
}

func newTable() *Table {
	view := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetBorders(false)
	view.SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle(fmt.Sprintf(tableTitleFmt, "ECS Instances", "default/auto", 0))
	view.SetSelectedStyle(tcell.StyleDefault.Background(tcell.GetColor(orange)).Foreground(tcell.ColorBlack))

	t := &Table{view: view}
	t.drawHeader()
	t.showPlaceholder("Loading instances…")

	view.SetSelectionChangedFunc(func(row, column int) {
		if t.onSelect == nil {
			return
		}
		inst, ok := t.SelectedInstance()
		if !ok {
			t.onSelect(nil)
			return
		}
		t.onSelect(&inst)
	})

	return t
}

// SetOnSelect registers fn to be called whenever the selected instance
// changes. It is not called for the table built by newTable until the
// first SetInstances call.
func (t *Table) SetOnSelect(fn func(inst *model.ECSInstance)) {
	t.onSelect = fn
}

func (t *Table) drawHeader() {
	for col, name := range tableColumns {
		cell := tview.NewTableCell(name).
			SetSelectable(false).
			SetTextColor(tcell.GetColor(orange2)).
			SetAttributes(tcell.AttrBold)
		t.view.SetCell(0, col, cell)
	}
}

// SetInstances replaces the displayed rows, sorted by name, and updates the
// table title to reflect scope and count. It preserves the current row
// selection when possible, and always notifies onSelect so dependent views
// (e.g. the info panel) stay in sync.
func (t *Table) SetInstances(instances []model.ECSInstance, scope string) {
	sorted := make([]model.ECSInstance, len(instances))
	copy(sorted, instances)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	prevRow, _ := t.view.GetSelection()

	t.instances = sorted
	t.view.Clear()
	t.drawHeader()
	t.view.SetTitle(fmt.Sprintf(tableTitleFmt, "ECS Instances", scope, len(sorted)))

	if len(sorted) == 0 {
		t.showPlaceholder("No instances found")
		if t.onSelect != nil {
			t.onSelect(nil)
		}
		return
	}

	for i, inst := range sorted {
		t.setRow(i+1, inst)
	}

	row := prevRow
	if row < 1 || row > len(sorted) {
		row = 1
	}
	t.view.Select(row, 0)
	if t.onSelect != nil {
		selected := sorted[row-1]
		t.onSelect(&selected)
	}
}

func (t *Table) setRow(row int, inst model.ECSInstance) {
	expireText, expireColor := billingExpiry(inst)
	kindText, kindColor := instanceKind(inst.Type)

	values := []string{
		blank(inst.Name, "-"),
		inst.ID,
		inst.Status,
		inst.Type,
		kindText,
		orDash(blank(inst.OSName, inst.OSType)),
		orDash(inst.PrivateIP),
		orDash(inst.PublicIP),
		pairLabel(inst.VPCName, inst.VPCID),
		pairLabel(inst.VSwitchName, inst.VSwitchID),
		zoneLabel(inst.Zone),
		billingLabel(inst.ChargeType),
		expireText,
	}

	for col, v := range values {
		cell := tview.NewTableCell(v)
		switch col {
		case statusColumn:
			cell.SetTextColor(stateColor(inst.Status))
		case computeColumn:
			cell.SetTextColor(tcell.GetColor(kindColor))
		case expiresColumn:
			cell.SetTextColor(tcell.GetColor(expireColor))
		}
		t.view.SetCell(row, col, cell)
	}
}

// showPlaceholder renders a centered, non-selectable message in row 1,
// used both for the initial "Loading instances…" state before the first
// fetch completes and for the "No instances found" empty state after.
func (t *Table) showPlaceholder(msg string) {
	cell := tview.NewTableCell(msg).
		SetSelectable(false).
		SetAlign(tview.AlignCenter)
	t.view.SetCell(1, 0, cell)
}

// SelectedInstance returns the instance under the current selection, if any.
func (t *Table) SelectedInstance() (model.ECSInstance, bool) {
	row, _ := t.view.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(t.instances) {
		return model.ECSInstance{}, false
	}
	return t.instances[idx], true
}

// SelectTop moves the selection to the first data row.
func (t *Table) SelectTop() {
	if len(t.instances) > 0 {
		t.view.Select(1, 0)
	}
}

// SelectBottom moves the selection to the last data row.
func (t *Table) SelectBottom() {
	if len(t.instances) > 0 {
		t.view.Select(len(t.instances), 0)
	}
}
