package discord

import "testing"

func TestExtractInviteCodes(t *testing.T) {
	t.Parallel()
	text := `Join https://discord.gg/VoluumAff and discord.com/invite/AffNet-2024`
	got := ExtractInviteCodes(text)
	if len(got) != 2 {
		t.Fatalf("codes=%v", got)
	}
}

func TestHeuristicTriageInviteDropsNSFW(t *testing.T) {
	t.Parallel()
	status, _ := HeuristicTriageInvite("nsfw-club", "crypto pump group")
	if status != "drop" {
		t.Fatalf("status=%q", status)
	}
}

func TestAppendInvitesDedupes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/invites.json"
	added, err := AppendInvites(path, "serp", "site:discord.gg voluum", []string{"affnet", "affnet"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d", added)
	}
	f, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Invites) != 1 || f.Invites[0].InviteCode != "affnet" {
		t.Fatalf("invites=%+v", f.Invites)
	}
}
