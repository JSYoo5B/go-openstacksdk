package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func snapshotMutationOptionsLocation() resource.CloudLocation {
	cloud, region, project, domainID, domainName := "source-cloud", "source-region", "source-project", "domain-ID", "source-domain"
	return resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`{"zones":[true,900719925474099312345]}`), Project: resource.CloudProject{ID: json.RawMessage(`{"scope":[false,null,900719925474099312345]}`), Name: &project, DomainID: &domainID, DomainName: &domainName}}
}

func snapshotMutationOptionsDamageLocation(value *resource.CloudLocation) {
	*value.Cloud, *value.RegionName = "changed-cloud", "changed-region"
	*value.Project.Name, *value.Project.DomainID, *value.Project.DomainName = "changed-project", "changed-ID", "changed-domain"
	value.Zone[0], value.Project.ID[0] = '!', '!'
}

func snapshotMutationOptionsCreate() blockstorage.CreateVolumeSnapshotOpts {
	force, wait, timeout, interval := false, false, time.Duration(0), time.Millisecond
	name, displayName, description, displayDescription := "canonical", "display", "canonical description", "display description"
	location := snapshotMutationOptionsLocation()
	return blockstorage.CreateVolumeSnapshotOpts{Force: &force, Wait: &wait, WaitPolicy: blockstorage.SnapshotMutationWaitOpts{Timeout: &timeout, PollInterval: &interval}, Attributes: blockstorage.CreateVolumeSnapshotAttributes{Name: &name, DisplayName: &displayName, Description: &description, DisplayDescription: &displayDescription, Fields: map[string]json.RawMessage{"name": json.RawMessage(`null`), "display_name": json.RawMessage(`{"number":900719925474099312345}`), "description": json.RawMessage(`false`), "display_description": json.RawMessage(` [null,true,"\u0061"] `)}}, Location: &location}
}

func snapshotMutationOptionsDelete() blockstorage.DeleteVolumeSnapshotOpts {
	wait, timeout, interval := false, time.Duration(0), time.Millisecond
	location := snapshotMutationOptionsLocation()
	return blockstorage.DeleteVolumeSnapshotOpts{Wait: &wait, WaitPolicy: blockstorage.SnapshotMutationWaitOpts{Timeout: &timeout, PollInterval: &interval}, Location: &location}
}

func snapshotMutationOptionsDamageCreate(value *blockstorage.CreateVolumeSnapshotOpts) {
	*value.Force, *value.Wait = true, true
	*value.WaitPolicy.Timeout, *value.WaitPolicy.PollInterval = time.Hour, time.Hour
	*value.Attributes.Name, *value.Attributes.DisplayName = "changed-name", "changed-display"
	*value.Attributes.Description, *value.Attributes.DisplayDescription = "changed-description", "changed-display-description"
	for key, raw := range value.Attributes.Fields {
		if len(raw) != 0 {
			raw[0] = '!'
		}
		value.Attributes.Fields[key] = raw
	}
	value.Attributes.Fields["new-entry"] = json.RawMessage(`true`)
	snapshotMutationOptionsDamageLocation(value.Location)
}

func snapshotMutationOptionsDamageDelete(value *blockstorage.DeleteVolumeSnapshotOpts) {
	*value.Wait = true
	*value.WaitPolicy.Timeout, *value.WaitPolicy.PollInterval = time.Hour, time.Hour
	snapshotMutationOptionsDamageLocation(value.Location)
}

func snapshotMutationOptionsPrepareCallbacks(ctx context.Context, family string, nilIndex int, callbacks ...func() error) (any, error) {
	if family == "create" {
		options := make([]blockstorage.CreateVolumeSnapshotOption, len(callbacks))
		for i, callback := range callbacks {
			callback := callback
			if i != nilIndex {
				options[i] = func(next *blockstorage.CreateVolumeSnapshotOpts) error {
					*next = snapshotMutationOptionsCreate()
					return callback()
				}
			}
		}
		return blockstorage.PrepareCreateVolumeSnapshotOptions(ctx, options...)
	}
	options := make([]blockstorage.DeleteVolumeSnapshotOption, len(callbacks))
	for i, callback := range callbacks {
		callback := callback
		if i != nilIndex {
			options[i] = func(next *blockstorage.DeleteVolumeSnapshotOpts) error {
				*next = snapshotMutationOptionsDelete()
				return callback()
			}
		}
	}
	return blockstorage.PrepareDeleteVolumeSnapshotOptions(ctx, options...)
}

