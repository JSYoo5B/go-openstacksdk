package serviceinfo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type preparedRecordSource struct {
	*preparedSource
	ctx      context.Context
	check    func(context.Context) error
	location json.RawMessage
}

// Owned records add sticky target and context guards to the existing discovery
// source capture. Legacy typed discovery continues using its existing policy.
func (a *API) captureRecord(ctx context.Context) (*preparedRecordSource, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	source, err := a.capture(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, resourceBase, base, version, serviceType := source.source.Endpoint, source.source.ResourceBase, source.base, source.source.Microversion, source.source.Type
	outer := rest.OperationGuard(ctx)
	var observed error
	check := func(checkCtx context.Context) error {
		if observed != nil {
			return cloudread.ContextError(checkCtx, observed)
		}
		var route, ancestor error
		if source.source.Endpoint != endpoint || source.source.ResourceBase != resourceBase || source.source.ServiceURL() != base || source.source.Microversion != version || source.source.Type != serviceType {
			route = infoInvalid("record source target or service facts changed")
		}
		if outer != nil {
			ancestor = outer(checkCtx)
		}
		observed = errors.Join(route, source.check(checkCtx), ancestor)
		return cloudread.ContextError(checkCtx, observed)
	}
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return nil, err
	}
	location := json.RawMessage("null")
	if a.dependencies.CloudLocation != nil {
		facts, readErr := a.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return nil, err
	}
	return &preparedRecordSource{preparedSource: source, ctx: opctx, check: check, location: location}, nil
}
func discoveryRecordCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
