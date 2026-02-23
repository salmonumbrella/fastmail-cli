package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

const defaultBulkBatchSize = 50

type bulkInputOptions struct {
	IDsFile   string
	FromStdin bool
	BatchSize int
}

func addBulkInputFlags(cmd *cobra.Command, opts *bulkInputOptions) {
	cmd.Flags().BoolVar(&opts.FromStdin, "stdin", false, "Read whitespace-delimited email IDs from stdin")
	cmd.Flags().StringVar(&opts.IDsFile, "ids-file", "", "Read whitespace-delimited email IDs from file")
	cmd.Flags().IntVar(&opts.BatchSize, "batch-size", defaultBulkBatchSize, "Email IDs per API request")
}

func validateBulkInputArgs(cmd *cobra.Command, args []string) error {
	useStdin, err := cmd.Flags().GetBool("stdin")
	if err != nil {
		return err
	}
	idsFile, err := cmd.Flags().GetString("ids-file")
	if err != nil {
		return err
	}
	if len(args) == 0 && !useStdin && strings.TrimSpace(idsFile) == "" {
		return fmt.Errorf("requires at least 1 arg(s), only received 0")
	}
	return nil
}

func collectBulkIDs(args []string, opts bulkInputOptions) ([]string, error) {
	ids := normalizeIDTokens(args)

	if path := strings.TrimSpace(opts.IDsFile); path != "" {
		fileIDs, err := readIDsFromFile(path)
		if err != nil {
			return nil, err
		}
		ids = append(ids, fileIDs...)
	}

	if opts.FromStdin {
		stdinIDs, err := readIDs(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read --stdin IDs: %w", err)
		}
		ids = append(ids, stdinIDs...)
	}

	// Deduplicate while preserving order
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	ids = unique

	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: no email IDs provided", ErrUsage)
	}

	return ids, nil
}

func runBulkInBatches(ids []string, batchSize int, opLabel string, op func(batch []string) (*jmap.BulkResult, error)) (*jmap.BulkResult, int, error) {
	if batchSize <= 0 {
		return nil, 0, fmt.Errorf("%w: --batch-size must be greater than 0", ErrUsage)
	}

	totalBatches := (len(ids) + batchSize - 1) / batchSize
	merged := &jmap.BulkResult{
		Succeeded: make([]string, 0, len(ids)),
		Failed:    make(map[string]string),
	}

	for start := 0; start < len(ids); start += batchSize {
		end := min(start+batchSize, len(ids))
		batchNum := (start / batchSize) + 1

		result, err := op(ids[start:end])
		if err != nil {
			return nil, totalBatches, fmt.Errorf("%s batch %d/%d: %w", opLabel, batchNum, totalBatches, err)
		}
		if result == nil {
			return nil, totalBatches, fmt.Errorf("%s batch %d/%d: empty result", opLabel, batchNum, totalBatches)
		}

		merged.Succeeded = append(merged.Succeeded, result.Succeeded...)
		for id, msg := range result.Failed {
			merged.Failed[id] = msg
		}
	}

	return merged, totalBatches, nil
}

func readIDsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open IDs file %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	ids, err := readIDs(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read IDs file %q: %w", path, err)
	}
	return ids, nil
}

func readIDs(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var ids []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ids = append(ids, strings.Fields(line)...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func normalizeIDTokens(tokens []string) []string {
	var ids []string
	for _, token := range tokens {
		if strings.TrimSpace(token) == "" {
			continue
		}
		ids = append(ids, strings.Fields(token)...)
	}
	return ids
}
