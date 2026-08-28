package sources

import (
	"context"
	"testing"
)

type stubSource struct {
	name string
}

func (s stubSource) Name() string { return s.name }

func (s stubSource) Collect(context.Context, EmitFunc) error { return nil }

func TestOrderByCollectPriority(t *testing.T) {
	t.Parallel()
	srcs := []Source{
		stubSource{name: "lander"},
		stubSource{name: "reddit"},
		stubSource{name: "github"},
		stubSource{name: "forum"},
	}
	OrderByCollectPriority(srcs)
	want := []string{"forum", "reddit", "github", "lander"}
	for i, name := range want {
		if srcs[i].Name() != name {
			t.Fatalf("i=%d got=%s want=%s", i, srcs[i].Name(), name)
		}
	}
}