func snapshotMutationOptionsZero(family string) any {
	if family == "create" {
		return blockstorage.CreateVolumeSnapshotOpts{}
	}
	return blockstorage.DeleteVolumeSnapshotOpts{}
}

func TestSnapshotMutationOptionsPreserveOmissionAndExplicitFalseZero(t *testing.T) {
	create, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background())
	if err != nil || !reflect.DeepEqual(create, blockstorage.CreateVolumeSnapshotOpts{}) {
		t.Fatalf("omitted create policy=%+v error=%v", create, err)
	}
	deleted, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background())
	if err != nil || !reflect.DeepEqual(deleted, blockstorage.DeleteVolumeSnapshotOpts{}) {
		t.Fatalf("omitted delete policy=%+v error=%v", deleted, err)
	}
	zero := time.Duration(0)
	create, err = blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), blockstorage.WithCreateVolumeSnapshotForce(false), blockstorage.WithCreateVolumeSnapshotWait(false), blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &zero, PollInterval: &zero}), blockstorage.WithCreateVolumeSnapshotName(""), blockstorage.WithCreateVolumeSnapshotDisplayName(""), blockstorage.WithCreateVolumeSnapshotDescription(""), blockstorage.WithCreateVolumeSnapshotDisplayDescription(""))
	if err != nil || create.Force == nil || *create.Force || create.Wait == nil || *create.Wait || create.WaitPolicy.Timeout == nil || *create.WaitPolicy.Timeout != 0 || create.WaitPolicy.PollInterval == nil || *create.WaitPolicy.PollInterval != 0 {
		t.Fatalf("explicit create false/zero policy=%+v error=%v", create, err)
	}
	for _, field := range []*string{create.Attributes.Name, create.Attributes.DisplayName, create.Attributes.Description, create.Attributes.DisplayDescription} {
		if field == nil || *field != "" {
			t.Fatal("explicit empty alias pointer was discarded", create.Attributes)
		}
	}
	deleted, err = blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), blockstorage.WithDeleteVolumeSnapshotWait(false), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &zero, PollInterval: &zero}))
	if err != nil || deleted.Wait == nil || *deleted.Wait || deleted.WaitPolicy.Timeout == nil || *deleted.WaitPolicy.Timeout != 0 || deleted.WaitPolicy.PollInterval == nil || *deleted.WaitPolicy.PollInterval != 0 {
		t.Fatalf("explicit delete false/zero policy=%+v error=%v", deleted, err)
	}
	negative := -time.Second
	create, err = blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &negative, PollInterval: &negative}))
	if err != nil || create.Wait != nil || create.WaitPolicy.Timeout == nil || create.WaitPolicy.PollInterval == nil || *create.WaitPolicy.Timeout != negative || *create.WaitPolicy.PollInterval != negative {
		t.Fatal("pure preparation prematurely evaluated wait policy", create, err)
	}
}

func TestSnapshotMutationOptionsCompleteFactoriesOwnEverySubtreeAcrossReuses(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		input, expected := snapshotMutationOptionsCreate(), snapshotMutationOptionsCreate()
		factory := blockstorage.WithCreateVolumeSnapshotOptions(input)
		snapshotMutationOptionsDamageCreate(&input)
		first, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
		if err != nil || !reflect.DeepEqual(first, expected) {
			t.Fatalf("caller input changed factory=%+v error=%v", first, err)
		}
		snapshotMutationOptionsDamageCreate(&first)
		second, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
		if err != nil || !reflect.DeepEqual(second, expected) || second.Force == first.Force || second.WaitPolicy.Timeout == first.WaitPolicy.Timeout || second.Attributes.Name == first.Attributes.Name || second.Location == first.Location {
			t.Fatalf("prior result changed reusable factory=%+v error=%v", second, err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		input, expected := snapshotMutationOptionsDelete(), snapshotMutationOptionsDelete()
		factory := blockstorage.WithDeleteVolumeSnapshotOptions(input)
		snapshotMutationOptionsDamageDelete(&input)
		first, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), factory)
		if err != nil || !reflect.DeepEqual(first, expected) {
			t.Fatalf("caller input changed factory=%+v error=%v", first, err)
		}
		snapshotMutationOptionsDamageDelete(&first)
		second, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), factory)
		if err != nil || !reflect.DeepEqual(second, expected) || second.Wait == first.Wait || second.WaitPolicy.Timeout == first.WaitPolicy.Timeout || second.Location == first.Location {
			t.Fatalf("prior result changed reusable factory=%+v error=%v", second, err)
		}
	})
}

