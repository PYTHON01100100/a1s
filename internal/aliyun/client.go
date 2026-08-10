package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"a1s/internal/config"
	"a1s/internal/model"
)

type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "aliyun", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("aliyun: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return out, nil
}

type Client struct {
	cfg    config.Config
	runner Runner
}

func New(cfg config.Config, r Runner) *Client {
	if r == nil {
		r = ExecRunner{}
	}
	return &Client{cfg: cfg, runner: r}
}
func (c *Client) common(args []string) []string {
	if c.cfg.Profile != "" {
		args = append(args, "--profile", c.cfg.Profile)
	}
	return args
}
func (c *Client) regionArgs(args []string) []string {
	if c.cfg.Region != "" {
		args = append(args, "--RegionId", c.cfg.Region)
	}
	return c.common(args)
}
func (c *Client) CheckCLI(ctx context.Context) error {
	_, err := c.runner.Run(ctx, "version")
	return err
}

func (c *Client) ListInstances(ctx context.Context) ([]model.ECSInstance, error) {
	if c.cfg.Demo {
		return []model.ECSInstance{}, nil
	}
	out, err := c.runner.Run(ctx, c.regionArgs([]string{"ecs", "DescribeInstances", "--PageSize", "100"})...)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	items := digSlice(raw, "Instances", "Instance")
	res := make([]model.ECSInstance, 0, len(items))
	for _, x := range items {
		m, _ := x.(map[string]any)
		vpc := digMap(m, "VpcAttributes")
		res = append(res, model.ECSInstance{ID: str(m["InstanceId"]), Name: str(m["InstanceName"]), Status: str(m["Status"]), Type: str(m["InstanceType"]), Zone: str(m["ZoneId"]), PrivateIP: firstIP(m, "VpcAttributes", "PrivateIpAddress", "IpAddress"), PublicIP: firstIP(m, "PublicIpAddress", "IpAddress"), OSName: str(m["OSName"]), OSType: str(m["OSType"]), ImageID: str(m["ImageId"]), ChargeType: str(m["InstanceChargeType"]), VPCID: str(vpc["VpcId"]), VSwitchID: str(vpc["VSwitchId"])})
	}
	c.enrichNetworkNames(ctx, res)
	return res, nil
}

func (c *Client) enrichNetworkNames(ctx context.Context, xs []model.ECSInstance) {
	vpcs := map[string]string{}
	vsw := map[string]string{}
	if out, err := c.runner.Run(ctx, c.regionArgs([]string{"vpc", "DescribeVpcs", "--PageNumber", "1", "--PageSize", "100"})...); err == nil {
		var raw map[string]any
		if json.Unmarshal(out, &raw) == nil {
			for _, x := range digSlice(raw, "Vpcs", "Vpc") {
				m, _ := x.(map[string]any)
				vpcs[str(m["VpcId"])] = str(m["VpcName"])
			}
		}
	}
	if out, err := c.runner.Run(ctx, c.regionArgs([]string{"vpc", "DescribeVSwitches", "--PageNumber", "1", "--PageSize", "100"})...); err == nil {
		var raw map[string]any
		if json.Unmarshal(out, &raw) == nil {
			for _, x := range digSlice(raw, "VSwitches", "VSwitch") {
				m, _ := x.(map[string]any)
				vsw[str(m["VSwitchId"])] = str(m["VSwitchName"])
			}
		}
	}
	for i := range xs {
		xs[i].VPCName = vpcs[xs[i].VPCID]
		xs[i].VSwitchName = vsw[xs[i].VSwitchID]
	}
}

