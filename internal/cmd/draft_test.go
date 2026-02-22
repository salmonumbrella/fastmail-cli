package cmd

import "testing"

func TestDraftCmdStructure(t *testing.T) {
	app := newTestApp()
	cmd := newDraftCmd(app)

	if cmd.Use != "draft" {
		t.Errorf("expected Use to be 'draft', got %q", cmd.Use)
	}

	wanted := map[string]bool{"list": false, "get": false, "new": false, "send": false, "delete": false}
	for _, c := range cmd.Commands() {
		if _, ok := wanted[c.Name()]; ok {
			wanted[c.Name()] = true
		}
	}
	for name, found := range wanted {
		if !found {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestDraftListFlags(t *testing.T) {
	app := newTestApp()
	cmd := newDraftListCmd(app)

	if cmd.Flags().Lookup("limit") == nil {
		t.Error("expected --limit flag")
	}
}

func TestDraftSendFlags(t *testing.T) {
	app := newTestApp()
	cmd := newDraftSendCmd(app)

	// --yes is a global persistent flag (defined on the root command).
	// Subcommands should not define it locally (it would shadow the persistent flag).
	if cmd.Flags().Lookup("yes") != nil {
		t.Error("did not expect a local --yes flag (it should be inherited from root)")
	}
}

func TestDraftDeleteFlags(t *testing.T) {
	app := newTestApp()
	cmd := newDraftDeleteCmd(app)

	if cmd.Flags().Lookup("yes") != nil {
		t.Error("did not expect a local --yes flag (it should be inherited from root)")
	}
}
