package ui

import (
	"testing"

	"github.com/PYTHON01100100/a1s/internal/model"
)

func sampleInstances() []model.ECSInstance {
	return []model.ECSInstance{
		{ID: "i-1", Name: "web-01", Status: "Running", Type: "ecs.g8i.large", Zone: "me-central-1a"},
		{ID: "i-2", Name: "web-02", Status: "Stopped", Type: "ecs.g8i.large", Zone: "me-central-1b"},
		{ID: "i-3", Name: "db-01", Status: "Running", Type: "ecs.r8i.large", Zone: "me-central-1a"},
	}
}

func TestFilterInstancesSubstring(t *testing.T) {
	out := filterInstances(sampleInstances(), "web")
	if len(out) != 2 {
		t.Fatalf("expected 2 web instances, got %d", len(out))
	}
}

func TestFilterInstancesFieldMatch(t *testing.T) {
	out := filterInstances(sampleInstances(), "status=running")
	if len(out) != 2 {
		t.Fatalf("expected 2 running instances, got %d", len(out))
	}
	for _, x := range out {
		if x.Status != "Running" {
			t.Fatalf("filter leaked non-running instance: %+v", x)
		}
	}
}

func TestFilterInstancesEmptyQueryReturnsAll(t *testing.T) {
	in := sampleInstances()
	out := filterInstances(in, "")
	if len(out) != len(in) {
		t.Fatalf("empty filter should return all instances, got %d", len(out))
	}
}

func TestSetThemeUnknown(t *testing.T) {
	if err := setTheme("neon"); err == nil {
		t.Fatal("expected error for unknown theme")
	}
	if err := setTheme("mono"); err != nil {
		t.Fatal(err)
	}
	if err := setTheme("alibaba"); err != nil {
		t.Fatal(err)
	}
}