func (c *Client) CPU(ctx context.Context, id string, minutes int) ([]model.MetricPoint, error) {
	if minutes <= 0 {
		minutes = 60
	}
	end := time.Now().UnixMilli()
	start := time.Now().Add(-time.Duration(minutes) * time.Minute).UnixMilli()
	dim := fmt.Sprintf(`{"instanceId":"%s"}`, id)
	args := []string{"cms", "DescribeMetricList", "--Namespace", "acs_ecs_dashboard", "--MetricName", "CPUUtilization", "--Dimensions", dim, "--StartTime", strconv.FormatInt(start, 10), "--EndTime", strconv.FormatInt(end, 10), "--Period", "60"}
	out, err := c.runner.Run(ctx, c.common(args)...)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if json.Unmarshal(out, &raw) != nil {
		return nil, err
	}
	datapoints := str(raw["Datapoints"])
	if datapoints == "" || datapoints == "[]" {
		return nil, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(datapoints), &rows); err != nil {
		return nil, err
	}
	pts := make([]model.MetricPoint, 0, len(rows))
	for _, r := range rows {
		pts = append(pts, model.MetricPoint{Timestamp: i64(r["timestamp"]), Average: f64(r["Average"]), Maximum: f64(r["Maximum"]), Minimum: f64(r["Minimum"])})
	}
	return pts, nil
}
func (c *Client) RunCommand(ctx context.Context, id, command string) (model.CommandResult, error) {
	if c.cfg.ReadOnly {
		return model.CommandResult{}, errors.New("read-only mode: command execution is disabled")
	}
	args := []string{"ecs", "RunCommand", "--Type", "RunShellScript", "--CommandContent", command, "--ContentEncoding", "PlainText", "--InstanceId.1", id, "--KeepCommand", "false"}
	out, err := c.runner.Run(ctx, c.regionArgs(args)...)
	if err != nil {
		return model.CommandResult{}, err
	}
	var rr struct {
		InvokeID string `json:"InvokeId"`
	}
	if err := json.Unmarshal(out, &rr); err != nil {
		return model.CommandResult{}, err
	}
	if rr.InvokeID == "" {
		return model.CommandResult{}, errors.New("RunCommand returned no InvokeId")
	}
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		res, done, err := c.invocationResult(ctx, rr.InvokeID, id)
		if err != nil {
			return model.CommandResult{}, err
		}
		if done {
			return res, nil
		}
		time.Sleep(1500 * time.Millisecond)
	}
	return model.CommandResult{InvokeID: rr.InvokeID, Status: "Timeout"}, errors.New("timed out waiting for Cloud Assistant result")
}
func (c *Client) invocationResult(ctx context.Context, invokeID, id string) (model.CommandResult, bool, error) {
	args := []string{"ecs", "DescribeInvocationResults", "--InvokeId", invokeID, "--InstanceId", id, "--ContentEncoding", "PlainText", "--MaxResults", "10"}
	out, err := c.runner.Run(ctx, c.regionArgs(args)...)
	if err != nil {
		return model.CommandResult{}, false, err
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return model.CommandResult{}, false, err
	}
	rows := digSlice(raw, "Invocation", "InvocationResults", "InvocationResult")
	if len(rows) == 0 {
		return model.CommandResult{InvokeID: invokeID, Status: "Pending"}, false, nil
	}
	m, _ := rows[0].(map[string]any)
	status := str(m["InvocationStatus"])
	res := model.CommandResult{InvokeID: invokeID, Status: status, Output: str(m["Output"]), ExitCode: int(i64(m["ExitCode"]))}
	done := status == "Finished" || status == "Failed" || status == "Stopped" || status == "Terminated"
	return res, done, nil
}
func (c *Client) Bill(ctx context.Context, cycle string) (model.BillSummary, error) {
	args := []string{"bssopenapi", "QueryAccountBill", "--BillingCycle", cycle, "--PageNum", "1", "--PageSize", "100"}
	out, err := c.runner.Run(ctx, c.common(args)...)
	if err != nil {
		return model.BillSummary{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return model.BillSummary{}, err
	}
	cur := str(raw["Currency"])
	if cur == "" {
		cur = str(raw["CurrencyCode"])
	}
	if cur == "" {
		cur = "USD"
	}
	amount := f64(raw["PretaxAmount"])
	if amount == 0 {
		for _, x := range digSlice(raw, "Data", "Items", "Item") {
			if m, ok := x.(map[string]any); ok {
				amount += f64(m["PretaxAmount"])
			}
		}
	}
	return model.BillSummary{BillingCycle: cycle, PretaxAmount: amount, Currency: cur}, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
func f64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		n, _ := strconv.ParseFloat(x, 64)
		return n
	case json.Number:
		n, _ := x.Float64()
		return n
	}
	return 0
}
func i64(v any) int64 { return int64(f64(v)) }
func digMap(m map[string]any, keys ...string) map[string]any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		cur = mm[k]
	}
	mm, _ := cur.(map[string]any)
	if mm == nil {
		return map[string]any{}
	}
	return mm
}
func digSlice(m map[string]any, keys ...string) []any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	s, _ := cur.([]any)
	return s
}
func firstIP(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[k]
	}
	if a, ok := cur.([]any); ok && len(a) > 0 {
		return str(a[0])
	}
	return ""
}
