package version

import "testing"

func TestDefaultVersion(t *testing.T) {
	if ServerVersion != "0.1.0" {
		t.Errorf("ServerVersion = %q, want %q", ServerVersion, "0.1.0")
	}
}

func TestDefaultGitCommit(t *testing.T) {
	if GitCommit != "unknown" {
		t.Errorf("GitCommit = %q, want %q", GitCommit, "unknown")
	}
}
