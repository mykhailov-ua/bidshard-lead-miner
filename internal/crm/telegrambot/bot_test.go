package telegrambot

import "testing"

func TestParseCommand(t *testing.T) {
	t.Parallel()
	cmd, args := parseCommand("/export new 50")
	if cmd != "export" || len(args) != 2 || args[0] != "new" || args[1] != "50" {
		t.Fatalf("got cmd=%q args=%v", cmd, args)
	}
	cmd, args = parseCommand("/help@MyBot")
	if cmd != "help" || len(args) != 0 {
		t.Fatalf("got cmd=%q args=%v", cmd, args)
	}
}

func TestStatusOrAll(t *testing.T) {
	t.Parallel()
	if statusOrAll("") != "all" {
		t.Fatal("empty status")
	}
	if statusOrAll("new") != "new" {
		t.Fatal("new status")
	}
}
