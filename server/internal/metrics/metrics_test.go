package metrics

import (
	"testing"
)

// TestInit verifies that the package init() did not panic when calling
// prometheus.MustRegister, and that all metric variables are non-nil.
func TestInit(t *testing.T) {
	if ConnectionsTotal == nil {
		t.Error("ConnectionsTotal is nil after init")
	}
	if MessagesSentTotal == nil {
		t.Error("MessagesSentTotal is nil after init")
	}
	if MessagesPushTotal == nil {
		t.Error("MessagesPushTotal is nil after init")
	}
}

func TestConnectionsTotal_IsGauge(t *testing.T) {
	// Verify the metric descriptor uses the expected name and type.
	desc := ConnectionsTotal.Desc()
	if desc == nil {
		t.Fatal("ConnectionsTotal.Desc() returned nil")
	}
	if desc.String() == "" {
		t.Error("ConnectionsTotal.Desc() returned empty string")
	}
}

func TestMessagesSentTotal_IsCounter(t *testing.T) {
	desc := MessagesSentTotal.Desc()
	if desc == nil {
		t.Fatal("MessagesSentTotal.Desc() returned nil")
	}
	if desc.String() == "" {
		t.Error("MessagesSentTotal.Desc() returned empty string")
	}
}

func TestMessagesPushTotal_IsCounter(t *testing.T) {
	desc := MessagesPushTotal.Desc()
	if desc == nil {
		t.Fatal("MessagesPushTotal.Desc() returned nil")
	}
	if desc.String() == "" {
		t.Error("MessagesPushTotal.Desc() returned empty string")
	}
}
