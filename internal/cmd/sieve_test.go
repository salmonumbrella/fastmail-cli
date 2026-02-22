package cmd

import "testing"

func TestSieveCmdStructure(t *testing.T) {
	app := newTestApp()
	cmd := newSieveCmd(app)

	if cmd.Use != "sieve" {
		t.Errorf("expected Use to be 'sieve', got %q", cmd.Use)
	}

	wanted := map[string]bool{"auth": false, "get": false, "set": false, "edit": false}
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

func TestSieveAuthFlags(t *testing.T) {
	app := newTestApp()
	cmd := newSieveAuthCmd(app)

	if cmd.Flags().Lookup("token") == nil {
		t.Error("expected --token flag")
	}
	if cmd.Flags().Lookup("cookie") == nil {
		t.Error("expected --cookie flag")
	}
	if cmd.Flags().Lookup("remove") == nil {
		t.Error("expected --remove flag")
	}
}

func TestSieveGetFlags(t *testing.T) {
	app := newTestApp()
	cmd := newSieveGetCmd(app)

	if cmd.Flags().Lookup("block") == nil {
		t.Error("expected --block flag")
	}
}

func TestSieveSetFlags(t *testing.T) {
	app := newTestApp()
	cmd := newSieveSetCmd(app)

	for _, name := range []string{"start", "start-file", "middle", "middle-file", "end", "end-file"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected --%s flag", name)
		}
	}
}

func TestSieveEditFlags(t *testing.T) {
	app := newTestApp()
	cmd := newSieveEditCmd(app)

	if cmd.Flags().Lookup("block") == nil {
		t.Error("expected --block flag")
	}
}
