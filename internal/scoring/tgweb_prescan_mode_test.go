package scoring

import "testing"

func TestParseTgWebPrescanModeDefaultAggressive(t *testing.T) {
	t.Parallel()
	if ParseTgWebPrescanMode("") != TgWebPrescanAggressive {
		t.Fatal("empty should default aggressive")
	}
	if ParseTgWebPrescanMode("aggressive") != TgWebPrescanAggressive {
		t.Fatal("expected aggressive")
	}
}

func TestParseTgWebPrescanModeStrict(t *testing.T) {
	t.Parallel()
	if ParseTgWebPrescanMode("strict") != TgWebPrescanStrict {
		t.Fatal("expected strict")
	}
	if ParseTgWebPrescanMode("conservative") != TgWebPrescanStrict {
		t.Fatal("expected conservative -> strict")
	}
}
