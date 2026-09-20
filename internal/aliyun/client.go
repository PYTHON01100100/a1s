package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PYTHON01100100/a1s/internal/config"
	"github.com/PYTHON01100100/a1s/internal/model"
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
	sample []model.ECSInstance
}

func New(cfg config.Config, r Runner) *Client {
	if r == nil {
		r = ExecRunner{}
	}
	c := &Client{cfg: cfg, runner: r}
	if cfg.SampleData {
		c.sample = sampleInstances(cfg.Region)
	}
	return c
}

// sampleInstances returns deterministic, clearly-fake ECS instances used by
// `--sample-data` so the UI, lifecycle actions, and diagnostics can be
// exercised and screenshotted without touching a real Alibaba Cloud account.
func sampleInstances(region string) []model.ECSInstance {
	if region == "" {
		region = "me-central-1"
	}
	return []model.ECSInstance{
		{ID: "i-demo-web01", Name: "web-01", Status: "Running", Type: "ecs.g8i.large", Zone: region + "a", PrivateIP: "10.0.0.11", PublicIP: "47.100.10.11", OSName: "Alibaba Cloud Linux 3", OSType: "linux", ChargeType: "PostPaid", VPCID: "vpc-demo01", VPCName: "demo-vpc", VSwitchID: "vsw-demo01", VSwitchName: "demo-subnet-a"},
		{ID: "i-demo-web02", Name: "web-02", Status: "Running", Type: "ecs.g8i.large", Zone: region + "a", PrivateIP: "10.0.0.12", PublicIP: "47.100.10.12", OSName: "Alibaba Cloud Linux 3", OSType: "linux", ChargeType: "PostPaid", VPCID: "vpc-demo01", VPCName: "demo-vpc", VSwitchID: "vsw-demo01", VSwitchName: "demo-subnet-a"},
		{ID: "i-demo-db01", Name: "db-01", Status: "Stopped", Type: "ecs.r8i.large", Zone: region + "b", PrivateIP: "10.0.1.11", OSName: "Alibaba Cloud Linux 3", OSType: "linux", ChargeType: "PostPaid", VPCID: "vpc-demo01", VPCName: "demo-vpc", VSwitchID: "vsw-demo02", VSwitchName: "demo-subnet-b"},
		{ID: "i-demo-cache01", Name: "cache-01", Status: "Running", Type: "ecs.c8i.large", Zone: region + "a", PrivateIP: "10.0.0.20", OSName: "Alibaba Cloud Linux 3", OSType: "linux", ChargeType: "PrePaid", VPCID: "vpc-demo01", VPCName: "demo-vpc", VSwitchID: "vsw-demo01", VSwitchName: "demo-subnet-a"},
	}
}

