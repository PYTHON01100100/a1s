package aliyun

import (
	"github.com/PYTHON01100100/a1s/internal/config"
	"context"
	"strings"
	"testing"
)

type fakeRunner struct {
	outputs [][]byte
	calls   [][]string
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if len(f.outputs) == 0 {
		return []byte(`{}`), nil
	}
	o := f.outputs[0]
	f.outputs = f.outputs[1:]
	return o, nil
}

func TestListInstances(t *testing.T) {
	f := &fakeRunner{outputs: [][]byte{[]byte(`{"Instances":{"Instance":[{"InstanceId":"i-1","InstanceName":"web","Status":"Running","InstanceType":"ecs.g8i.large","ZoneId":"me-central-1a","VpcAttributes":{"PrivateIpAddress":{"IpAddress":["10.0.0.2"]}},"PublicIpAddress":{"IpAddress":["1.2.3.4"]}}]}}`)}}
	c := New(config.Config{Region: "me-central-1"}, f)
	xs, err := c.ListInstances(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(xs) != 1 || xs[0].ID != "i-1" || xs[0].PrivateIP != "10.0.0.2" {
		t.Fatalf("bad parse: %#v", xs)
	}
	if !strings.Contains(strings.Join(f.calls[0], " "), "DescribeInstances") {
		t.Fatal("DescribeInstances not called")
	}
}

func TestStopInstanceModes(t *testing.T) {
	f := &fakeRunner{}
	c := New(config.Config{Region: "me-central-1"}, f)

	if err := c.StopInstance(context.Background(), "i-1", StopNormal, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(f.calls[0], " "), "StoppedMode KeepCharging") {
		t.Fatalf("normal stop must request KeepCharging: %v", f.calls[0])
	}

	if err := c.StopInstance(context.Background(), "i-1", StopEco, true); err != nil {
		t.Fatal(err)
	}
	call := strings.Join(f.calls[1], " ")
	if !strings.Contains(call, "StoppedMode StopCharging") {
		t.Fatalf("eco stop must request StopCharging: %v", f.calls[1])
	}
	if !strings.Contains(call, "ForceStop true") {
		t.Fatalf("force flag not passed through: %v", f.calls[1])
	}
}

func TestLifecycleActionsRespectReadOnly(t *testing.T) {
	f := &fakeRunner{}
	c := New(config.Config{Region: "me-central-1", ReadOnly: true}, f)

	if err := c.StartInstance(context.Background(), "i-1"); err == nil {
		t.Fatal("expected StartInstance to be blocked in read-only mode")
	}
	if err := c.StopInstance(context.Background(), "i-1", StopNormal, false); err == nil {
		t.Fatal("expected StopInstance to be blocked in read-only mode")
	}
	if err := c.RebootInstance(context.Background(), "i-1", false); err == nil {
		t.Fatal("expected RebootInstance to be blocked in read-only mode")
	}
	if err := c.DeleteInstance(context.Background(), "i-1", false); err == nil {
		t.Fatal("expected DeleteInstance to be blocked in read-only mode")
	}
	if len(f.calls) != 0 {
		t.Fatalf("read-only mode must not shell out to aliyun CLI, got calls: %v", f.calls)
	}
}
