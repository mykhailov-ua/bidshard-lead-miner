package sources

import (
	"context"

	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Source interface {
	Name() string
	Collect(ctx context.Context, emit EmitFunc) error
}
