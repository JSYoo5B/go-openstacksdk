package cloudsnapshot

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// A 404 may complete deletion only after fault-free native rejection handling.
func (p *reader) deletePoll(ctx context.Context, target string) (*rest.Response, error) {
	return rest.DoJSONGuardedRejections(ctx, &p.source.Client, p.source.Guard, http.MethodGet, target, nil, nil,
		rest.RejectionPolicy{Codes: []int{http.StatusNotFound}}, sourceCodes()...)
}

// Backup export request framing shares the same operation-owned fault state.
type rejectedPollFaults struct{ rest.RejectedResponseFaults }

func (f *rejectedPollFaults) add(err error) { f.Add(err) }
func (f *rejectedPollFaults) error() error  { return f.Err() }

type rejectedPollTransport func(*http.Request) (*http.Response, error)

func (f rejectedPollTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
