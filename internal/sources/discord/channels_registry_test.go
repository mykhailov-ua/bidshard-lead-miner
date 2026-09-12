package discord

import "testing"

func TestMergeChannelEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/channels.json"
	added, err := MergeChannelEntries(path, []ChannelEntry{
		{ChannelID: "111", GuildID: "g1", ChannelName: "affiliate", Enabled: true},
	})
	if err != nil || added != 1 {
		t.Fatalf("added=%d err=%v", added, err)
	}
	added, err = MergeChannelEntries(path, []ChannelEntry{
		{ChannelID: "111", GuildID: "g1", ChannelName: "affiliate", Enabled: true},
		{ChannelID: "222", GuildID: "g1", ChannelName: "support", Enabled: true},
	})
	if err != nil || added != 1 {
		t.Fatalf("second added=%d err=%v", added, err)
	}
	ids, err := ResolveChannelIDs(nil, path, true)
	if err != nil || len(ids) != 2 {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}
