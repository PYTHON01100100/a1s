package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/PYTHON01100100/a1s/internal/model"
)

// filterInstances narrows an ECS list either by a `key=value` match against
// one field, or by a plain substring match across name/id/status/type/zone/
// billing/IPs/VPC/vSwitch/CIDR/ENI. It backs both the "/" live filter box
// and the palette's `query` command.
func filterInstances(xs []model.ECSInstance, q string) []model.ECSInstance {
	q = strings.TrimSpace(q)
	if q == "" {
		return xs
	}
	if k, v, ok := strings.Cut(q, "="); ok {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.ToLower(strings.TrimSpace(v))
		out := make([]model.ECSInstance, 0, len(xs))
		for _, x := range xs {
			if strings.Contains(strings.ToLower(fieldValue(x, k)), v) {
				out = append(out, x)
			}
		}
		return out
	}
	ql := strings.ToLower(q)
	out := make([]model.ECSInstance, 0, len(xs))
	for _, x := range xs {
		hay := strings.ToLower(strings.Join([]string{
			x.Name, x.ID, x.Status, x.Type, x.Zone, x.ChargeType, x.PublicIP, x.PrivateIP,
			x.VPCID, x.VPCName, x.VPCCIDR, x.VSwitchID, x.VSwitchName, x.VSwitchCIDR, x.ENIID,
		}, " "))
		if strings.Contains(hay, ql) {
			out = append(out, x)
		}
	}
	return out
}

// fieldValue returns the raw value of one named field, used by
// filterInstances' key=value matcher.
func fieldValue(x model.ECSInstance, key string) string {
	switch key {
	case "status":
		return x.Status
	case "type":
		return x.Type
	case "compute", "kind":
		kind, _ := instanceKind(x.Type)
		return kind
	case "zone":
		return x.Zone
	case "region":
		return regionFromZone(x.Zone)
	case "name":
		return x.Name
	case "id", "instance", "instanceid":
		return x.ID
	case "ip":
		return x.PrivateIP + " " + x.PublicIP
	case "internalip", "privateip":
		return x.PrivateIP
	case "externalip", "publicip":
		return x.PublicIP
	case "vpc", "vpcid":
		return x.VPCID
	case "vpcname":
		return x.VPCName
	case "vswitch", "vswitchid", "vsw":
		return x.VSwitchID
	case "vswitchname":
		return x.VSwitchName
	case "cidr":
		return x.VPCCIDR + " " + x.VSwitchCIDR
	case "vpccidr":
		return x.VPCCIDR
	case "vswitchcidr":
		return x.VSwitchCIDR
	case "eni", "eniid":
		return x.ENIID
	case "charge", "chargetype", "billing":
		return x.ChargeType
	}
	return ""
}

// newFilterInput builds the "/" filter input field. onApply is called on
// every keystroke change (live filtering) and onClose when the field is
// dismissed via Esc or Enter.
func newFilterInput(initial string, onApply func(query string), onClose func()) *tview.InputField {
	field := tview.NewInputField().
		SetLabel("Filter (text or field=value, e.g. status=running): ").
		SetFieldWidth(0)
	field.SetText(initial)

	field.SetChangedFunc(onApply)

	field.SetDoneFunc(func(key tcell.Key) {
		onClose()
	})

	return field
}
