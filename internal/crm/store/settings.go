package store

import (
	"context"

	"github.com/bidshard/parser/internal/ops"
)

func (s *LeadStore) Settings() *ops.SettingsStore {
	if s == nil {
		return nil
	}
	return s.settings
}

func (s *LeadStore) GetGlobalSettings(ctx context.Context) (ops.GlobalSettings, error) {
	if st := s.Settings(); st != nil {
		return st.Get(ctx)
	}
	return ops.GlobalSettings{LLMScoringEnabled: true}, nil
}

func (s *LeadStore) SetLLMScoringEnabled(ctx context.Context, enabled bool) error {
	st := s.Settings()
	if st == nil {
		return ops.ErrSettingsNotConfigured
	}
	return st.SetLLMScoringEnabled(ctx, enabled)
}
