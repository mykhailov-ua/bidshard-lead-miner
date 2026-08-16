package validate

import (
	"context"
	"net"
	"strings"
)

type MXValidator interface {
	HasMX(ctx context.Context, email string) (bool, error)
}

type Resolver struct{}

func (Resolver) HasMX(ctx context.Context, email string) (bool, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, nil
	}
	records, err := net.DefaultResolver.LookupMX(ctx, strings.ToLower(parts[1]))
	if err != nil || len(records) == 0 {
		return false, err
	}
	return true, nil
}

type StubMX struct {
	OK bool
}

func (s StubMX) HasMX(ctx context.Context, email string) (bool, error) {
	_ = ctx
	_ = email
	return s.OK, nil
}
