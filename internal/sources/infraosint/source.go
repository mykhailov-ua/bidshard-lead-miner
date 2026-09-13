package infraosint

import (
	"context"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/model"
)

const defaultClustersPath = "data/runtime/infra_clusters.json"

// Crawler reads scored infra_clusters.json and emits intel_only rows (H6).
type Crawler struct {
	Path string
}

func NewCrawler(path string) *Crawler {
	return &Crawler{Path: strings.TrimSpace(path)}
}

func (c *Crawler) Name() string { return "infraosint" }

type EmitFunc func(ctx context.Context, item model.RawItem) error

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	path := c.Path
	if path == "" {
		path = defaultClustersPath
	}
	file, err := discover.LoadInfraClusters(path)
	if err != nil {
		return err
	}
	for _, cluster := range file.Clusters {
		if err := ctx.Err(); err != nil {
			return err
		}
		if cluster.Score < 30 {
			continue
		}
		domain := strings.TrimSpace(cluster.SampleDomain)
		if domain == "" && len(cluster.Domains) > 0 {
			domain = cluster.Domains[0]
		}
		tracker := strings.TrimSpace(cluster.TrackerHint)
		text := fmt.Sprintf(
			"infra cluster ip=%s domains=%d tracker=%s country=%s sample=%s intel_queue=manual whitepage_tg_domain=%s",
			cluster.IP,
			len(cluster.Domains),
			tracker,
			strings.TrimSpace(cluster.Country),
			domain,
			domain,
		)
		source := fmt.Sprintf("infraosint:%s", strings.TrimSpace(cluster.IP))
		if source == "infraosint:" {
			source = "infraosint:cluster"
		}
		contact := ""
		if domain != "" {
			contact = "domain:" + domain
		}
		if err := emit(ctx, model.RawItem{
			Source:  source,
			Raw:     text,
			Title:   domain,
			Contact: contact,
		}); err != nil {
			return err
		}
	}
	return nil
}
