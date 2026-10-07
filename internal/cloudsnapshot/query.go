package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type listPolicy struct {
	detailed, paginated bool
	query               map[string]json.RawMessage
	local               []cloudfilter.JSONMember
	limit, maximum      json.RawMessage
	microversion        *string
	headers             map[string]string
	expression          *string
}

// compileList follows the distinct Cloud -> Proxy -> Resource binding stages.
// Explicit Go Resource controls override the final raw/injected controls;
// already-bound Proxy controls cannot erase later argument collisions.
func compileList(options ListOptions) (*listPolicy, error) {
	return compileListFor(options, snapshotReadSchema())
}

func compileListFor(options ListOptions, schema readSchema) (*listPolicy, error) {
	options = cloneList(options)
	policy := &listPolicy{detailed: true, paginated: true, query: make(map[string]json.RawMessage), headers: map[string]string{"Accept": "application/json"}}
	attrs, err := snapshotQueryMembers(options.Filters)
	if err != nil {
		return nil, err
	}
	for _, member := range attrs {
		switch member.Key {
		case "details", "base_path", "resource_type", "self":
			return nil, snapshotQueryError("raw filters collide with Proxy argument %q", member.Key)
		}
	}
	if options.Detailed != nil {
		policy.detailed = *options.Detailed
	}
	if schema.bindAllProjects {
		allProjects, exists := snapshotQueryTake(&attrs, "all_projects")
		if exists {
			truthy, err := cloudfilter.PythonTruthy(allProjects)
			if err != nil {
				return nil, snapshotQueryCause("all_projects", err)
			}
			if truthy {
				snapshotQuerySet(&attrs, "all_projects", json.RawMessage("true"))
			}
		}
	}
	paginated, exists := snapshotQueryTake(&attrs, "paginated")
	if exists {
		policy.paginated, err = cloudfilter.PythonTruthy(paginated)
		if err != nil {
			return nil, snapshotQueryCause("paginated", err)
		}
	}
	if options.Paginated != nil {
		policy.paginated = *options.Paginated
	}
	expression, exists := snapshotQueryTake(&attrs, "jmespath_filters")
	if exists && options.Expression == nil {
		policy.expression, err = snapshotQueryExpression(expression)
		if err != nil {
			return nil, err
		}
	}
	if options.Expression != nil {
		if !utf8.ValidString(*options.Expression) {
			return nil, snapshotQueryError("expression must be UTF-8")
		}
		policy.expression = clonePointer(options.Expression)
	}
	if options.ConflictingAttrs != nil {
		if err := snapshotQueryDocument(*options.ConflictingAttrs); err != nil {
			return nil, snapshotQueryCause("__conflicting_attrs", err)
		}
		snapshotQuerySet(&attrs, "__conflicting_attrs", *options.ConflictingAttrs)
	}
	conflicting, exists := snapshotQueryLookup(attrs, "__conflicting_attrs")
	if exists {
		truthy, err := cloudfilter.PythonTruthy(conflicting)
		if err != nil {
			return nil, snapshotQueryCause("__conflicting_attrs", err)
		}
		if truthy {
			members, err := cloudfilter.ObjectMembers(conflicting)
			if err != nil {
				return nil, snapshotQueryCause("__conflicting_attrs must be a mapping", err)
			}
			for _, member := range members {
				snapshotQuerySet(&attrs, member.Key, member.Value)
			}
		}
		snapshotQueryTake(&attrs, "__conflicting_attrs")
	}
	for _, member := range attrs {
		switch member.Key {
		case "paginated", "base_path", "session", "cls":
			return nil, snapshotQueryError("filters collide with Resource argument %q", member.Key)
		}
	}
	// Go fields override final Resource-bound raw counterparts, including
	// values introduced by the late __conflicting_attrs mapping.
	if options.MaxItems != nil {
		snapshotQuerySet(&attrs, "max_items", json.RawMessage(strconv.Itoa(*options.MaxItems)))
	}
	if options.Microversion != nil {
		if !utf8.ValidString(*options.Microversion) {
			return nil, snapshotQueryError("microversion must be UTF-8")
		}
		raw, _ := json.Marshal(*options.Microversion)
		snapshotQuerySet(&attrs, "microversion", raw)
	}
	if options.Headers != nil {
		for key, value := range options.Headers {
			if !utf8.ValidString(key) || !utf8.ValidString(value) {
				return nil, snapshotQueryError("headers must contain UTF-8 strings")
			}
		}
		raw, err := json.Marshal(options.Headers)
		if err != nil {
			return nil, snapshotQueryCause("headers", err)
		}
		snapshotQuerySet(&attrs, "headers", raw)
	}
	if maximum, exists := snapshotQueryTake(&attrs, "max_items"); exists {
		if _, err := maxReached(0, maximum); err != nil {
			return nil, err
		}
		policy.maximum = bytes.Clone(maximum)
	}
	if version, exists := snapshotQueryTake(&attrs, "microversion"); exists {
		policy.microversion, err = snapshotQueryString(version, "microversion")
		if err != nil {
			return nil, err
		}
	}
	if headers, exists := snapshotQueryTake(&attrs, "headers"); exists {
		values, err := snapshotQueryHeaders(headers)
		if err != nil {
			return nil, err
		}
		for key, value := range values {
			policy.headers[key] = value
		}
	}
	// This Source argument is observable but Resource.list always validates
	// query with allow_unknown_params=True. Every JSON value is a no-op.
	snapshotQueryTake(&attrs, "allow_unknown_params")
	queryNames := []string{"limit", "marker", "name", "status", "volume_id", "project_id", "offset", "sort_dir", "sort_key", "sort"}
	queryKeys := make(map[string]bool, len(queryNames)+1)
	for _, name := range queryNames {
		queryKeys[name] = true
		if raw, exists := snapshotQueryLookup(attrs, name); exists {
			policy.query[name] = bytes.Clone(raw)
		}
	}
	queryKeys["all_projects"] = true
	if raw, exists := snapshotQueryLookup(attrs, "all_projects"); exists {
		policy.query["all_tenants"] = bytes.Clone(raw)
	} else if raw, exists := snapshotQueryLookup(attrs, "all_tenants"); exists {
		policy.query["all_tenants"] = bytes.Clone(raw)
	}
	for _, member := range attrs {
		if queryKeys[member.Key] {
			continue
		}
		for _, attribute := range schema.bodyAttributes {
			if member.Key == attribute {
				policy.local = append(policy.local, cloudfilter.JSONMember{Key: member.Key, Value: bytes.Clone(member.Value)})
				break
			}
		}
	}
	maximumTruthy, err := cloudfilter.PythonTruthy(policy.maximum)
	if err != nil {
		return nil, snapshotQueryCause("max_items", err)
	}
	limitTruthy, err := cloudfilter.PythonTruthy(policy.query["limit"])
	if err != nil {
		return nil, snapshotQueryCause("limit", err)
	}
	if maximumTruthy && !limitTruthy {
		policy.query["limit"] = bytes.Clone(policy.maximum)
	}
	policy.limit = bytes.Clone(policy.query["limit"])
	return policy, nil
}