func TestSnapshotMutationOptionsFactoriesSupportConcurrentIndependentPreparedValues(t *testing.T) {
	createExpected, deleteExpected := snapshotMutationOptionsCreate(), snapshotMutationOptionsDelete()
	createFactory, deleteFactory := blockstorage.WithCreateVolumeSnapshotOptions(createExpected), blockstorage.WithDeleteVolumeSnapshotOptions(deleteExpected)
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			create, createErr := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), createFactory)
			deleted, deleteErr := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), deleteFactory)
			if createErr != nil || deleteErr != nil || !reflect.DeepEqual(create, createExpected) || !reflect.DeepEqual(deleted, deleteExpected) {
				t.Errorf("independent prepared policy create=%+v delete=%+v errors=%v/%v", create, deleted, createErr, deleteErr)
				return
			}
			snapshotMutationOptionsDamageCreate(&create)
			snapshotMutationOptionsDamageDelete(&deleted)
		}()
	}
	workers.Wait()
	create, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), createFactory)
	if err != nil || !reflect.DeepEqual(create, createExpected) {
		t.Fatal("concurrent result rewrote reusable create factory", create, err)
	}
	deleted, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), deleteFactory)
	if err != nil || !reflect.DeepEqual(deleted, deleteExpected) {
		t.Fatal("concurrent result rewrote reusable delete factory", deleted, err)
	}
}

func TestSnapshotMutationOptionsAttributeFactoriesRetainAllAliasesWithoutNormalizing(t *testing.T) {
	original, expected := snapshotMutationOptionsCreate(), snapshotMutationOptionsCreate()
	factory := blockstorage.WithCreateVolumeSnapshotAttributes(original.Attributes)
	snapshotMutationOptionsDamageCreate(&original)
	first, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
	if err != nil || !reflect.DeepEqual(first.Attributes, expected.Attributes) || first.Force != nil || first.Wait != nil || first.Location != nil {
		t.Fatal("attribute factory changed alias values or unrelated policy", first, err)
	}
	// A typed alias and its raw same-key value coexist until runtime payload
	// selection. Preparation must preserve null, false and precision literally.
	if *first.Attributes.Name != "canonical" || string(first.Attributes.Fields["name"]) != `null` || string(first.Attributes.Fields["description"]) != `false` || string(first.Attributes.Fields["display_name"]) != `{"number":900719925474099312345}` {
		t.Fatal("pure preparation performed alias/body selection", first.Attributes)
	}
	*first.Attributes.DisplayName = "caller changed alias"
	first.Attributes.Fields["display_description"][0] = '!'
	second, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
	if err != nil || !reflect.DeepEqual(second.Attributes, expected.Attributes) {
		t.Fatal("attribute factory reused result storage", second, err)
	}
}

