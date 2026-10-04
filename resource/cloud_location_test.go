package resource_test

import (
	"encoding/json"
	"errors"
	"gophercloudsdk/resource"
	"testing"
)

func TestCloudLocationForeignProjectsAndRawZoneSourceTruthiness(t *testing.T) {
	name, domain, cloud, region := "configured", "domain", "dev", ""
	value := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`1`), Name: &name, DomainName: &domain}}
	for _, tc := range []struct{ id, wantID, wantName string }{
		{"null", "1", `"configured"`}, {"false", "1", `"configured"`}, {"0", "1", `"configured"`}, {`""`, "1", `"configured"`}, {"[]", "1", `"configured"`}, {"{}", "1", `"configured"`},
		{"true", "1", `"configured"`}, {"1.0", "1", `"configured"`}, {`"foreign"`, `"foreign"`, "null"}, {"[1]", "[1]", "null"}, {`{"id":1}`, `{"id":1}`, "null"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			raw, err := value.ForResource(json.RawMessage(tc.id), json.RawMessage(`false`))
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Cloud, RegionName json.RawMessage
				Zone              json.RawMessage
				Project           map[string]json.RawMessage
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(raw, &fields)
			_ = json.Unmarshal(fields["project"], &got.Project)
			if string(got.Project["id"]) != tc.wantID || string(got.Project["name"]) != tc.wantName || string(fields["zone"]) != "false" || string(fields["region_name"]) != `""` || string(fields["cloud"]) != `"dev"` {
				t.Fatal(string(raw))
			}
			if tc.wantName == "null" && (string(got.Project["domain_name"]) != "null" || string(got.Project["domain_id"]) != "null") {
				t.Fatal(string(raw))
			}
		})
	}
	if raw, err := value.ForResource(nil, nil); err != nil || !json.Valid(raw) {
		t.Fatal(string(raw), err)
	}
}

func TestCloudLocationCloneAndComputedSnapshotOwnAllPointersAndBytes(t *testing.T) {
	cloud, region, name, domainID, domainName := "cloud", "region", "name", "domain-id", "domain-name"
	value := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`{"zone":1}`), Project: resource.CloudProject{ID: json.RawMessage(`"p"`), Name: &name, DomainID: &domainID, DomainName: &domainName}}
	copy := value.Clone()
	raw, err := value.ForResource(json.RawMessage(`"p"`), json.RawMessage(`{"zone":2}`))
	if err != nil {
		t.Fatal(err)
	}
	*copy.Cloud = "changed"
	*copy.RegionName = "changed"
	*copy.Project.Name = "changed"
	*copy.Project.DomainID = "changed"
	*copy.Project.DomainName = "changed"
	copy.Zone[0] = '!'
	copy.Project.ID[0] = '!'
	if cloud != "cloud" || region != "region" || name != "name" || domainID != "domain-id" || domainName != "domain-name" || !json.Valid(value.Zone) || !json.Valid(value.Project.ID) {
		t.Fatal("clone aliases original", value)
	}
	value.Project.ID[0] = '!'
	value.Zone[0] = '!'
	cloud = "caller"
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if string(fields["cloud"]) != `"cloud"` || string(fields["zone"]) != `{"zone":2}` {
		t.Fatal("computed snapshot aliases original", string(raw))
	}
}

func TestCloudLocationRejectsMalformedConsumedRawWithoutPartialJSON(t *testing.T) {
	for _, tc := range []struct{ current, id, zone json.RawMessage }{
		{json.RawMessage(`broken`), nil, nil}, {nil, json.RawMessage(`{} {}`), nil}, {nil, nil, json.RawMessage{0xff}}, {nil, nil, json.RawMessage(`[] trailing`)},
	} {
		value := resource.CloudLocation{Project: resource.CloudProject{ID: tc.current}}
		raw, err := value.ForResource(tc.id, tc.zone)
		if raw != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(string(raw), err)
		}
	}
}