func encodeQuery(query map[string]json.RawMessage) (url.Values, error) {
	result := make(url.Values)
	for key, raw := range query {
		if !utf8.ValidString(key) {
			return nil, snapshotQueryError("query key must be UTF-8")
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, snapshotQueryCause("query "+key, err)
		}
		for _, value := range values {
			result.Add(key, value)
		}
	}
	return result, nil
}

func snapshotQueryMembers(raw *json.RawMessage) ([]cloudfilter.JSONMember, error) {
	if raw == nil {
		return nil, nil
	}
	if err := snapshotQueryDocument(*raw); err != nil {
		return nil, snapshotQueryCause("filters", err)
	}
	truthy, err := cloudfilter.PythonTruthy(*raw)
	if err != nil {
		return nil, snapshotQueryCause("filters", err)
	}
	if !truthy {
		return nil, nil
	}
	members, err := cloudfilter.ObjectMembers(*raw)
	if err != nil {
		return nil, snapshotQueryCause("filters must be a mapping", err)
	}
	return members, nil
}

func snapshotQueryDocument(raw json.RawMessage) error {
	if raw == nil || len(raw) == 0 {
		return fmt.Errorf("explicit raw value must be complete JSON")
	}
	_, err := cloudfilter.PythonTruthy(raw)
	return err
}

