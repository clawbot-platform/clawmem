package version

import "testing"

func TestGetReturnsCurrentBuildInfo(t *testing.T) {
	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
	})

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-03-25"

	info := Get()
	if info.Version != "1.2.3" || info.Commit != "abc123" || info.Date != "2026-03-25" {
		t.Fatalf("unexpected version info %#v", info)
	}
}
