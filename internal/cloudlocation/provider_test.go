package cloudlocation_test

import (
	"encoding/json"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	v2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/resource"
	"net/http"
	"testing"
)

type recordedLocationOpaqueResult struct{}

func (recordedLocationOpaqueResult) ExtractTokenID() (string, error) { return "opaque", nil }

func TestRecordedCloudLocationNativeScopeShapesNeedNoExpiryOrHTTP(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	makeV3 := func() v3.CreateResult {
		value := v3.CreateResult{}
		value.Header = http.Header{"X-Subject-Token": {"recorded"}}
		value.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "v3-project"}}}
		return value
	}
	makeV2 := func() v2.CreateResult {
		value := v2.CreateResult{}
		value.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "recorded", "expires": "deliberately invalid", "tenant": map[string]any{"id": "v2-project"}}}}
		return value
	}
	a, b := makeV3(), makeV2()
	c, d := v3.GetResult{}, v2.GetResult{CreateResult: makeV2()}
	c.Header = a.Header
	c.Body = a.Body
	for _, tc := range []struct {
		result gophercloud.AuthResult
		want   string
	}{{a, `"v3-project"`}, {&a, `"v3-project"`}, {c, `"v3-project"`}, {&c, `"v3-project"`}, {b, `"v2-project"`}, {&b, `"v2-project"`}, {d, `"v2-project"`}, {&d, `"v2-project"`}, {recordedLocationOpaqueResult{}, ""}} {
		if err := provider.SetTokenAndAuthResult(tc.result); err != nil {
			t.Fatal(err)
		}
		raw, err := cloudlocation.ProjectID(provider)
		if err != nil || string(raw) != tc.want {
			t.Fatal(string(raw), tc.want, err)
		}
		if tc.want != "" && !json.Valid(raw) {
			t.Fatal(string(raw))
		}
	}
	provider.SetToken("manual-token")
	if raw, err := cloudlocation.ProjectID(provider); raw != nil || err != nil {
		t.Fatal(string(raw), err)
	}
	unscoped := makeV3()
	unscoped.Body = map[string]any{"token": map[string]any{"domain": map[string]any{"id": "domain"}}}
	if err := provider.SetTokenAndAuthResult(unscoped); err != nil {
		t.Fatal(err)
	}
	if raw, err := cloudlocation.ProjectID(provider); raw != nil || err != nil {
		t.Fatal("domain scope became project", string(raw), err)
	}
}

func TestRecordedCloudLocationPreservesNativeExtractionCause(t *testing.T) {
	if raw, err := cloudlocation.ProjectID(nil); raw != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(string(raw), err)
	}
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	sentinel := errors.New("recorded auth extraction failed")
	value := &v3.CreateResult{}
	value.Header = http.Header{"X-Subject-Token": {"recorded"}}
	if err := provider.SetTokenAndAuthResult(value); err != nil {
		t.Fatal(err)
	}
	value.Err = sentinel
	if raw, err := cloudlocation.ProjectID(provider); raw != nil || !errors.Is(err, sentinel) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(string(raw), err)
	}
}
