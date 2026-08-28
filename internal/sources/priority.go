package sources

import (
	"sort"

	"github.com/bidshard/parser/internal/filter"
)

// OrderByCollectPriority sorts sources high-to-low (reddit/forum before lander/github).
func OrderByCollectPriority(sources []Source) {
	sort.SliceStable(sources, func(i, j int) bool {
		pi := filter.CollectPriorityFamily(sources[i].Name())
		pj := filter.CollectPriorityFamily(sources[j].Name())
		if pi != pj {
			return pi > pj
		}
		return sources[i].Name() < sources[j].Name()
	})
}
