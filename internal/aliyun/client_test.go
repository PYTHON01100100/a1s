package aliyun

import (
	"a1s/internal/config"
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
