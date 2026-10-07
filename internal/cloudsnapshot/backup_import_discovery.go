package cloudsnapshot

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cinderrequest"
)

func (p *reader) backupImportMicroversion(ctx context.Context) (string, []*MutationPage, error) {
	version, wire, err := cinderrequest.Negotiate(ctx, p.source, "3.64")
	pages := make([]*MutationPage, 0, len(wire))
	for _, response := range wire {
		pages = append(pages, mutationProof(response))
	}
	if len(pages) == 0 {
		pages = nil
	}
	return version, pages, err
}
