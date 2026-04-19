package writer

import (
	"context"

	"fastReadFile/internal/core"
)

type IngestionAdapter interface {
	Submit(ctx context.Context, records []core.RawRecord) error
}

type boundedIngestionAdapter struct {
	host WriterHost
}

var _ IngestionAdapter = (*boundedIngestionAdapter)(nil)

func NewBoundedIngestionAdapter(host WriterHost) IngestionAdapter {
	return &boundedIngestionAdapter{host: host}
}

func (a *boundedIngestionAdapter) Submit(ctx context.Context, records []core.RawRecord) error {
	return a.host.Submit(ctx, records)
}