func TestSnapshotMutationOptionsRawFactoriesPreserveNilEmptyMapsAndExactByteShapes(t *testing.T) {
	for _, family := range []string{"complete", "attributes", "fields"} {
		for _, fields := range []map[string]json.RawMessage{nil, {}, {"name": nil}, {"name": json.RawMessage{}}, {"name": json.RawMessage(`null`), "display_name": json.RawMessage(`false`), "description": json.RawMessage(`0`), "display_description": json.RawMessage(`[]`)}, {"name": json.RawMessage(` {"ordered": [true,null], "large":123456789012345678901234567890.001,"escaped":"\u0061"} `)}} {
			t.Run(family, func(t *testing.T) {
				var expected map[string]json.RawMessage
				if fields != nil {
					expected = make(map[string]json.RawMessage, len(fields))
					for key, raw := range fields {
						expected[key] = bytes.Clone(raw)
					}
				}
				var factory blockstorage.CreateVolumeSnapshotOption
				switch family {
				case "complete":
					factory = blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{Attributes: blockstorage.CreateVolumeSnapshotAttributes{Fields: fields}})
				case "attributes":
					factory = blockstorage.WithCreateVolumeSnapshotAttributes(blockstorage.CreateVolumeSnapshotAttributes{Fields: fields})
				case "fields":
					factory = blockstorage.WithCreateVolumeSnapshotFields(fields)
				}
				for key, raw := range fields {
					if len(raw) != 0 {
						raw[0] = '!'
					}
					fields[key] = json.RawMessage(`"caller"`)
				}
				if fields != nil {
					fields["later"] = json.RawMessage(`true`)
				}
				first, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
				if err != nil || !reflect.DeepEqual(first.Attributes.Fields, expected) {
					t.Fatal("raw factory collapsed omission/empty/raw bytes", first.Attributes.Fields, expected, err)
				}
				if first.Attributes.Fields != nil {
					for key, raw := range first.Attributes.Fields {
						if len(raw) != 0 {
							raw[0] = '?'
						}
						first.Attributes.Fields[key] = raw
					}
					first.Attributes.Fields["first-result"] = json.RawMessage(`true`)
				}
				second, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory)
				if err != nil || !reflect.DeepEqual(second.Attributes.Fields, expected) {
					t.Fatal("raw bytes/map leaked into reusable factory", second.Attributes.Fields, expected, err)
				}
			})
		}
	}
}

func TestSnapshotMutationOptionsIndividualFactoriesOverrideOnlySelectedSubtrees(t *testing.T) {
	timeout, interval := time.Millisecond, 2*time.Millisecond
	location := snapshotMutationOptionsLocation()
	ownedLocation := location.Clone()
	fields := map[string]json.RawMessage{"name": json.RawMessage(`null`), "display_name": json.RawMessage(`"raw display"`)}
	fieldsFactory := blockstorage.WithCreateVolumeSnapshotFields(fields)
	createWaitFactory := blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &timeout, PollInterval: &interval})
	deleteWaitFactory := blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{Timeout: &timeout, PollInterval: &interval})
	createLocationFactory, deleteLocationFactory := blockstorage.WithCreateVolumeSnapshotLocation(location), blockstorage.WithDeleteVolumeSnapshotLocation(location)
	fields["display_name"][0] = '!'
	timeout, interval = time.Hour, time.Hour
	snapshotMutationOptionsDamageLocation(&location)
	options := []blockstorage.CreateVolumeSnapshotOption{blockstorage.WithCreateVolumeSnapshotOptions(snapshotMutationOptionsCreate()), blockstorage.WithCreateVolumeSnapshotForce(true), blockstorage.WithCreateVolumeSnapshotForce(false), blockstorage.WithCreateVolumeSnapshotWait(true), createWaitFactory, fieldsFactory, blockstorage.WithCreateVolumeSnapshotName("later name"), blockstorage.WithCreateVolumeSnapshotDisplayName("later display"), blockstorage.WithCreateVolumeSnapshotDescription("later description"), blockstorage.WithCreateVolumeSnapshotDisplayDescription("later display description"), createLocationFactory}
	checkCreate := func(value blockstorage.CreateVolumeSnapshotOpts, err error) {
		t.Helper()
		if err != nil || value.Force == nil || value.Wait == nil || value.WaitPolicy.Timeout == nil || value.WaitPolicy.PollInterval == nil || value.Attributes.Name == nil || value.Attributes.DisplayName == nil || value.Attributes.Description == nil || value.Attributes.DisplayDescription == nil || value.Location == nil || *value.Force || !*value.Wait || *value.WaitPolicy.Timeout != time.Millisecond || *value.WaitPolicy.PollInterval != 2*time.Millisecond || *value.Attributes.Name != "later name" || *value.Attributes.DisplayName != "later display" || *value.Attributes.Description != "later description" || *value.Attributes.DisplayDescription != "later display description" || string(value.Attributes.Fields["name"]) != `null` || string(value.Attributes.Fields["display_name"]) != `"raw display"` || len(value.Attributes.Fields) != 2 || !reflect.DeepEqual(*value.Location, ownedLocation) {
			t.Fatal("individual options changed another subtree or borrowed input", value, err)
		}
	}
	first, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), options...)
	checkCreate(first, err)
	snapshotMutationOptionsDamageCreate(&first)
	second, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), options...)
	checkCreate(second, err)
	deleted, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), blockstorage.WithDeleteVolumeSnapshotOptions(snapshotMutationOptionsDelete()), blockstorage.WithDeleteVolumeSnapshotWait(true), blockstorage.WithDeleteVolumeSnapshotWait(false), deleteWaitFactory, deleteLocationFactory)
	if err != nil || deleted.Wait == nil || deleted.WaitPolicy.Timeout == nil || deleted.WaitPolicy.PollInterval == nil || deleted.Location == nil || *deleted.Wait || *deleted.WaitPolicy.Timeout != time.Millisecond || *deleted.WaitPolicy.PollInterval != 2*time.Millisecond || !reflect.DeepEqual(*deleted.Location, ownedLocation) {
		t.Fatal("delete individual factories changed or borrowed policy", deleted, err)
	}
	snapshotMutationOptionsDamageDelete(&deleted)
	deleted, err = blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), deleteWaitFactory, deleteLocationFactory)
	if err != nil || deleted.Wait != nil || deleted.WaitPolicy.Timeout == nil || deleted.WaitPolicy.PollInterval == nil || deleted.Location == nil || *deleted.WaitPolicy.Timeout != time.Millisecond || *deleted.WaitPolicy.PollInterval != 2*time.Millisecond || !reflect.DeepEqual(*deleted.Location, ownedLocation) {
		t.Fatal("reusable individual delete policy/location changed", deleted, err)
	}
}

