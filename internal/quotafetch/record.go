package quotafetch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type Record struct {
	Resource, Wire                  *resource.RawResource
	Envelope                        json.RawMessage
	Header                          http.Header
	StatusCode                      int
	ProjectID, UserID, Microversion string
	Discovery                       microversions.Discovery
}

func project(response *rest.Response, projectID, userID, version string, profile Profile, c Options) (*Record, error) {
	r := &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, ProjectID: projectID, UserID: userID, Microversion: version}
	normalized := make(map[string]json.RawMessage)
	if !utf8.Valid(response.Body) {
		return nil, fmt.Errorf("quota response must be UTF8 JSON or text")
	}
	raw := bytes.TrimSpace(response.Body)
	// The pinned quota translator treats JSON parse failure as a bodyless fetch.
	// Actual non-JSON bytes still remain in Envelope; they do not become Wire.
	if len(raw) > 0 && utf8.Valid(raw) && json.Valid(raw) {
		members, err := cloudfilter.ObjectMembers(raw)
		if err != nil {
			return nil, err
		}
		original := make(map[string]json.RawMessage, len(members))
		for _, m := range members {
			original[m.Key] = bytes.Clone(m.Value)
		}
		if body, exists := original["quota_set"]; exists {
			members, err = cloudfilter.ObjectMembers(body)
			if err != nil {
				return nil, err
			}
			original = make(map[string]json.RawMessage, len(members))
			for _, m := range members {
				original[m.Key] = bytes.Clone(m.Value)
			}
		}
		r.Wire = &resource.RawResource{Metadata: resource.Metadata{Body: cloneFields(original), Header: r.Header.Clone(), StatusCode: r.StatusCode}}
		normalized["usage"] = json.RawMessage("{}")
		normalized["reservation"] = json.RawMessage("{}")
		for _, m := range members {
			if m.Key == "self" {
				continue
			}
			trim := bytes.TrimSpace(m.Value)
			if len(trim) > 0 && trim[0] == '{' {
				values, err := cloudfilter.ObjectMembers(m.Value)
				if err != nil {
					return nil, err
				}
				nested := make(map[string]json.RawMessage, len(values))
				for _, v := range values {
					nested[v.Key] = v.Value
				}
				for _, field := range []struct{ wire, view string }{{"in_use", "usage"}, {"reserved", "reservation"}} {
					if value, exists := nested[field.wire]; exists {
						var dictionary map[string]json.RawMessage
						if err := json.Unmarshal(normalized[field.view], &dictionary); err != nil || dictionary == nil {
							return nil, fmt.Errorf("quota %s cannot receive a nested %s value", field.view, field.wire)
						}
						dictionary[m.Key] = bytes.Clone(value)
						normalized[field.view], _ = json.Marshal(dictionary)
					}
				}
				if limit, exists := nested["limit"]; exists {
					normalized[m.Key] = bytes.Clone(limit)
				}
			} else {
				normalized[m.Key] = bytes.Clone(m.Value)
			}
		}
	}
	for _, key := range profile.IntegerFields {
		value, exists := normalized[key]
		if !exists {
			normalized[key] = json.RawMessage("null")
			continue
		}
		converted, err := jsonfilter.DescriptorIntegerJSON(value)
		if err != nil {
			return nil, fmt.Errorf("quota integer %s: %w", key, err)
		}
		normalized[key] = converted
	}
	if profile.NovaDetail {
		if value, exists := normalized["force"]; exists {
			if !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				truth, err := cloudfilter.PythonJSONTruthy(value)
				if err != nil {
					return nil, err
				}
				normalized["force"], _ = json.Marshal(truth)
			}
		} else {
			normalized["force"] = json.RawMessage("null")
		}
	}
	for _, key := range []string{"id", "name", "usage", "reservation"} {
		value, exists := normalized[key]
		if !exists {
			normalized[key] = json.RawMessage("null")
			continue
		}
		if (key == "usage" || key == "reservation") && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) && bytes.TrimSpace(value)[0] != '{' {
			normalized[key] = json.RawMessage("{}")
		}
	}
	// location is an SDK-owned connection projection; actual response extensions
	// are preserved separately. Selected microversion owns Resource.microversion.
	normalized["location"] = json.RawMessage("null")
	if _, exists := normalized["project_id"]; exists {
		normalized["project_id"], _ = json.Marshal(projectID)
	}
	if c.Location != nil {
		scope, _ := json.Marshal(projectID)
		location, err := c.Location.ForResource(scope, normalized["availability_zone"])
		if err != nil {
			return nil, err
		}
		normalized["location"] = location
	}
	if _, present := normalized["microversion"]; present {
		normalized["microversion"] = json.RawMessage("null")
		if version != "" {
			normalized["microversion"], _ = json.Marshal(version)
		}
	}
	r.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: cloneFields(normalized), Header: r.Header.Clone(), StatusCode: r.StatusCode}}
	return r, nil
}

func observedRecord(response *rest.Response, projectID, userID, version string, discovery microversions.Discovery) *Record {
	return &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, ProjectID: projectID, UserID: userID, Microversion: version, Discovery: discovery}
}
