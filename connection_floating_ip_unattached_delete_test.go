package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/resource"
)

func unattachedDeletes(events []string) []string {
	var result []string
	for _, event := range events {
		if strings.HasPrefix(event, "DELETE ") {
			result = append(result, event)
		}
	}
	return result
}

func TestFloatingIPUnattachedEagerPagedInventoryAndConfiguredDeleteOrder(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	inventoryDone := false
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			if !inventoryDone {
				t.Error("DELETE started before inventory completion")
			}
			if r.Header.Get("If-Match") != "" {
				t.Error("unexpected revision delete", r.Header)
			}
			return 204, ""
		}
		if inventoryDone {
			return 200, `{"floatingips":[]}`
		}
		if r.URL.Query().Get("marker") == "second" {
			inventoryDone = true
			return 200, `{"floatingips":[{"id":"b","port_id":null}]}`
		}
		return 200, `{"floatingips":[{"id":"a","port_id":""},{"id":"unused/unsafe","port_id":"attached"}],"floatingips_links":[{"rel":"next","href":"?marker=second"}]}`
	}
	result, err := conn.DeleteUnattachedFloatingIPs(context.Background())
	want := []string{
		"GET /delete-network/v2.0/floatingips", "GET /delete-network/v2.0/floatingips",
		"DELETE /delete-network/v2.0/floatingips/a", "GET /delete-network/v2.0/floatingips",
		"DELETE /delete-network/v2.0/floatingips/b", "GET /delete-network/v2.0/floatingips",
	}
	if err != nil || result == nil || !result.Eligible || !result.AllDeleted || result.Count != 2 || len(result.Items) != 2 || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network"}) {
		t.Fatal(result, err, state)
	}
	if result.Inventory == nil || result.Inventory.Backend != compute.FloatingIPNeutron || len(result.Inventory.Pages) != 2 || len(result.Inventory.FloatingIPs) != 3 {
		t.Fatal(result)
	}
	for index, id := range []string{"a", "b"} {
		item := result.Items[index]
		if item.ID != id || item.FloatingIP == nil || item.Deletion == nil || !item.Deletion.Deleted || !item.Deletion.Absent || item.Deletion.Down || len(item.Deletion.Attempts) != 1 {
			t.Fatal(item)
		}
		attempt := item.Deletion.Attempts[0]
		if !attempt.Accepted || !attempt.Verified || attempt.Response.StatusCode != 204 || attempt.Verification == nil || len(attempt.Verification.Pages) != 1 {
			t.Fatal(attempt)
		}
	}
}

func TestFloatingIPUnattachedSourceAndServiceGateDistinguishesSuccessfulZero(t *testing.T) {
	for _, mode := range []string{"nova", "none", "absent", "mixed-absence", "empty", "attached"} {
		t.Run(mode, func(t *testing.T) {
			source := compute.FloatingIPNeutron
			if mode == "nova" {
				source = compute.FloatingIPNova
			}
			if mode == "none" {
				source = compute.FloatingIPNone
			}
			_, conn, state := floatingDeleteFixture(t, source)
			cause := errors.New("catalog terminal cause")
			if mode == "absent" {
				state.catalogError = &gophercloud.ErrEndpointNotFound{}
			}
			if mode == "mixed-absence" {
				state.catalogError = errors.Join(&gophercloud.ErrEndpointNotFound{}, cause)
			}
			state.reply = func(*http.Request) (int, string) {
				if mode != "empty" && mode != "attached" {
					t.Error("gate made an HTTP request")
				}
				if mode == "attached" {
					return 200, `{"floatingips":[{"id":"unsafe/unused","port_id":" "}]}`
				}
				return 200, `{"floatingips":[]}`
			}
			result, err := conn.DeleteUnattachedFloatingIPs(context.Background())
			if mode == "mixed-absence" {
				if !errors.Is(err, cause) || len(state.events) != 0 {
					t.Fatal(result, err, state)
				}
				if result != nil && (result.AllDeleted || result.Count != 0 || len(result.Items) != 0 || result.Inventory != nil) {
					t.Fatal(result)
				}
				return
			}
			if err != nil || result == nil || result.Count != 0 || !result.AllDeleted || len(result.Items) != 0 {
				t.Fatal(result, err, state)
			}
			eligible := mode == "empty" || mode == "attached"
			if result.Eligible != eligible || (result.Inventory != nil) != eligible {
				t.Fatal(result)
			}
			if !eligible && len(state.events) != 0 {
				t.Fatal(result, state)
			}
			if eligible && !reflect.DeepEqual(state.events, []string{"GET /delete-network/v2.0/floatingips"}) {
				t.Fatal(state)
			}
			if (mode == "nova" || mode == "none") && len(state.locators) != 0 {
				t.Fatal(state.locators)
			}
			if mode != "nova" && mode != "none" && !reflect.DeepEqual(state.locators, []string{"network"}) {
				t.Fatal(state.locators)
			}
		})
	}
}

