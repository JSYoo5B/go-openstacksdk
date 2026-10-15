package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CloudConsoleOpts owns the action's optional JSON length. A nil pointer or
// explicit JSON null omits it; every other finite JSON value is sent literally.
type CloudConsoleOpts struct{ Length *json.RawMessage }
type CloudConsoleOption func(*CloudConsoleOpts) error

func WithCloudConsoleOpts(value CloudConsoleOpts) CloudConsoleOption {
	owned := cloneCloudConsoleOpts(value)
	return func(c *CloudConsoleOpts) error { *c = cloneCloudConsoleOpts(owned); return nil }
}
func WithCloudConsoleLength(value int) CloudConsoleOption {
	raw, _ := json.Marshal(value)
	return WithCloudConsoleLengthJSON(raw)
}
func WithCloudConsoleLengthJSON(value json.RawMessage) CloudConsoleOption {
	owned := bytes.Clone(value)
	return func(c *CloudConsoleOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		c.Length = &raw
		return nil
	}
}
func WithoutCloudConsoleLength() CloudConsoleOption {
	return func(c *CloudConsoleOpts) error { c.Length = nil; return nil }
}
func cloneCloudConsoleOpts(value CloudConsoleOpts) CloudConsoleOpts {
	if value.Length != nil {
		raw := json.RawMessage(bytes.Clone(*value.Length))
		value.Length = &raw
	}
	return value
}

// CloudConsoleRecord keeps the uncoerced output value and its presence separate
// from the actual response envelope. Discovery has its own HTTP receipts.
type CloudConsoleRecord struct {
	Value        json.RawMessage
	HasOutput    bool
	Envelope     json.RawMessage
	Header       http.Header
	StatusCode   int
	Microversion string
	Discovery    []ServerFindObservation
}

// ConsoleOutputRecord performs only the console action. It does not find a
// server and does not suppress HTTP 400. An empty ID deliberately preserves
// the Source falsey-ID route /servers//action; other IDs are literal segments.
func (a *API) ConsoleOutputRecord(ctx context.Context, serverID string, options ...CloudConsoleOption) (*CloudConsoleRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("ConsoleOutputRecord", "servers", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(consoleRecordInvalid("server API is required"))
	}
	serverID = strings.Trim(serverID, "/")
	if !utf8.ValidString(serverID) || serverID == "." || serverID == ".." || strings.IndexFunc(serverID, unicode.IsControl) >= 0 {
		return nil, wrap(consoleRecordInvalid("server ID must be literal UTF-8 without controls or dot segments"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, wrap(err)
	}
	var changed error
	ownGuard := func(c context.Context) error {
		if changed == nil && a.client != selected {
			changed = consoleRecordInvalid("selected server API changed")
		}
		return errors.Join(changed, source.Guard(c))
	}
	ctx = rest.WithOperationGuard(ctx, ownGuard)
	guard := func(c context.Context) error { return errors.Join(ownGuard(c), rest.CheckOperationGuard(c)) }
	config := CloudConsoleOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(consoleRecordInvalid("nil console action option"))
		}
		next := cloneCloudConsoleOpts(config)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		config = cloneCloudConsoleOpts(next)
	}
	fields := make(map[string]json.RawMessage)
	if config.Length != nil {
		raw := bytes.TrimSpace(*config.Length)
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, wrap(consoleRecordInvalid("length requires valid UTF-8 JSON"))
		}
		if !bytes.Equal(raw, []byte("null")) {
			fields["length"] = bytes.Clone(raw)
		}
	}
	result := &CloudConsoleRecord{}
	version := source.Client.Microversion
	if version == "" {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		discovery, e := microversions.Read(ctx, source, profile)
		for _, r := range discovery.Responses {
			result.Discovery = append(result.Discovery, ServerFindObservation{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode})
		}
		err = e
		if err == nil {
			version, err = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.100")
		}
		if err != nil {
			if len(discovery.Responses) > 0 {
				err = discovery.Responses[len(discovery.Responses)-1].Fail(err)
			}
			return result, wrap(errors.Join(err, guard(ctx)))
		}
	}
	result.Microversion = version
	client := source.Client
	client.Microversion = version
	client.MoreHeaders = maps.Clone(client.MoreHeaders)
	for key := range client.MoreHeaders {
		if strings.EqualFold(key, "Accept") || strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(client.MoreHeaders, key)
		}
	}
	headers := maps.Clone(client.MoreHeaders)
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Accept"] = ""
	policy := rest.RequestHeaderPolicy{Values: headers, Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}}
	if version == "" {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version", "X-OpenStack-Nova-API-Version")
	} else {
		headers["OpenStack-API-Version"] = "compute " + version
		headers["X-OpenStack-Nova-API-Version"] = version
	}
	client.MoreHeaders = maps.Clone(headers)
	body := map[string]any{"os-getConsoleOutput": fields}
	// Trim outer slashes like utils.urljoin, then keep the effective ID literal.
	target := client.ServiceURL("servers", url.PathEscape(strings.Trim(serverID, "/")), "action")
	response, prior := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &client, guard, http.MethodPost, target, body, policy, rest.RejectionPolicy{Codes: serverFindCodes(400, 1000), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, serverFindCodes(100, 400)...)
	if response != nil {
		result.Envelope = bytes.Clone(response.Body)
		result.Header = response.Header.Clone()
		result.StatusCode = response.StatusCode
	} else {
		actual, found := consoleRecordNativeReceipt(prior)
		if found {
			result.Envelope = bytes.Clone(actual.Body)
			result.Header = actual.ResponseHeader.Clone()
			result.StatusCode = actual.Actual
		}
	}
	if err := errors.Join(prior, guard(ctx)); err != nil {
		return result, wrap(err)
	}
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	var value json.RawMessage
	var present bool
	if err == nil {
		value, present, err = consoleRecordOutput(normalized)
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return result, wrap(response.Fail(err))
	}
	result.Value = bytes.Clone(value)
	result.HasOutput = present
	return result, nil
}

func consoleRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func consoleRecordOutput(raw json.RawMessage) (json.RawMessage, bool, error) {
	raw = bytes.TrimSpace(raw)
	if !utf8.Valid(raw) {
		return nil, false, fmt.Errorf("%w: console response needs charset decoding beyond the supported explicit set", resource.ErrUnsupported)
	}
	if !json.Valid(raw) {
		return nil, false, consoleRecordInvalid("console response requires UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var loaded any
	if err := decoder.Decode(&loaded); err != nil {
		return nil, false, err
	}
	if err := consoleRecordFinite(loaded); err != nil {
		return nil, false, err
	}
	switch raw[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, false, err
		}
		value, present := fields["output"]
		if !present {
			return json.RawMessage("null"), false, nil
		}
		return bytes.Clone(value), true, nil
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, false, err
		}
		for _, value := range values {
			var text string
			if json.Unmarshal(value, &text) == nil && text == "output" {
				return nil, false, consoleRecordInvalid("console array containing output cannot be indexed by string")
			}
		}
		return json.RawMessage("null"), false, nil
	case '"':
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, false, err
		}
		if strings.Contains(text, "output") {
			return nil, false, consoleRecordInvalid("console string containing output cannot be indexed by string")
		}
		return json.RawMessage("null"), false, nil
	default:
		return nil, false, consoleRecordInvalid("console response does not support output membership")
	}
}

// Raw values preserve arbitrary-size integers. Non-finite Python float results
// cannot be carried as finite JSON and remain an explicit unsupported boundary.
func consoleRecordFinite(value any) error {
	switch v := value.(type) {
	case json.Number:
		if strings.ContainsAny(string(v), ".eE") {
			f, e := strconv.ParseFloat(string(v), 64)
			if math.IsInf(f, 0) || math.IsNaN(f) {
				return fmt.Errorf("%w: console non-finite loaded Python number", resource.ErrUnsupported)
			}
			if e != nil && !errors.Is(e, strconv.ErrRange) {
				return e
			}
		}
	case []any:
		for _, item := range v {
			if err := consoleRecordFinite(item); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, item := range v {
			if err := consoleRecordFinite(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func consoleRecordNativeReceipt(err error) (gophercloud.ErrUnexpectedResponseCode, bool) {
	receipt := err
	var unable *gophercloud.ErrUnableToReauthenticate
	if errors.As(receipt, &unable) {
		receipt = unable.ErrOriginal
	}
	var after *gophercloud.ErrErrorAfterReauthentication
	if errors.As(receipt, &after) {
		receipt = after.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	ok := errors.As(receipt, &native)
	return native, ok
}
