package discord

import "testing"

func TestGuildLooksICP(t *testing.T) {
	t.Parallel()
	if !GuildLooksICP("Voluum Affiliates", "") {
		t.Fatal("expected voluum guild pass")
	}
	if GuildLooksICP("crypto pump signals vip", "") {
		t.Fatal("expected junk guild drop")
	}
}

func TestChannelLooksReadable(t *testing.T) {
	t.Parallel()
	if !ChannelLooksReadable("affiliate-chat", 0) {
		t.Fatal("expected affiliate chat")
	}
	if ChannelLooksReadable("rules", 0) {
		t.Fatal("expected rules skip")
	}
	if ChannelLooksReadable("voice lobby", 2) {
		t.Fatal("expected voice skip")
	}
}