func TestSnapshotMutationOptionsCompleteReplacementClearsPriorPolicyAndSubtrees(t *testing.T) {
	create, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), blockstorage.WithCreateVolumeSnapshotOptions(snapshotMutationOptionsCreate()), blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{}))
	if err != nil || !reflect.DeepEqual(create, blockstorage.CreateVolumeSnapshotOpts{}) {
		t.Fatal("whole create options merged earlier policy", create, err)
	}
	deleted, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), blockstorage.WithDeleteVolumeSnapshotOptions(snapshotMutationOptionsDelete()), blockstorage.WithDeleteVolumeSnapshotOptions(blockstorage.DeleteVolumeSnapshotOpts{}))
	if err != nil || !reflect.DeepEqual(deleted, blockstorage.DeleteVolumeSnapshotOpts{}) {
		t.Fatal("whole delete options merged earlier policy", deleted, err)
	}
	create, err = blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), blockstorage.WithCreateVolumeSnapshotOptions(snapshotMutationOptionsCreate()), blockstorage.WithCreateVolumeSnapshotAttributes(blockstorage.CreateVolumeSnapshotAttributes{}), blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{}))
	if err != nil || !reflect.DeepEqual(create.Attributes, blockstorage.CreateVolumeSnapshotAttributes{}) || !reflect.DeepEqual(create.WaitPolicy, blockstorage.SnapshotMutationWaitOpts{}) || create.Force == nil || *create.Force || create.Wait == nil || *create.Wait || create.Location == nil {
		t.Fatal("subtree replacement retained attributes or erased unrelated policy", create, err)
	}
	deleted, err = blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), blockstorage.WithDeleteVolumeSnapshotOptions(snapshotMutationOptionsDelete()), blockstorage.WithDeleteVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{}))
	if err != nil || !reflect.DeepEqual(deleted.WaitPolicy, blockstorage.SnapshotMutationWaitOpts{}) || deleted.Wait == nil || *deleted.Wait || deleted.Location == nil {
		t.Fatal("delete wait subtree replacement changed other fields", deleted, err)
	}
}