func TestFloatingIPUnattachedFalseContinuesAndErrorStopsWithPartialHistory(t *testing.T) {
	for _, mode := range []string{"first-false", "delete-error", "verification-error", "second-missing"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			inventoryReturned := false
			lastID := ""
			deletes := map[string]int{}
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" && !inventoryReturned {
					inventoryReturned = true
					return 200, `{"floatingips":[{"id":"a"},{"id":"b"},{"id":"c"}]}`
				}
				if r.Method == "GET" {
					if lastID == "b" && mode == "verification-error" {
						return 403, "verification denied"
					}
					if lastID == "b" && mode == "second-missing" {
						return 200, `{"floatingips":[{"id":"b","status":"ACTIVE"}]}`
					}
					return 200, `{"floatingips":[]}`
				}
				lastID = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
				deletes[lastID]++
				if lastID == "a" && mode == "first-false" {
					return 404, "a missing"
				}
				if lastID == "b" && mode == "delete-error" {
					return 403, "b denied"
				}
				if lastID == "b" && mode == "second-missing" && deletes[lastID] == 2 {
					return 404, "b second missing"
				}
				return 202, "passive accepted"
			}
			result, err := conn.DeleteUnattachedFloatingIPs(context.Background())
			if result == nil || !result.Eligible || result.AllDeleted || len(result.Inventory.FloatingIPs) != 3 {
				t.Fatal(result, err)
			}
			if mode == "delete-error" || mode == "verification-error" {
				if !gophercloud.ResponseCodeIs(err, 403) || result.Count != 1 || len(result.Items) != 2 || deletes["c"] != 0 {
					t.Fatal(result, err, state)
				}
			} else if err != nil || result.Count != 2 || len(result.Items) != 3 || deletes["c"] != 1 {
				t.Fatal(result, err, state)
			}
			if mode == "first-false" {
				item := result.Items[0]
				if item.Deletion.Deleted || len(item.Deletion.Attempts) != 1 || item.Deletion.Attempts[0].NotFound == nil || item.Deletion.Attempts[0].Accepted {
					t.Fatal(item)
				}
			}
			if mode == "verification-error" {
				item := result.Items[1]
				if item.Deletion.Deleted || !item.Deletion.Attempts[0].Accepted || item.Deletion.LastVerification == nil || item.Deletion.Failure.StatusCode != 403 {
					t.Fatal(item)
				}
			}
			if mode == "second-missing" {
				item := result.Items[1]
				if item.Deletion.Deleted || len(item.Deletion.Attempts) != 2 || !item.Deletion.Attempts[0].Accepted || item.Deletion.Attempts[1].Accepted || item.Deletion.Attempts[1].NotFound == nil || item.Deletion.LastVerification.FloatingIP == nil {
					t.Fatal(item)
				}
			}
		})
	}
}

