package lander

import (
	"context"
	"fmt"
)

type HeadlessFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
}

type DisabledHeadless struct{}

func (DisabledHeadless) Fetch(ctx context.Context, url string) (string, error) {
	_ = ctx
	_ = url
	return "", fmt.Errorf("headless disabled")
}
