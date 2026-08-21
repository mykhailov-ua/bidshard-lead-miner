package gemini

// Priority splits generateContent RPM/TPM/RPD budgets.
type Priority int

const (
	PriorityCritical Priority = iota // sync geo gate (small slice)
	PriorityHigh                     // accepted lead batch analysis
	PriorityNormal                   // junk batch analyze
	PriorityLow                      // reports, keyword/discover diff
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityHigh:
		return "high"
	case PriorityNormal:
		return "normal"
	case PriorityLow:
		return "low"
	default:
		return "unknown"
	}
}

// QuotaSplit holds percent of generate RPM per priority (must sum to 100).
type QuotaSplit struct {
	Critical int
	High     int
	Normal   int
	Low      int
}

func DefaultQuotaSplit() QuotaSplit {
	return QuotaSplit{Critical: 20, High: 40, Normal: 25, Low: 15}
}

func (s QuotaSplit) Normalize() QuotaSplit {
	if s.Critical+s.High+s.Normal+s.Low != 100 {
		return DefaultQuotaSplit()
	}
	if s.Critical < 1 {
		s.Critical = 1
	}
	return s
}

func (s QuotaSplit) RPM(total int, p Priority) int {
	if total <= 0 {
		total = 10
	}
	s = s.Normalize()
	var pct int
	switch p {
	case PriorityCritical:
		pct = s.Critical
	case PriorityHigh:
		pct = s.High
	case PriorityNormal:
		pct = s.Normal
	default:
		pct = s.Low
	}
	rpm := total * pct / 100
	if rpm < 1 {
		return 1
	}
	return rpm
}
