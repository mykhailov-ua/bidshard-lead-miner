package store

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidStatus = errors.New("invalid status")

var allowedStatuses = map[string]struct{}{
	"new":       {},
	"contacted": {},
	"won":       {},
	"lost":      {},
	"spam":      {},
	"archived":  {},
}

func NormalizeStatus(raw string) (string, error) {
	status := strings.ToLower(strings.TrimSpace(raw))
	if status == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidStatus)
	}
	if _, ok := allowedStatuses[status]; !ok {
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, raw)
	}
	return status, nil
}
