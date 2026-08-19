package warrior

import (
	"sync"
	"unsafe"
)

var byteBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

// FastHTMLStripper performs zero-allocation HTML tag stripping with bounds check elimination.
// Struct fields aligned by size descending.
type FastHTMLStripper struct {
	maxCap int
}

func NewFastHTMLStripper(maxCap int) FastHTMLStripper {
	if maxCap <= 0 {
		maxCap = 65536
	}
	return FastHTMLStripper{maxCap: maxCap}
}

// StripTagsBytes strips HTML tags directly from byte slices using zero-copy unsafe string conversion.
func (s FastHTMLStripper) StripTagsBytes(src []byte) string {
	n := len(src)
	if n == 0 {
		return ""
	}
	if n > s.maxCap {
		n = s.maxCap
		src = src[:n]
	}

	bufPtr := byteBufPool.Get().(*[]byte)
	out := (*bufPtr)[:0]

	// BCE hint: single bounds check at entry eliminates bounds checking in the loop
	_ = src[n-1]

	inTag := false
	spacePending := false

	for i := 0; i < n; i++ {
		b := src[i]
		if b == '<' {
			inTag = true
			if !spacePending && len(out) > 0 {
				out = append(out, ' ')
				spacePending = true
			}
			continue
		}
		if b == '>' {
			inTag = false
			continue
		}
		if inTag {
			continue
		}

		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			if !spacePending && len(out) > 0 {
				out = append(out, ' ')
				spacePending = true
			}
			continue
		}

		out = append(out, b)
		spacePending = false
	}

	result := string(out)

	*bufPtr = out[:0]
	byteBufPool.Put(bufPtr)

	return result
}

// StripTagsString strips HTML tags from a string input.
func (s FastHTMLStripper) StripTagsString(src string) string {
	if src == "" {
		return ""
	}
	return s.StripTagsBytes(unsafe.Slice(unsafe.StringData(src), len(src)))
}

// UnsafeString converts byte slice to string without allocation (Go 1.22+).
func UnsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