func TestSnapshotMutationOptionsCaptureOriginalSliceAndIsolateRetainedCallbacks(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		original, expected := snapshotMutationOptionsCreate(), snapshotMutationOptionsCreate()
		var retained, last *blockstorage.CreateVolumeSnapshotOpts
		var order []int
		options := make([]blockstorage.CreateVolumeSnapshotOption, 2)
		options[0] = func(next *blockstorage.CreateVolumeSnapshotOpts) error {
			order = append(order, 1)
			*next = original
			retained = next
			options[1] = func(*blockstorage.CreateVolumeSnapshotOpts) error { t.Fatal("replacement original ran"); return nil }
			return nil
		}
		options[1] = func(next *blockstorage.CreateVolumeSnapshotOpts) error {
			order = append(order, 2)
			snapshotMutationOptionsDamageCreate(retained)
			if !reflect.DeepEqual(*next, expected) {
				t.Fatal("retained callback changed next owned policy", next)
			}
			last = next
			return nil
		}
		value, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), options...)
		if err != nil || !reflect.DeepEqual(order, []int{1, 2}) || !reflect.DeepEqual(value, expected) {
			t.Fatal("original callbacks/order", value, err, order)
		}
		snapshotMutationOptionsDamageCreate(last)
		if !reflect.DeepEqual(value, expected) {
			t.Fatal("last callback handle changed result", value)
		}
	})
	t.Run("delete", func(t *testing.T) {
		original, expected := snapshotMutationOptionsDelete(), snapshotMutationOptionsDelete()
		var retained, last *blockstorage.DeleteVolumeSnapshotOpts
		var order []int
		options := make([]blockstorage.DeleteVolumeSnapshotOption, 2)
		options[0] = func(next *blockstorage.DeleteVolumeSnapshotOpts) error {
			order = append(order, 1)
			*next = original
			retained = next
			options[1] = func(*blockstorage.DeleteVolumeSnapshotOpts) error { t.Fatal("replacement original ran"); return nil }
			return nil
		}
		options[1] = func(next *blockstorage.DeleteVolumeSnapshotOpts) error {
			order = append(order, 2)
			snapshotMutationOptionsDamageDelete(retained)
			if !reflect.DeepEqual(*next, expected) {
				t.Fatal("retained callback changed next owned policy", next)
			}
			last = next
			return nil
		}
		value, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), options...)
		if err != nil || !reflect.DeepEqual(order, []int{1, 2}) || !reflect.DeepEqual(value, expected) {
			t.Fatal("original callbacks/order", value, err, order)
		}
		snapshotMutationOptionsDamageDelete(last)
		if !reflect.DeepEqual(value, expected) {
			t.Fatal("last callback handle changed result", value)
		}
	})
}

func TestSnapshotMutationOptionsRejectMissingCanceledOrExpiredContextBeforeCallbacks(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		for _, state := range []string{"nil", "canceled", "deadline", "deadline-cause"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				cause := errors.New("snapshot mutation caller cause")
				var ctx context.Context
				if state == "canceled" {
					canceled, cancel := context.WithCancelCause(context.Background())
					cancel(cause)
					ctx = canceled
				}
				if state == "deadline" {
					expired, cancel := context.WithTimeout(context.Background(), 0)
					defer cancel()
					ctx = expired
				}
				if state == "deadline-cause" {
					expired, cancel := context.WithDeadlineCause(context.Background(), time.Unix(0, 0), cause)
					defer cancel()
					ctx = expired
				}
				calls := 0
				value, err := snapshotMutationOptionsPrepareCallbacks(ctx, family, -1, func() error { calls++; return nil })
				if err == nil || calls != 0 || !reflect.DeepEqual(value, snapshotMutationOptionsZero(family)) {
					t.Fatal("preflight executed original or exposed policy", value, err, calls)
				}
				if state == "nil" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if state == "canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal("lost cancellation cause", err)
				}
				if state == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal("lost deadline", err)
				}
				if state == "deadline-cause" && (!errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, cause)) {
					t.Fatal("lost deadline cause", err)
				}
			})
		}
	}
}

