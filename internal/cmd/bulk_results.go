package cmd

import "fmt"

// printBulkResults prints the outcome of a bulk email operation (delete, move, mark).
//
// It uses map[string]string for the failed parameter because email operations can fail
// independently per message ID, and each failure has a specific error message from the
// server (e.g., "message not found", "permission denied"). The map keys are email IDs
// and values are the corresponding error messages.
//
// Parameters:
//   - action: past-tense verb describing what was done (e.g., "Deleted", "Moved", "Marked")
//   - target: optional qualifier for the action (e.g., "messages", "to Archive")
//   - succeededCount: number of successfully processed emails
//   - failedCount: number of emails that failed to process
//   - failed: map of email ID to error message for each failure
//
// Example output:
//
//	Deleted 5 messages
//	Moved 3 to Archive, 2 failed:
//	  M123abc: message not found
//	  M456def: permission denied
func printBulkResults(action, target string, succeededCount, failedCount int, failed map[string]string) {
	if failedCount == 0 {
		if target != "" {
			fmt.Printf("%s %d %s\n", action, succeededCount, target)
			return
		}
		fmt.Printf("%s %d\n", action, succeededCount)
		return
	}

	if target != "" {
		fmt.Printf("%s %d %s, %d failed:\n", action, succeededCount, target, failedCount)
	} else {
		fmt.Printf("%s %d, %d failed:\n", action, succeededCount, failedCount)
	}
	for id, errMsg := range failed {
		fmt.Printf("  %s: %s\n", id, errMsg)
	}
}