func TestFloatingIPUnattachedPreparedRetryPolicyIsAppliedOnceAcrossItems(t *testing.T) {
	for _, retry := range []int{1, 0, -2, 3} {
		t.Run(fmt.Sprint(retry), func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			inventoryReturned := false
			lastID := ""
			attempts := map[string]int{}
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" && !inventoryReturned {
					inventoryReturned = true
					return 200, `{"floatingips":[{"id":"a"},{"id":"b"}]}`
				}
				if r.Method == "DELETE" {
					lastID = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
					attempts[lastID]++
					return 202, "not-json"
				}
				if retry == 0 {
					t.Error("retry zero performed verification")
				}
				status := "DOWN"
				if lastID == "a" && attempts["a"] < max(0, retry)+1 {
					status = "ACTIVE"
				}
				return 200, fmt.Sprintf(`{"floatingips":[{"id":%q,"status":%q}]}`, lastID, status)
			}
			calls := 0
			result, err := conn.DeleteUnattachedFloatingIPs(context.Background(), func(o *compute.FloatingIPDeleteOpts) error {
				calls++
				if o.Retries != 1 {
					t.Error("default retries", o.Retries)
				}
				o.Retries = retry
				return nil
			})
			wantA := max(0, retry) + 1
			if retry == 0 {
				wantA = 1
			}
			if err != nil || calls != 1 || result == nil || !result.AllDeleted || result.Count != 2 || attempts["a"] != wantA || attempts["b"] != 1 || len(result.Items[0].Deletion.Attempts) != wantA {
				t.Fatal(result, err, calls, state)
			}
			if retry == 0 {
				if len(state.events) != 3 || result.Items[0].Deletion.LastVerification != nil || result.Items[1].Deletion.LastVerification != nil {
					t.Fatal(result, state)
				}
			} else if len(state.events) != 1+2*(wantA+1) || !result.Items[0].Deletion.Down || !result.Items[1].Deletion.Down {
				t.Fatal(result, state)
			}
		})
	}
}

