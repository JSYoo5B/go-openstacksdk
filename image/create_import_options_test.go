package image

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/resource"
)

func TestCreateImportOptionsOwnReplacementAndReusableSnapshots(t *testing.T) {
	zero, disabled, duration := 0, false, time.Second
	metadata := CreateImportMetadataOpts{MinDisk: &zero, Hidden: &disabled, Tags: []string{"original"}, Properties: map[string]any{"vendor": map[string]any{"large": json.Number("9007199254740993")}}, Headers: map[string]string{"X-Note": "before"}}
	value := CreateImportOpts{Metadata: metadata, Stage: imagedata.StageOpts{Headers: map[string]string{"X-Stage": "before"}}, Import: imageimport.ImportOpts{Stores: []string{"fast"}}, Wait: &CreateImportWaitOpts{Timeout: &duration, FailureStates: []string{}}}
	option := WithCreateImportOpts(value)
	zero, disabled, duration = 7, true, time.Hour
	metadata.Tags[0] = "changed"
	metadata.Properties["vendor"].(map[string]any)["large"] = 1
	metadata.Headers["X-Note"] = "changed"
	value.Stage.Headers["X-Stage"] = "changed"
	value.Import.Stores[0] = "changed"
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			prepared, err := parseCreateImportOptions([]CreateImportOption{WithCreateImportWait(CreateImportWaitOpts{}), option})
			if err != nil || *prepared.Metadata.MinDisk != 0 || *prepared.Metadata.Hidden || prepared.Metadata.Tags[0] != "original" || prepared.Import.Stores[0] != "fast" || *prepared.Wait.Timeout != time.Second || prepared.Wait.FailureStates == nil {
				t.Errorf("prepared=%+v err=%v", prepared, err)
				return
			}
			if string(prepared.Metadata.Properties["vendor"].(json.RawMessage)) != `{"large":9007199254740993}` || prepared.Metadata.Headers["X-Note"] != "before" || prepared.Stage.Headers["X-Stage"] != "before" {
				t.Errorf("snapshot=%+v", prepared)
			}
			prepared.Metadata.Properties["vendor"] = nil
			prepared.Stage.Headers["X-Stage"] = "owned mutation"
			prepared.Import.Stores[0] = "owned mutation"
		}()
	}
	wait.Wait()
	prepared, err := parseCreateImportOptions([]CreateImportOption{option, WithCreateImportOpts(CreateImportOpts{})})
	if err != nil || prepared.Wait != nil || prepared.Metadata.MinDisk != nil || len(prepared.Import.Stores) != 0 {
		t.Fatalf("replace=%+v err=%v", prepared, err)
	}
}

func TestCreateImportDeferredPhaseCallbacksAreAppliedOnceAndCanBeCleared(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.test/v2/", Type: "image"}
	stageCalls, importCalls := 0, 0
	var capturedStage *imagedata.StageOpts
	var capturedImport *imageimport.ImportOpts
	stageOptions := []imagedata.StageOption{func(value *imagedata.StageOpts) error {
		stageCalls++
		value.Headers = map[string]string{"x-stage": "before"}
		capturedStage = value
		return nil
	}}
	importOptions := []imageimport.ImportOption{func(value *imageimport.ImportOpts) error {
		importCalls++
		value.Stores = []string{"fast"}
		capturedImport = value
		return nil
	}}
	stage, imp := WithCreateImportStageOptions(stageOptions...), WithCreateImportImportOptions(importOptions...)
	stageOptions[0], importOptions[0] = nil, nil
	prepared, err := parseCreateImportOptions([]CreateImportOption{stage, imp})
	if err != nil || stageCalls != 0 || importCalls != 0 {
		t.Fatalf("parse err=%v calls=%d/%d", err, stageCalls, importCalls)
	}
	prepared.Stage, err = imagedata.New(client).PrepareStageOptions(context.Background(), prepared.stageOptions...)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Import, err = imageimport.New(client).PrepareImportOptions(context.Background(), prepared.importOptions...)
	if err != nil {
		t.Fatal(err)
	}
	capturedStage.Headers["x-stage"], capturedImport.Stores[0] = "after", "after"
	if stageCalls != 1 || importCalls != 1 || prepared.Stage.Headers["X-Stage"] != "before" || prepared.Import.Stores[0] != "fast" {
		t.Fatalf("prepared=%+v calls=%d/%d", prepared, stageCalls, importCalls)
	}
	cleared, err := parseCreateImportOptions([]CreateImportOption{stage, imp, WithCreateImportStage(imagedata.StageOpts{}), WithCreateImportImport(imageimport.ImportOpts{})})
	if err != nil || len(cleared.stageOptions) != 0 || len(cleared.importOptions) != 0 {
		t.Fatalf("cleared=%+v err=%v", cleared, err)
	}
}

