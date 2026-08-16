package pipeline

import "github.com/bidshard/parser/internal/model"

type Task struct {
	RoundID string
	Item    model.RawItem
	Stats   *RoundState
}
