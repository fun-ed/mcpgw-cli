package cli

import (
	"os"
	"testing"

	"github.com/fun-ed/mcpgw-cli/internal/gw"
)

func TestDoctorMismatch(t *testing.T) {
	rows := []gw.ToolRow{
		{Name: "a_one", Target: "a", Description: "d"},
		{Name: "a_two", Target: "a", Description: "d"},
	}
	rep := doctorReport{URL: "http://x/mcp", Targets: map[string]int{}}
	for _, r := range rows {
		rep.Targets[r.Target]++
	}
	rep.Tools = len(rows)
	if got := len(rep.Targets); got != 1 {
		t.Fatalf("targets = %d", got)
	}
}

func TestPrintDoctorLines(t *testing.T) {
	f := t.TempDir() + "/out.txt"
	file, err := os.Create(f)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rep := doctorReport{
		URL: "http://x/mcp", Reachable: true, Protocol: "2025-06-18",
		InitializeMS: 500, CloseMS: 40,
		Targets: map[string]int{"a": 2, "b": 1}, Tools: 3,
	}
	printDoctor(file, rep)
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(f)
	want := "gateway\tok\thttp://x/mcp\ninitialize\t500ms\tprotocol 2025-06-18\n"
	if string(data[:len(want)]) != want {
		t.Fatalf("got %q", data)
	}
}