func TestCreateImportMetadataReusesUploadBuilderAndPreservesZeroFields(t *testing.T) {
	zero, disabled := 0, false
	shared := VisibilityShared
	value := CreateImportMetadataOpts{DiskFormat: "raw", Visibility: &shared, MinDisk: &zero, MinRAM: &zero, Protected: &disabled, Hidden: &disabled, Tags: []string{"linux"}, Properties: map[string]any{"vendor": json.RawMessage(`{"enabled":false,"large":9007199254740993}`)}}
	actual, err := prepareCreateImportMetadata("authoritative", value)
	if err != nil {
		t.Fatal(err)
	}
	visibility := VisibilityPrivate
	options := uploadImageOptions{base: images.CreateOpts{Name: "authoritative", DiskFormat: "qcow2", ContainerFormat: "bare", Visibility: &visibility}, properties: make(map[string]json.RawMessage)}
	for _, apply := range []UploadImageOption{WithDiskFormat("raw"), WithVisibility(VisibilityShared), WithMinDisk(0), WithMinRAM(0), WithProtected(false), WithHidden(false), WithTags("linux"), WithProperties(value.Properties)} {
		if err := apply(&options); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := (uploadMetadata{options: options}).ToImageCreateMap()
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual=%#v expected=%#v err=%v", actual, expected, err)
	}
	if actual["min_disk"] != 0 || actual["min_ram"] != 0 || actual["protected"] != false || actual["os_hidden"] != false || actual["name"] != "authoritative" {
		t.Fatalf("metadata=%#v", actual)
	}
	for _, property := range []string{"name", "id", "disk_format", "status", "size", "self"} {
		_, err := prepareCreateImportMetadata("name", CreateImportMetadataOpts{Properties: map[string]any{property: "override"}})
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("property=%s err=%v", property, err)
		}
	}
}

func TestCreateImportWaitOptionsHaveConcretePreflightPolicy(t *testing.T) {
	negative, zero, interval := -time.Second, time.Duration(0), time.Millisecond
	for _, value := range []*CreateImportWaitOpts{{Timeout: &negative}, {PollInterval: &zero}, {FailureStates: []string{" "}}} {
		_, err := prepareCreateImportWait(value)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	for _, value := range []*CreateImportWaitOpts{nil, {}, {Timeout: &zero, PollInterval: &interval, FailureStates: []string{}}} {
		options, err := prepareCreateImportWait(value)
		if err != nil {
			t.Fatal(err)
		}
		if value == nil && options != nil {
			t.Fatal("wait-disabled generated policy")
		}
	}
}

type createImportInvalidJSON struct{ cause error }

func (value createImportInvalidJSON) MarshalJSON() ([]byte, error) { return nil, value.cause }

func TestCreateImportMetadataRetainsCustomJSONFailureCause(t *testing.T) {
	cause := errors.New("vendor property encoding failed")
	metadata := CreateImportMetadataOpts{Properties: map[string]any{"vendor": createImportInvalidJSON{cause: cause}}}
	for _, test := range []struct {
		name   string
		option CreateImportOption
	}{
		{"metadata helper", WithCreateImportMetadata(metadata)},
		{"complete replacement", WithCreateImportOpts(CreateImportOpts{Metadata: metadata})},
		{"custom workflow option", func(value *CreateImportOpts) error { value.Metadata = metadata; return nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseCreateImportOptions([]CreateImportOption{test.option})
			var jsonCause *json.MarshalerError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.As(err, &jsonCause) {
				t.Fatalf("metadata failure lost encoding cause: %v", err)
			}
		})
	}
}
