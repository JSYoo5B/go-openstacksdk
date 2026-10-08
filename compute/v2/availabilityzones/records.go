package availabilityzones

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AvailabilityZoneRecord separates passive source fields from the actual row.
// Missing descriptors are null; a zone name never synthesizes an ID.
type AvailabilityZoneRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *AvailabilityZoneRecord) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("%w: availability zone record is required", resource.ErrInvalidOption)
	}
	if r.Wire == nil {
		r.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, r.Wire); err != nil {
		return err
	}
	r.Envelope = bytes.Clone(data)
	r.Resource = nil
	r.Header = nil
	r.StatusCode = 0
	return nil
}

type AvailabilityZoneListOpts struct{ Microversion *string }
type AvailabilityZoneListOption = request.Option[AvailabilityZoneListOpts]

func ownOptions(c *request.Config[AvailabilityZoneListOpts]) {
	cloudread.OwnReadConfig(c)
	if c.Options.Microversion != nil {
		value := *c.Options.Microversion
		c.Options.Microversion = &value
	}
}
func WithAvailabilityZoneListOptions(value AvailabilityZoneListOpts) AvailabilityZoneListOption {
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return func(c *request.Config[AvailabilityZoneListOpts]) error { c.Options = value; ownOptions(c); return nil }
}
func WithAvailabilityZoneMicroversion(value string) AvailabilityZoneListOption {
	return func(c *request.Config[AvailabilityZoneListOpts]) error {
		owned := value
		c.Options.Microversion = &owned
		return nil
	}
}
func WithAvailabilityZoneHeader(key, value string) AvailabilityZoneListOption {
	return request.WithHeader[AvailabilityZoneListOpts](key, value)
}

// ListRecords lazily reads ordinary zones. The privileged detail branch is
// exposed separately by the existing native ListDetail facade.
func (a *API) ListRecords(ctx context.Context, options ...AvailabilityZoneListOption) iter.Seq2[*AvailabilityZoneRecord, error] {
	captured := slices.Clone(options)
	return func(yield func(*AvailabilityZoneRecord, error) bool) {
		fail := func(err error) {
			yield(nil, request.Wrap("ListRecords", "availabilityzones", cloudread.ContextError(ctx, err)))
		}
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(fmt.Errorf("%w: availability zone API is required", resource.ErrInvalidOption))
			return
		}
		original := a.client
		source, err := cloudread.Capture(ctx, original, "compute")
		if err != nil {
			fail(err)
			return
		}
		outer := rest.OperationGuard(ctx)
		check := func(ctx context.Context) error {
			var changed, parent error
			if a.client != original {
				changed = fmt.Errorf("%w: availability zone API changed", resource.ErrInvalidOption)
			}
			if outer != nil {
				parent = outer(ctx)
			}
			return errors.Join(source.Guard(ctx), changed, parent)
		}
		opctx := rest.WithOperationGuard(ctx, check)
		config, err := cloudread.ApplyReadOptions(opctx, AvailabilityZoneListOpts{}, captured, ownOptions, check)
		if err == nil {
			err = request.ValidateCapabilities(config, false, false, true)
		}
		if err == nil {
			err = source.WithPolicy(opctx, config.Options.Microversion, config.Headers)
		}
		if err != nil {
			fail(err)
			return
		}
		if _, present := source.Client.MoreHeaders["Accept"]; !present {
			if source.Client.MoreHeaders == nil {
				source.Client.MoreHeaders = make(map[string]string)
			}
			source.Client.MoreHeaders["Accept"] = "application/json"
		}
		codes := make([]int, 200)
		for i := range codes {
			codes[i] = i + 200
		}
		spec := rest.CollectionSpec[AvailabilityZoneRecord]{
			Client: &source.Client, Path: "os-availability-zone", Kind: "availabilityzones", PluralKey: "availabilityZoneInfo",
			Validate: check, SourceGuard: check, ListCodes: codes,
			Metadata: func(r *AvailabilityZoneRecord) *resource.Metadata {
				if r.Wire == nil {
					r.Wire = &resource.RawResource{}
				}
				return &r.Wire.Metadata
			},
			ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
				return microversions.MemberGet(ctx, source, target, source.Client.Microversion, microversions.NovaProfile, codes...)
			},
			ValidateItem: func(r *AvailabilityZoneRecord) error {
				if err := check(opctx); err != nil {
					return err
				}
				view := r.Wire.Clone()
				view.Body = make(map[string]json.RawMessage, 5)
				for _, key := range []string{"id", "name", "state", "hosts"} {
					view.Body[key] = json.RawMessage("null")
				}
				members, err := cloudfilter.ObjectMembers(r.Envelope)
				if err != nil {
					return err
				}
				for _, member := range members {
					key := member.Key
					switch key {
					case "zoneName":
						key = "name"
					case "zoneState":
						key = "state"
					case "id", "name", "state", "hosts":
					default:
						continue
					}
					raw, err := resource.BodyRecordField(map[string]json.RawMessage{key: member.Value}, key, resource.BodyFieldJSON)
					if err != nil {
						return err
					}
					view.Body[key] = raw
				}
				view.Body["location"] = json.RawMessage("null")
				r.Resource = view
				r.Header = r.Wire.Header.Clone()
				r.StatusCode = r.Wire.StatusCode
				return check(opctx)
			},
			Paging: rest.PagePolicy[AvailabilityZoneRecord]{LinkKeys: []string{"links", "availabilityZoneInfo_links"}, HTTPLink: true, DictionaryLinks: true, AllowFirstServerLimit: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		for record, err := range rest.List(opctx, spec, nil) {
			err = errors.Join(err, check(opctx))
			if !yield(record, request.Wrap("ListRecords", "availabilityzones", cloudread.ContextError(ctx, err))) || err != nil {
				return
			}
			if err := check(opctx); err != nil {
				fail(err)
				return
			}
		}
		if err := check(opctx); err != nil {
			fail(err)
		}
	}
}
