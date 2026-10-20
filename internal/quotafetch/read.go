package quotafetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type Profile struct {
	Kind, Role, Ceiling string
	IntegerFields       []string
	NovaDetail          bool
}

func Read(ctx context.Context, client *gophercloud.ServiceClient, projectID, userID string, profile Profile, options ...Option) (*Record, error) {
	profile.IntegerFields = slices.Clone(profile.IntegerFields)
	source, err := cloudread.Capture(ctx, client, profile.Role)
	if err != nil {
		return nil, err
	}
	if !text(projectID) || strings.TrimSpace(projectID) == "" || projectID == "." || projectID == ".." {
		return nil, invalid("quota project must be a nonblank literal ID")
	}
	if userID != "" && (!text(userID) || strings.TrimSpace(userID) == "" || userID == "." || userID == "..") {
		return nil, invalid("quota user must be a nonblank literal ID")
	}
	check := func() error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	c, err := applyOptions(profile, userID, options, check)
	if err = errors.Join(err, source.Guard(ctx)); err != nil {
		return nil, err
	}
	version := source.Client.Microversion
	var discovery microversions.Discovery
	if c.Microversion != nil {
		version = *c.Microversion
	} else if version == "" && profile.Ceiling != "" {
		discovery, err = microversions.Read(ctx, source, microversions.NovaProfile)
		if err != nil {
			return nil, err
		}
		version, err = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, profile.Ceiling)
		if err != nil {
			return nil, err
		}
	}
	target, err := targetURL(&source.Client, projectID, profile, c)
	if err != nil {
		return nil, err
	}
	query := make(url.Values)
	for key, raw := range c.Query {
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, err
		}
		if len(values) > 0 {
			query[key] = values
		}
	}
	if !profile.NovaDetail {
		if c.usageValue != nil {
			values, err := cloudfilter.RequestQueryValues(c.usageValue)
			if err != nil {
				return nil, err
			}
			if len(values) > 0 {
				query["usage"] = values
			}
		} else {
			query.Set("usage", "False")
			if c.Usage {
				query.Set("usage", "True")
			}
		}
	}
	if userID != "" {
		query.Set("user_id", userID)
	}
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	op := source.Client
	op.Microversion = version
	headers := make(map[string]string)
	legacy := "X-OpenStack-Volume-API-Version"
	if profile.NovaDetail {
		legacy = "X-OpenStack-Nova-API-Version"
	}
	for key := range op.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, legacy) {
			delete(op.MoreHeaders, key)
		}
	}
	policy := rest.RequestHeaderPolicy{Values: headers}
	if version != "" {
		headers["OpenStack-API-Version"] = profile.Role + " " + version
		headers[legacy] = version
	} else {
		policy.Absent = []string{"OpenStack-API-Version", legacy}
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = i + 400
	}
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	wire, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &op, guard, http.MethodGet, target, nil, policy, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes...)
	if err != nil {
		var observed *Record
		if wire != nil {
			observed = observedRecord(wire, projectID, userID, version, discovery)
		}
		if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == 404 {
			err = &resource.NotFoundError{Resource: profile.Kind, Reference: projectID, Cause: err}
		}
		if c.ErrorMessage != nil && *c.ErrorMessage != "" {
			err = fmt.Errorf("%s: %w", *c.ErrorMessage, err)
		}
		return observed, err
	}
	result, err := project(wire, projectID, userID, version, profile, c)
	if err = cloudread.ContextError(ctx, errors.Join(err, check())); err != nil {
		return observedRecord(wire, projectID, userID, version, discovery), wire.Fail(err)
	}
	result.Discovery = discovery
	return result, nil
}
func targetURL(client *gophercloud.ServiceClient, projectID string, profile Profile, c Options) (string, error) {
	suffix := ""
	if profile.NovaDetail && c.Usage {
		suffix = "/detail"
	}
	path := "os-quota-sets/" + url.PathEscape(projectID) + suffix
	if c.BasePath != nil {
		path = strings.ReplaceAll(*c.BasePath, "%(project_id)s", url.PathEscape(projectID))
		path = strings.ReplaceAll(path, "%(id)s", "None")
		if strings.Contains(path, "%(") {
			return "", invalid("unknown quota base_path substitution")
		}
		// Adapter-relative paths are rooted in the effective service base, including
		// reverse-proxy prefixes. Absolute same-origin targets remain explicitly owned.
		if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
			path = strings.TrimLeft(path, "/")
		}
	}
	target := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		target = client.ServiceURL(path)
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(parsed.EscapedPath(), "%2F..") {
		return "", invalid("quota base_path must be a fixed path without query or fragment")
	}
	if err := rest.ValidateTarget(client, target); err != nil {
		return "", err
	}
	return target, nil
}