func TestFloatingIPUnattachedResourcePortTruthinessAndStrictFallback(t *testing.T) {
	t.Run("neutron truthiness", func(t *testing.T) {
		_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
		values := []string{"missing", "null", `""`, "false", "0", "[]", "{}", `" "`, `"0"`, "1", `[false]`, `{"x":null}`}
		var rows []string
		for index, value := range values {
			id := fmt.Sprintf("ip%d", index)
			if index >= 7 {
				id = fmt.Sprintf("unused/unsafe%d", index)
			}
			row := fmt.Sprintf(`{"id":%q`, id)
			if value != "missing" {
				row += `,"port_id":` + value
			}
			rows = append(rows, row+"}")
		}
		body := `{"floatingips":[` + strings.Join(rows, ",") + `]}`
		state.reply = func(r *http.Request) (int, string) {
			if r.Method == "GET" {
				return 200, body
			}
			return 204, ""
		}
		result, err := conn.DeleteUnattachedFloatingIPs(context.Background(), compute.WithFloatingIPDeleteRetries(0), compute.WithFloatingIPDeleteStrict(true))
		if err != nil || result == nil || result.Count != 7 || !result.AllDeleted || len(result.Items) != 7 || len(state.events) != 8 {
			t.Fatal(result, err, state)
		}
		for index, item := range result.Items {
			if item.ID != fmt.Sprintf("ip%d", index) || !item.Deletion.Deleted {
				t.Fatal(item)
			}
		}
		queryRaw(t, result.Inventory.FloatingIPs[0].Resource, "port_id", "null")
		if _, present := result.Inventory.FloatingIPs[0].Wire.Body["port_id"]; present {
			t.Fatal("invented missing wire port")
		}
	})
	for _, strict := range []bool{false, true} {
		t.Run(fmt.Sprint("fallback strict", strict), func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					if r.URL.Path != "/delete-network/v2.0/floatingips/7" {
						t.Error("used inventory backend for DELETE", r.URL.Path)
					}
					return 204, ""
				}
				if strings.Contains(r.URL.Path, "compute") {
					return 200, `{"floating_ips":[{"id":7,"ip":"known","instance_id":"attached-in-nova"}]}`
				}
				return 404, "no neutron floating IP inventory"
			}
			result, err := conn.DeleteUnattachedFloatingIPs(context.Background(), compute.WithFloatingIPDeleteRetries(0), compute.WithFloatingIPDeleteStrict(strict))
			if result == nil || !result.Eligible || result.Inventory == nil || result.Inventory.Backend != compute.FloatingIPNova || result.Inventory.FallbackError == nil {
				t.Fatal(result, err)
			}
			row := result.Inventory.FloatingIPs[0]
			if row.NormalizationSource != compute.FloatingIPNeutron {
				t.Fatal(row)
			}
			if strict {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Count != 0 || result.AllDeleted || len(unattachedDeletes(state.events)) != 0 {
					t.Fatal(result, err, state)
				}
				if _, present := row.Resource.Body["port_id"]; present {
					t.Fatal(row)
				}
			} else if err != nil || result.Count != 1 || !result.AllDeleted || result.Items[0].Deletion.Backend != compute.FloatingIPNeutron || len(state.events) != 3 {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestFloatingIPUnattachedInventoryErrorsNeverDeleteEligiblePrefix(t *testing.T) {
	for _, mode := range []string{"early403", "descriptor", "late403", "malformed", "cycle", "close", "source", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			cached, cacheErr := conn.Network(context.Background())
			if cacheErr != nil {
				t.Fatal(cacheErr)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("late inventory boundary")
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "DELETE" {
					t.Error("deleted incomplete inventory")
					return 204, ""
				}
				if mode == "early403" {
					return 403, "inventory denied"
				}
				if r.URL.Query().Get("marker") == "" {
					row := `{"id":"a"}`
					if mode == "descriptor" {
						row = `{"id":"a","revision_number":"²"}`
					}
					return 200, `{"floatingips":[` + row + `],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
				if mode == "late403" {
					return 403, "late denied"
				}
				if mode == "malformed" {
					return 200, `{"floatingips":{}}`
				}
				if mode == "cycle" {
					return 200, `{"floatingips":[{"id":"b"}],"floatingips_links":[{"rel":"next","href":"?marker=next"}]}`
				}
				return 200, `{"floatingips":[{"id":"b"}]}`
			}
			state.afterClose = func(r *http.Request) error {
				if r.URL.Query().Get("marker") == "" {
					return nil
				}
				if mode == "close" {
					return cause
				}
				if mode == "source" {
					cached.RawClient().Endpoint += "changed/"
				}
				if mode == "cancel" {
					cancel(cause)
				}
				return nil
			}
			result, err := conn.DeleteUnattachedFloatingIPs(ctx)
			if err == nil || result == nil || !result.Eligible || result.AllDeleted || result.Count != 0 || len(result.Items) != 0 || result.Inventory == nil || result.Inventory.Value != nil || len(unattachedDeletes(state.events)) != 0 {
				t.Fatal(result, err, state)
			}
			expected := 2
			if mode == "early403" || mode == "descriptor" {
				expected = 1
			}
			if len(state.events) != expected {
				t.Fatal(state.events)
			}
			if mode == "early403" || mode == "late403" {
				if !gophercloud.ResponseCodeIs(err, 403) || result.Inventory.Failure.StatusCode != 403 {
					t.Fatal(result, err)
				}
			}
			if mode == "descriptor" || mode == "malformed" || mode == "close" || mode == "source" || mode == "cancel" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || result.Inventory.Failure == nil || result.Inventory.Failure.StatusCode != 200 {
					t.Fatal(result, err)
				}
			}
			if mode == "close" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode != "early403" && len(result.Inventory.Pages) == 0 {
				t.Fatal(result.Inventory)
			}
		})
	}
}

func TestFloatingIPUnattachedSequentialInvalidIDAndDuplicateHistory(t *testing.T) {
	for _, mode := range []string{"unsafe", "missing", "duplicates"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			inventoryReturned := false
			deletes := 0
			bad := `{"id":"bad/id"}`
			if mode == "missing" {
				bad = `{}`
			}
			if mode == "duplicates" {
				bad = `{"id":"a"}`
			}
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					if inventoryReturned {
						t.Error("retry0 unexpected verification")
					}
					inventoryReturned = true
					return 200, `{"floatingips":[{"id":"a"},` + bad + `,{"id":"unused/unsafe","port_id":"attached"},{"id":"c"}]}`
				}
				deletes++
				if mode == "duplicates" && deletes == 2 {
					return 404, "duplicate already gone"
				}
				return 202, "accepted"
			}
			result, err := conn.DeleteUnattachedFloatingIPs(context.Background(), compute.WithFloatingIPDeleteRetries(0))
			if result == nil || !result.Eligible || result.AllDeleted || result.Inventory == nil || len(result.Inventory.FloatingIPs) != 4 || !result.Items[0].Deletion.Deleted || !result.Items[0].Deletion.Attempts[0].Accepted {
				t.Fatal(result, err)
			}
			if mode == "duplicates" {
				want := []string{"DELETE /delete-network/v2.0/floatingips/a", "DELETE /delete-network/v2.0/floatingips/a", "DELETE /delete-network/v2.0/floatingips/c"}
				if err != nil || result.Count != 2 || len(result.Items) != 3 || !reflect.DeepEqual(unattachedDeletes(state.events), want) || result.Items[1].Deletion.Deleted || result.Items[1].Deletion.Attempts[0].NotFound == nil {
					t.Fatal(result, err, state)
				}
			} else {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Count != 1 || deletes != 1 || len(state.events) != 2 {
					t.Fatal(result, err, state)
				}
				// Current production retains the bad candidate before its
				// route ID validation, keeping FloatingIP with Deletion=nil.
				if len(result.Items) != 2 || result.Items[1].FloatingIP == nil || result.Items[1].Deletion != nil || !errors.Is(result.Items[1].Error, resource.ErrInvalidOption) {
					t.Fatal(result)
				}
			}
		})
	}
}

func TestFloatingIPUnattachedAcceptedFailuresAndOneOperationBudget(t *testing.T) {
	for _, mode := range []string{"close", "source", "cancel", "expire", "default", "timeout", "unlimited", "parent"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
			cached, cacheErr := conn.Network(context.Background())
			if cacheErr != nil {
				t.Fatal(cacheErr)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var parentDeadline time.Time
			var parentCancel context.CancelFunc
			if mode == "parent" {
				ctx, parentCancel = context.WithTimeout(ctx, time.Minute)
				defer parentCancel()
				parentDeadline, _ = ctx.Deadline()
			}
			cause := errors.New("accepted unattached boundary")
			inventoryReturned := false
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					if !inventoryReturned {
						inventoryReturned = true
						return 200, `{"floatingips":[{"id":"a"},{"id":"b"},{"id":"c"}]}`
					}
					return 200, `{"floatingips":[]}`
				}
				return 202, "passive accepted body"
			}
			state.afterClose = func(r *http.Request) error {
				if r.Method != "DELETE" || !strings.HasSuffix(r.URL.Path, "/b") {
					return nil
				}
				if mode == "close" {
					return cause
				}
				if mode == "source" {
					cached.API = nil
				}
				if mode == "cancel" {
					cancel(cause)
				}
				if mode == "expire" {
					<-r.Context().Done()
				}
				return nil
			}
			options := []compute.FloatingIPDeleteOption{compute.WithFloatingIPDeleteRetries(0)}
			if mode == "timeout" || mode == "parent" {
				options = append(options, compute.WithFloatingIPDeleteTimeout(2*time.Minute))
			}
			if mode == "expire" {
				options = append(options, compute.WithFloatingIPDeleteTimeout(25*time.Millisecond))
			}
			if mode == "unlimited" {
				options = append(options, compute.WithFloatingIPDeleteTimeout(time.Minute), compute.WithUnlimitedFloatingIPDeleteTimeout())
			}
			result, err := conn.DeleteUnattachedFloatingIPs(ctx, options...)
			failed := mode == "close" || mode == "source" || mode == "cancel" || mode == "expire"
			if result == nil || result.Inventory == nil {
				t.Fatal(result, err)
			}
			if failed {
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || result.Count != 1 || result.AllDeleted || len(result.Items) != 2 || len(unattachedDeletes(state.events)) != 2 {
					t.Fatal(result, err, state)
				}
				item := result.Items[1]
				if item.Deletion == nil || item.Deletion.Deleted || len(item.Deletion.Attempts) != 1 || !item.Deletion.Attempts[0].Accepted || item.Deletion.Attempts[0].Verified || string(item.Deletion.Attempts[0].Response.Envelope) != "passive accepted body" || item.Deletion.Failure.StatusCode != 202 {
					t.Fatal(item)
				}
				if mode == "close" || mode == "cancel" {
					if !errors.Is(err, cause) {
						t.Fatal(err)
					}
				}
				if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if mode == "expire" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			} else if err != nil || !result.AllDeleted || result.Count != 3 || len(state.events) != 4 {
				t.Fatal(result, err, state)
			}
			// The same absolute deadline spans initial inventory and all items;
			// no per-item timeout resets. Default/unlimited add no deadline.
			for _, deadline := range state.deadlines {
				if !deadline.Equal(state.deadlines[0]) {
					t.Fatal(state.deadlines)
				}
				if mode == "default" || mode == "unlimited" {
					if !deadline.IsZero() {
						t.Fatal(deadline)
					}
				}
				if mode == "timeout" || mode == "expire" {
					if deadline.IsZero() {
						t.Fatal(deadline)
					}
				}
				if mode == "parent" && !deadline.Equal(parentDeadline) {
					t.Fatal(deadline, parentDeadline)
				}
			}
		})
	}
}

func TestFloatingIPUnattachedOwnedSnapshotsAndReusablePreparedOptions(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
	source := compute.FloatingIPNeutron
	name := "original"
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
	bulk := compute.WithFloatingIPDeleteOptions(compute.FloatingIPDeleteOpts{Retries: 1, Source: &source, Location: &location})
	source, name, location.Project.ID[1] = compute.FloatingIPNova, "changed", 'x'
	calls := 0
	option := func(*compute.FloatingIPDeleteOpts) error { calls++; return nil }
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			return 204, ""
		}
		if len(state.events) == 1 {
			return 200, `{"floatingips":[{"id":"a","description":"first"},{"id":"a","description":"second"}]}`
		}
		return 200, `{"floatingips":[{"id":"a","status":"DOWN","vendor":{"value":1}}]}`
	}
	result, err := conn.DeleteUnattachedFloatingIPs(context.Background(), bulk, option)
	if err != nil || calls != 1 || result.Count != 2 || !result.AllDeleted || len(result.Items) != 2 || len(state.events) != 5 {
		t.Fatal(result, err, state)
	}
	first, second := result.Items[0], result.Items[1]
	if first.FloatingIP == result.Inventory.FloatingIPs[0] || second.FloatingIP == result.Inventory.FloatingIPs[1] || first.FloatingIP == second.FloatingIP {
		t.Fatal("borrowed inventory/item record")
	}
	first.FloatingIP.Resource.Body["id"][1] = 'x'
	first.FloatingIP.Resource.Header.Set("X-Delete-Proof", "changed")
	queryRaw(t, result.Inventory.FloatingIPs[0].Resource, "id", `"a"`)
	queryRaw(t, second.FloatingIP.Resource, "id", `"a"`)
	queryRaw(t, first.FloatingIP.Wire, "id", `"a"`)
	first.Deletion.LastVerification.FloatingIP.Resource.Body["status"][1] = 'x'
	queryRaw(t, second.Deletion.LastVerification.FloatingIP.Resource, "status", `"DOWN"`)
	queryRaw(t, first.Deletion.LastVerification.FloatingIP.Wire, "status", `"DOWN"`)
	first.Deletion.Attempts[0].Response.Header.Set("X-Delete-Proof", "changed")
	if second.Deletion.Attempts[0].Response.Header.Get("X-Delete-Proof") == "changed" || result.Inventory.Pages[0].Header.Get("X-Delete-Proof") == "changed" {
		t.Fatal("borrowed response proof header")
	}
	// Factory input mutation and result mutation cannot change a reusable option.
	state.events, state.deadlines = nil, nil
	again, err := conn.DeleteUnattachedFloatingIPs(context.Background(), bulk, option)
	if err != nil || calls != 2 || again.Count != 2 || !again.AllDeleted || again.Inventory.Backend != compute.FloatingIPNeutron {
		t.Fatal(again, err, state)
	}
	var gotLocation resource.CloudLocation
	if err := json.Unmarshal(again.Inventory.FloatingIPs[0].Resource.Body["location"], &gotLocation); err != nil || string(gotLocation.Project.ID) != `"scope"` || gotLocation.Project.Name == nil || *gotLocation.Project.Name != "original" {
		t.Fatal(gotLocation, err)
	}

	// Even known source skip must preserve common option/context validity.
	for _, mode := range []string{"nil-option", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			_, skip, skipState := floatingDeleteFixture(t, compute.FloatingIPNone)
			skipCtx := context.Background()
			options := []compute.FloatingIPDeleteOption{nil}
			cause := errors.New("preflight cancellation")
			if mode == "cancelled" {
				cancelled, cancel := context.WithCancelCause(skipCtx)
				cancel(cause)
				skipCtx, options = cancelled, nil
			}
			_, err := skip.DeleteUnattachedFloatingIPs(skipCtx, options...)
			if err == nil || len(skipState.events) != 0 || len(skipState.locators) != 0 {
				t.Fatal(err, skipState)
			}
			if mode == "cancelled" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "nil-option" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}