func (c *Client) sampleIndex(id string) int {
	for i := range c.sample {
		if c.sample[i].ID == id {
			return i
		}
	}
	return -1
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
// CheckCLI verifies not just that the aliyun binary is installed, but that
// the active profile actually has usable credentials, by making a lightweight,
// read-only, region-independent STS call. A binary-only check (e.g. `aliyun
// version`) would report "ready" even for a freshly installed CLI with no
// account configured at all.
func (c *Client) CheckCLI(ctx context.Context) error {
	if _, err := c.runner.Run(ctx, "version"); err != nil {
		return fmt.Errorf("aliyun CLI not found or not executable: %w", err)
	}
	if _, err := c.runner.Run(ctx, c.common([]string{"sts", "GetCallerIdentity"})...); err != nil {
		return fmt.Errorf("no usable Alibaba Cloud credentials: %w", err)
	}
	return nil
}

func (c *Client) ListInstances(ctx context.Context) ([]model.ECSInstance, error) {
	if c.cfg.SampleData {
		out := make([]model.ECSInstance, len(c.sample))
		copy(out, c.sample)
		return out, nil
	}
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
	if c.cfg.SampleData {
		return sampleCPU(minutes), nil
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
// sampleCPU synthesizes a plausible CPU utilization curve for --sample-data
// mode so `metrics`/`doctor` have something to show without a real CMS call.
func sampleCPU(minutes int) []model.MetricPoint {
	now := time.Now()
	steps := minutes / 5
	if steps < 1 {
		steps = 1
	}
	pts := make([]model.MetricPoint, 0, steps)
	for i := steps; i >= 1; i-- {
		t := now.Add(-time.Duration(i*5) * time.Minute)
		avg := 22 + 14*math.Sin(float64(i)/3)
		if avg < 2 {
			avg = 2
		}
		pts = append(pts, model.MetricPoint{Timestamp: t.UnixMilli(), Average: avg, Maximum: avg + 9, Minimum: math.Max(avg-6, 0)})
	}
	return pts
}

func (c *Client) RunCommand(ctx context.Context, id, command string) (model.CommandResult, error) {
	if c.cfg.ReadOnly {
		return model.CommandResult{}, errors.New("read-only mode: command execution is disabled")
	}
	if c.cfg.SampleData {
		return model.CommandResult{InvokeID: "invoke-demo", Status: "Finished", Output: fmt.Sprintf("[sample-data] simulated output for: %s", command), ExitCode: 0}, nil
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
// StopMode controls whether Alibaba Cloud keeps billing a stopped instance's
// vCPU and memory. StopEco ("StopCharging") pauses that billing for eligible
// VPC pay-as-you-go instances; StopNormal ("KeepCharging") is the standard
// Alibaba Cloud default and keeps the instance ready for a fast restart.
type StopMode string

const (
	StopNormal StopMode = "KeepCharging"
	StopEco    StopMode = "StopCharging"
)

func (c *Client) requireWritable() error {
	if c.cfg.ReadOnly {
		return errors.New("read-only mode: instance actions are disabled")
	}
	return nil
}

func (c *Client) StartInstance(ctx context.Context, id string) error {
	if err := c.requireWritable(); err != nil {
		return err
	}
	if c.cfg.SampleData {
		i := c.sampleIndex(id)
		if i < 0 {
			return fmt.Errorf("sample instance %q not found", id)
		}
		c.sample[i].Status = "Running"
		return nil
	}
	_, err := c.runner.Run(ctx, c.regionArgs([]string{"ecs", "StartInstance", "--InstanceId", id})...)
	return err
}

func (c *Client) StopInstance(ctx context.Context, id string, mode StopMode, force bool) error {
	if err := c.requireWritable(); err != nil {
		return err
	}
	if c.cfg.SampleData {
		i := c.sampleIndex(id)
		if i < 0 {
			return fmt.Errorf("sample instance %q not found", id)
		}
		c.sample[i].Status = "Stopped"
		return nil
	}
	args := []string{"ecs", "StopInstance", "--InstanceId", id}
	if mode != "" {
		args = append(args, "--StoppedMode", string(mode))
	}
	if force {
		args = append(args, "--ForceStop", "true")
	}
	_, err := c.runner.Run(ctx, c.regionArgs(args)...)
	return err
}

func (c *Client) RebootInstance(ctx context.Context, id string, force bool) error {
	if err := c.requireWritable(); err != nil {
		return err
	}
	if c.cfg.SampleData {
		i := c.sampleIndex(id)
		if i < 0 {
			return fmt.Errorf("sample instance %q not found", id)
		}
		c.sample[i].Status = "Running"
		return nil
	}
	args := []string{"ecs", "RebootInstance", "--InstanceId", id}
	if force {
		args = append(args, "--ForceStop", "true")
	}
	_, err := c.runner.Run(ctx, c.regionArgs(args)...)
	return err
}

func (c *Client) DeleteInstance(ctx context.Context, id string, force bool) error {
	if err := c.requireWritable(); err != nil {
		return err
	}
	if c.cfg.SampleData {
		i := c.sampleIndex(id)
		if i < 0 {
			return fmt.Errorf("sample instance %q not found", id)
		}
		c.sample = append(c.sample[:i], c.sample[i+1:]...)
		return nil
	}
	args := []string{"ecs", "DeleteInstance", "--InstanceId", id}
	if force {
		args = append(args, "--Force", "true")
	}
	_, err := c.runner.Run(ctx, c.regionArgs(args)...)
	return err
}

func (c *Client) Bill(ctx context.Context, cycle string) (model.BillSummary, error) {
	if c.cfg.SampleData {
		return model.BillSummary{BillingCycle: cycle, PretaxAmount: 128.47, Currency: "USD"}, nil
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
