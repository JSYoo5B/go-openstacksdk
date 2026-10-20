package aggregates

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PrecacheImagesOpts carries transport headers; the action owns version 2.81.
type PrecacheImagesOpts struct{}
type PrecacheImagesOption = request.Option[PrecacheImagesOpts]

func WithPrecacheImagesHeader(key, value string) PrecacheImagesOption {
	return request.WithHeader[PrecacheImagesOpts](key, value)
}

// PrecacheImages sends aggregate_precache_images. A single image uses
// []string{image}; an empty list sends an empty cache array. IDs are literal.
// It does not resolve images or decode the status-only response body.
func (a *API) PrecacheImages(ctx context.Context, aggregateID int, images []string, options ...PrecacheImagesOption) error {
	fail := func(err error) error {
		return request.Wrap("PrecacheImages", "aggregates", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if a == nil {
		return fail(fmt.Errorf("%w: aggregate API is required", resource.ErrInvalidOption))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return fail(err)
	}
	var observed error
	guard := func(callCtx context.Context) error {
		if observed == nil && a.client != selected {
			observed = fmt.Errorf("%w: selected aggregate API changed", resource.ErrInvalidOption)
		}
		return errors.Join(cloudread.ContextError(callCtx, observed), source.Guard(callCtx))
	}
	cache := make([]map[string]string, len(images))
	for i, id := range images {
		cache[i] = map[string]string{"id": id}
	}
	cfg, err := cloudread.ApplyReadOptions(ctx, PrecacheImagesOpts{}, options, cloudread.OwnReadConfig[PrecacheImagesOpts], guard)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	version := "2.81"
	if err == nil {
		// Capture validated inherited aliases against the original client. The
		// action replaces them only on its owned copy before fixing version 2.81.
		source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
		err = source.WithPolicy(ctx, &version, cfg.Headers)
	}
	if err != nil {
		return fail(err)
	}
	target := source.Client.ServiceURL("os-aggregates", url.PathEscape(strconv.Itoa(aggregateID)), "images")
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	headers := maps.Clone(source.Client.MoreHeaders)
	if headers == nil {
		headers = map[string]string{}
	}
	headers["OpenStack-API-Version"] = "compute 2.81"
	headers["X-OpenStack-Nova-API-Version"] = "2.81"
	_, err = rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodPost, target, map[string]any{"cache": cache}, rest.RequestHeaderPolicy{Values: headers}, rest.RejectionPolicy{Codes: rejected, StopFaultRetryCallbacks: true}, codes...)
	return fail(errors.Join(err, guard(ctx)))
}
