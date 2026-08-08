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
		return demoInstances(), nil
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
		res = append(res, model.ECSInstance{
			ID: str(m["InstanceId"]), Name: str(m["InstanceName"]), Status: str(m["Status"]),
			Type: str(m["InstanceType"]), Zone: str(m["ZoneId"]),
			PrivateIP: firstIP(m, "VpcAttributes", "PrivateIpAddress", "IpAddress"),
			PublicIP:  firstIP(m, "PublicIpAddress", "IpAddress"),
		})
	}
	return res, nil
}

func (c *Client) CPU(ctx context.Context, instanceID string, minutes int) ([]model.MetricPoint, error) {
	if c.cfg.Demo {
		return demoMetrics(), nil
	}
	if minutes <= 0 {
		minutes = 60
	}
	end := time.Now().UnixMilli()
	start := time.Now().Add(-time.Duration(minutes) * time.Minute).UnixMilli()
	dim := fmt.Sprintf(`{"instanceId":"%s"}`, instanceID)
	args := []string{"cms", "DescribeMetricList", "--Namespace", "acs_ecs_dashboard", "--MetricName", "CPUUtilization", "--Dimensions", dim, "--StartTime", strconv.FormatInt(start, 10), "--EndTime", strconv.FormatInt(end, 10), "--Period", "60"}
	out, err := c.runner.Run(ctx, c.common(args)...)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
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

func (c *Client) RunCommand(ctx context.Context, instanceID, command string) (model.CommandResult, error) {
	if c.cfg.ReadOnly {
		return model.CommandResult{}, errors.New("read-only mode: command execution is disabled")
	}
	if c.cfg.Demo {
		return model.CommandResult{InvokeID: "demo-invoke-001", Status: "Finished", Output: demoCommandOutput(command), ExitCode: 0}, nil
	}
	args := []string{"ecs", "RunCommand", "--Type", "RunShellScript", "--CommandContent", command, "--ContentEncoding", "PlainText", "--InstanceId.1", instanceID, "--KeepCommand", "false"}
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
		select {
		case <-ctx.Done():
			return model.CommandResult{}, ctx.Err()
		default:
		}
		res, done, err := c.invocationResult(ctx, rr.InvokeID, instanceID)
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

func (c *Client) invocationResult(ctx context.Context, invokeID, instanceID string) (model.CommandResult, bool, error) {
	args := []string{"ecs", "DescribeInvocationResults", "--InvokeId", invokeID, "--InstanceId", instanceID, "--ContentEncoding", "PlainText", "--MaxResults", "10"}
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
	if c.cfg.Demo {
		return model.BillSummary{BillingCycle: cycle, PretaxAmount: 2840.50, Currency: "USD"}, nil
	}
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
		items := digSlice(raw, "Data", "Items", "Item")
		for _, x := range items {
			if m, ok := x.(map[string]any); ok {
				amount += f64(m["PretaxAmount"])
			}
		}
	}
	return model.BillSummary{BillingCycle: cycle, PretaxAmount: amount, Currency: cur}, nil
}

func demoInstances() []model.ECSInstance {
	return []model.ECSInstance{
		{ID: "i-demo-web01", Name: "web-prod-01", Status: "Running", Type: "ecs.g8i.large", Zone: "me-central-1a", PrivateIP: "10.0.1.10", PublicIP: "8.213.10.10"},
		{ID: "i-demo-api01", Name: "api-prod-01", Status: "Running", Type: "ecs.g8i.xlarge", Zone: "me-central-1a", PrivateIP: "10.0.1.11"},
		{ID: "i-demo-gpu01", Name: "llm-gpu-01", Status: "Running", Type: "ecs.gn8is.2xlarge", Zone: "me-central-1b", PrivateIP: "10.0.2.20"},
	}
}
func demoMetrics() []model.MetricPoint {
	now := time.Now().UnixMilli()
	return []model.MetricPoint{{Timestamp: now - 120000, Average: 32, Maximum: 48, Minimum: 18}, {Timestamp: now - 60000, Average: 57, Maximum: 88, Minimum: 31}, {Timestamp: now, Average: 73, Maximum: 94, Minimum: 45}}
}
func demoCommandOutput(cmd string) string {
	if strings.Contains(cmd, "nginx") {
		return "active\n"
	}
	if strings.Contains(cmd, "df") {
		return "Filesystem Size Used Avail Use% Mounted on\n/dev/vda1 40G 29G 11G 73% /\n"
	}
	return "demo: command completed successfully\n"
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
func digSlice(m map[string]any, keys ...string) []any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	if s, ok := cur.([]any); ok {
		return s
	}
	return nil
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
