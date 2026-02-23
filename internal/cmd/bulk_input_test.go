package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func TestValidateBulkInputArgs(t *testing.T) {
	t.Run("requires at least one source", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("stdin", false, "")
		cmd.Flags().String("ids-file", "", "")

		err := validateBulkInputArgs(cmd, nil)
		if err == nil {
			t.Fatal("expected error for missing IDs")
		}
		if !strings.Contains(err.Error(), "requires at least 1 arg") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("allows stdin without positional args", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("stdin", false, "")
		cmd.Flags().String("ids-file", "", "")
		if err := cmd.Flags().Set("stdin", "true"); err != nil {
			t.Fatalf("set stdin flag: %v", err)
		}

		if err := validateBulkInputArgs(cmd, nil); err != nil {
			t.Fatalf("validateBulkInputArgs unexpected error: %v", err)
		}
	})
}

func TestCollectBulkIDs_FromAllSources(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "ids-*.txt")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer func() { _ = tmp.Close() }()

	if _, writeErr := tmp.WriteString("id2\nid3\n# comment\n"); writeErr != nil {
		t.Fatalf("write temp file: %v", writeErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		t.Fatalf("close temp file: %v", closeErr)
	}

	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = r
	_, _ = w.WriteString("id3 id4\n")
	_ = w.Close()

	ids, err := collectBulkIDs([]string{"id1", "id2"}, bulkInputOptions{
		IDsFile:   tmp.Name(),
		FromStdin: true,
	})
	if err != nil {
		t.Fatalf("collectBulkIDs unexpected error: %v", err)
	}

	want := []string{"id1", "id2", "id2", "id3", "id3", "id4"}
	if len(ids) != len(want) {
		t.Fatalf("collectBulkIDs len=%d, want %d (%v)", len(ids), len(want), ids)
	}
	for i, id := range want {
		if ids[i] != id {
			t.Fatalf("collectBulkIDs[%d]=%q, want %q; all=%v", i, ids[i], id, ids)
		}
	}
}

func TestRunBulkInBatches(t *testing.T) {
	t.Run("merges results across batches", func(t *testing.T) {
		var calls int
		result, batches, err := runBulkInBatches([]string{"id1", "id2", "id3"}, 2, "moving emails", func(batch []string) (*jmap.BulkResult, error) {
			calls++
			switch calls {
			case 1:
				return &jmap.BulkResult{
					Succeeded: []string{"id1"},
					Failed:    map[string]string{"id2": "notFound"},
				}, nil
			case 2:
				return &jmap.BulkResult{
					Succeeded: []string{"id3"},
					Failed:    map[string]string{},
				}, nil
			default:
				t.Fatalf("unexpected extra batch call %d", calls)
				return nil, nil
			}
		})
		if err != nil {
			t.Fatalf("runBulkInBatches unexpected error: %v", err)
		}
		if batches != 2 {
			t.Fatalf("batches=%d, want 2", batches)
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2", calls)
		}
		if len(result.Succeeded) != 2 {
			t.Fatalf("succeeded=%v, want 2 entries", result.Succeeded)
		}
		if len(result.Failed) != 1 || result.Failed["id2"] == "" {
			t.Fatalf("failed=%v, want id2 failure", result.Failed)
		}
	})

	t.Run("rejects invalid batch size", func(t *testing.T) {
		_, _, err := runBulkInBatches([]string{"id1"}, 0, "moving emails", func(batch []string) (*jmap.BulkResult, error) {
			return &jmap.BulkResult{}, nil
		})
		if err == nil {
			t.Fatal("expected batch size error")
		}
		if !strings.Contains(err.Error(), "--batch-size") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
