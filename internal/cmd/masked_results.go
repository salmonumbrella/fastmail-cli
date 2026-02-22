package cmd

import "fmt"

// printMaskedBulkResults prints the outcome of a bulk masked email alias operation.
//
// This is separate from printBulkResults because masked alias operations have different
// output formatting needs:
//   - Domain-centric messaging ("aliases for example.com" vs generic "messages")
//   - Pre-formatted error strings (alias info already included in error text)
//   - "Successfully/Partially" prefix style for clarity
//
// It uses []string for errors because masked alias errors are pre-formatted by the
// caller and already contain the alias identifier within the error message
// (e.g., "alias@domain.com: server error"). This avoids redundant ID->error mapping
// when the ID is already embedded in the error string.
//
// Parameters:
//   - action: past-tense verb describing what was done (e.g., "enabled", "disabled", "deleted")
//   - succeeded: number of aliases successfully processed
//   - failed: number of aliases that failed to process
//   - domain: the domain these aliases belong to
//   - errors: pre-formatted error strings, each containing the alias and error details
//
// Example output:
//
//	Successfully enabled 5 aliases for example.com
//	Partially disabled 3 aliases, 2 failed:
//	  alias1@example.com: not found
//	  alias2@example.com: server error
func printMaskedBulkResults(action string, succeeded, failed int, domain string, errors []string) {
	if failed == 0 {
		fmt.Printf("Successfully %s %d aliases for %s\n", action, succeeded, domain)
		return
	}

	fmt.Printf("Partially %s %d aliases, %d failed:\n", action, succeeded, failed)
	for _, err := range errors {
		fmt.Printf("  %s\n", err)
	}
}