func snapshotQueryLookup(members []cloudfilter.JSONMember, key string) (json.RawMessage, bool) {
	for _, member := range members {
		if member.Key == key {
			return member.Value, true
		}
	}
	return nil, false
}
func snapshotQueryTake(members *[]cloudfilter.JSONMember, key string) (json.RawMessage, bool) {
	for i, member := range *members {
		if member.Key == key {
			value := member.Value
			copy((*members)[i:], (*members)[i+1:])
			*members = (*members)[:len(*members)-1]
			return value, true
		}
	}
	return nil, false
}
func snapshotQuerySet(members *[]cloudfilter.JSONMember, key string, value json.RawMessage) {
	for i := range *members {
		if (*members)[i].Key == key {
			(*members)[i].Value = bytes.Clone(value)
			return
		}
	}
	*members = append(*members, cloudfilter.JSONMember{Key: key, Value: bytes.Clone(value)})
}
func snapshotQueryString(raw json.RawMessage, control string) (*string, error) {
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, snapshotQueryCause(control+" must be a string or null", err)
	}
	return &value, nil
}
func snapshotQueryExpression(raw json.RawMessage) (*string, error) {
	truthy, err := cloudfilter.PythonTruthy(raw)
	if err != nil {
		return nil, snapshotQueryCause("jmespath_filters", err)
	}
	if !truthy {
		return nil, nil
	}
	return snapshotQueryString(raw, "jmespath_filters")
}
func snapshotQueryHeaders(raw json.RawMessage) (map[string]string, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, snapshotQueryCause("headers must be a string mapping or null", err)
	}
	result := make(map[string]string, len(members))
	for _, member := range members {
		var value string
		if bytes.Equal(bytes.TrimSpace(member.Value), []byte("null")) {
			return nil, snapshotQueryError("header %q must be a string", member.Key)
		}
		if err := json.Unmarshal(member.Value, &value); err != nil {
			return nil, snapshotQueryCause("header "+member.Key+" must be a string", err)
		}
		key := http.CanonicalHeaderKey(member.Key)
		if previous, exists := result[key]; exists && previous != value {
			return nil, snapshotQueryError("conflicting header aliases %q", member.Key)
		}
		result[key] = value
	}
	return result, nil
}

// maxReached compares a nonnegative consumed-row counter with Source's raw
// bool/number maximum, without float rounding or exponent-sized allocations.
// Zero/null/false is unlimited; a negative nonzero maximum stops at count0.
func maxReached(count int, maximum json.RawMessage) (bool, error) {
	if count < 0 {
		return false, snapshotQueryError("consumed row count must be nonnegative")
	}
	if maximum == nil {
		return false, nil
	}
	if err := snapshotQueryDocument(maximum); err != nil {
		return false, snapshotQueryCause("max_items", err)
	}
	raw := bytes.TrimSpace(maximum)
	if bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("false")) {
		return false, nil
	}
	if bytes.Equal(raw, []byte("true")) {
		return count >= 1, nil
	}
	if raw[0] != '-' && (raw[0] < '0' || raw[0] > '9') {
		return false, snapshotQueryError("max_items must be a number, boolean or null")
	}
	truthy, _ := cloudfilter.PythonTruthy(raw)
	if !truthy {
		return false, nil
	}
	if raw[0] == '-' {
		return true, nil
	}
	if count == 0 {
		return false, nil
	}
	digits, exponent := snapshotMaximumDecimal(string(raw))
	magnitude := new(big.Int).Add(exponent, big.NewInt(int64(len(digits))))
	counter := strconv.Itoa(count)
	if comparison := magnitude.Cmp(big.NewInt(int64(len(counter)))); comparison != 0 {
		return comparison < 0, nil
	}
	width := len(digits)
	if len(counter) > width {
		width = len(counter)
	}
	for i := 0; i < width; i++ {
		actual, expected := byte('0'), byte('0')
		if i < len(counter) {
			actual = counter[i]
		}
		if i < len(digits) {
			expected = digits[i]
		}
		if actual != expected {
			return actual > expected, nil
		}
	}
	return true, nil
}
func snapshotMaximumDecimal(text string) (string, *big.Int) {
	exponent := new(big.Int)
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		exponent.SetString(text[i+1:], 10)
		text = text[:i]
	}
	fraction := 0
	if i := strings.IndexByte(text, '.'); i >= 0 {
		fraction = len(text) - i - 1
		text = text[:i] + text[i+1:]
	}
	text = strings.TrimLeft(text, "0")
	digits := strings.TrimRight(text, "0")
	exponent.Add(exponent, big.NewInt(int64(len(text)-len(digits)-fraction)))
	return digits, exponent
}
func snapshotQueryError(format string, args ...any) error {
	return fmt.Errorf("%w: Cinder resource list: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func snapshotQueryCause(control string, cause error) error {
	return fmt.Errorf("%w: Cinder resource list %s: %w", resource.ErrInvalidOption, control, cause)
}
