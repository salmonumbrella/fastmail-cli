package cmd

import (
	"fmt"

	"github.com/salmonumbrella/fastmail-cli/internal/jmap"
	"github.com/spf13/cobra"
)

func buildMaskedDryRunAlias(email string, current, next jmap.MaskedEmailState) map[string]any {
	return map[string]any{
		"email":         email,
		"current_state": current,
		"new_state":     next,
	}
}

func printMaskedDryRunSingle(app *App, cmd *cobra.Command, email string, current, next jmap.MaskedEmailState) error {
	if app.IsJSON(cmd.Context()) {
		return app.PrintJSON(cmd, map[string]any{
			"dry_run":       true,
			"email":         email,
			"current_state": current,
			"new_state":     next,
		})
	}

	fmt.Printf("[dry-run] Would %s: %s (currently %s)\n", stateActionVerb(next), email, current)
	return nil
}

func printMaskedDryRunBulk(app *App, cmd *cobra.Command, domain string, next jmap.MaskedEmailState, toUpdate []jmap.MaskedEmail) error {
	if app.IsJSON(cmd.Context()) {
		aliases := make([]map[string]any, len(toUpdate))
		for i, alias := range toUpdate {
			aliases[i] = buildMaskedDryRunAlias(alias.Email, alias.State, next)
		}
		return app.PrintJSON(cmd, map[string]any{
			"dry_run": true,
			"domain":  domain,
			"count":   len(toUpdate),
			"aliases": aliases,
		})
	}

	fmt.Printf("[dry-run] Would %s %d aliases for %s:\n", stateActionVerb(next), len(toUpdate), domain)
	for _, alias := range toUpdate {
		fmt.Printf("  %s (currently %s)\n", alias.Email, alias.State)
	}
	return nil
}