func TestSnapshotMutationOptionsStopOnNilErrorOrCancellationWithoutPartialPolicy(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family+"/nil", func(t *testing.T) {
			calls := 0
			value, err := snapshotMutationOptionsPrepareCallbacks(context.Background(), family, 1, func() error { calls++; return nil }, func() error { t.Fatal("nil option invoked"); return nil }, func() error { t.Fatal("callback after nil ran"); return nil })
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || !reflect.DeepEqual(value, snapshotMutationOptionsZero(family)) {
				t.Fatal(value, err, calls)
			}
		})
		for _, state := range []string{"error", "cancel", "both"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				callbackError, cause := errors.New("original option error"), errors.New("option cancel cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls := 0
				value, err := snapshotMutationOptionsPrepareCallbacks(ctx, family, -1, func() error { calls++; return nil }, func() error {
					calls++
					if state != "error" {
						cancel(cause)
					}
					if state != "cancel" {
						return callbackError
					}
					return nil
				}, func() error { t.Fatal("callback after terminal option ran"); return nil })
				if err == nil || calls != 2 || !reflect.DeepEqual(value, snapshotMutationOptionsZero(family)) {
					t.Fatal("partial policy exposed after failure", value, err, calls)
				}
				if state != "cancel" && !errors.Is(err, callbackError) {
					t.Fatal("lost original error", err)
				}
				if state != "error" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal("lost cancel and cause", err)
				}
			})
		}
	}
}

func TestSnapshotMutationOptionsPreparationIsPureAndDefersReachedValueValidation(t *testing.T) {
	negative := -time.Second
	badName := string([]byte{0xff})
	badLocation := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not-json`)}}
	fields := map[string]json.RawMessage{"unknown-route-key": json.RawMessage(`{unfinished`), "name": nil, "description": json.RawMessage{}}
	calls := 0
	factory := blockstorage.WithCreateVolumeSnapshotOptions(blockstorage.CreateVolumeSnapshotOpts{Attributes: blockstorage.CreateVolumeSnapshotAttributes{Name: &badName, Fields: fields}, WaitPolicy: blockstorage.SnapshotMutationWaitOpts{Timeout: &negative, PollInterval: &negative}, Location: &badLocation})
	// Pure preparation receives no service/client or route and executes only
	// supplied callbacks. It owns even unusable reached values; workflow
	// validation decides which of them must be consumed before HTTP/sleep.
	value, err := blockstorage.PrepareCreateVolumeSnapshotOptions(context.Background(), factory, func(*blockstorage.CreateVolumeSnapshotOpts) error { calls++; return nil })
	if err != nil || calls != 1 || value.Attributes.Name == nil || value.WaitPolicy.Timeout == nil || value.WaitPolicy.PollInterval == nil || value.Location == nil || *value.Attributes.Name != badName || !reflect.DeepEqual(value.Attributes.Fields, fields) || *value.WaitPolicy.Timeout != negative || *value.WaitPolicy.PollInterval != negative || !reflect.DeepEqual(*value.Location, badLocation) {
		t.Fatal("pure preparer performed workflow validation", value, err, calls)
	}
	if value.Force != nil || value.Wait != nil {
		t.Fatal("pure preparer invented operational defaults", value)
	}
	deleted, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(context.Background(), blockstorage.WithDeleteVolumeSnapshotOptions(blockstorage.DeleteVolumeSnapshotOpts{WaitPolicy: blockstorage.SnapshotMutationWaitOpts{Timeout: &negative, PollInterval: &negative}, Location: &badLocation}))
	if err != nil || deleted.Wait != nil || deleted.WaitPolicy.Timeout == nil || deleted.WaitPolicy.PollInterval == nil || deleted.Location == nil || *deleted.WaitPolicy.Timeout != negative || *deleted.WaitPolicy.PollInterval != negative || !reflect.DeepEqual(*deleted.Location, badLocation) {
		t.Fatal("pure delete preparation consumed unused policy", deleted, err)
	}
}
