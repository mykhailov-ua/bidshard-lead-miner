package warmpath

import "github.com/bidshard/parser/internal/gemini"

var _ LeadBatchAnalyzer = (*gemini.Client)(nil)
