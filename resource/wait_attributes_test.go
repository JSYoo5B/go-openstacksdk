package resource_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestWaitStatusAttributeSelectsNativeNodeFields(t *testing.T) {
	for _, attribute := range []string{"power_state", "PowerState", "provision_state", "ProvisionState"} {
		t.Run(attribute, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
				if gets.Add(1) == 1 {
					testcloud.JSON(w, 200, `{"uuid":"changed","power_state":"power off","provision_state":"deploying"}`)
				} else {
					testcloud.JSON(w, 200, `{"uuid":"fixed","power_state":"power on","provision_state":"active"}`)
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("polling identity changed: %s", r.URL)
				http.Error(w, "unexpected target", 500)
			})
			target := "POWER ON"
			if attribute == "provision_state" || attribute == "ProvisionState" {
				target = "ACTIVE"
			}
			value, err := nodes.New(cloud.Client("baremetal", "/v1")).Resources.Wait(context.Background(), resource.ID("fixed"), target,
				resource.WithStatusAttribute(attribute), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
			if err != nil || value == nil || value.UUID != "fixed" || gets.Load() != 2 {
				t.Fatalf("value=%v gets=%d error=%v", value, gets.Load(), err)
			}
		})
	}
}

func TestWaitStatusAttributeUsesExtendedServerJSONTag(t *testing.T) {
	for _, attribute := range []string{"TaskState", "OS-EXT-STS:task_state"} {
		t.Run(attribute, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/servers/fixed", func(w http.ResponseWriter, r *http.Request) {
				state := "spawning"
				if gets.Add(1) > 1 {
					state = "ready"
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"fixed","status":"BUILD","OS-EXT-STS:task_state":%q,"progress":25}}`, state))
			})
			var progress []int
			value, err := servers.New(cloud.Client("compute", "/compute")).Resources.Wait(context.Background(), resource.ID("fixed"), "READY",
				resource.WithStatusAttribute(attribute), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
				resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
			if err != nil || value == nil || value.TaskState != "ready" || gets.Load() != 2 || !reflect.DeepEqual(progress, []int{25}) {
				t.Fatalf("value=%v gets=%d progress=%v error=%v", value, gets.Load(), progress, err)
			}
		})
	}
}

func TestWaitProgressIncludesEachNonterminalNativeServerResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/servers/fixed", func(w http.ResponseWriter, r *http.Request) {
		index := int(gets.Add(1)) - 1
		progress := []int{0, 25, 75, 100}
		if index >= len(progress) {
			t.Errorf("extra polling request: %d", index+1)
			http.Error(w, "unexpected poll", 500)
			return
		}
		state := "BUILD"
		if index == 3 {
			state = "ACTIVE"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"changed","status":%q,"progress":%d}}`, state, progress[index]))
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("response redirected polling: %s", r.URL)
		http.Error(w, "unexpected target", 500)
	})
	var progress []int
	value, err := servers.New(cloud.Client("compute", "/compute")).Resources.Wait(context.Background(), resource.ID("fixed"), "ACTIVE",
		resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || value == nil || gets.Load() != 4 || !reflect.DeepEqual(progress, []int{0, 25, 75}) {
		t.Fatalf("value=%v gets=%d progress=%v error=%v", value, gets.Load(), progress, err)
	}
}

func TestWaitProgressOmitsSuccessFailureAndHTTPError(t *testing.T) {
	for _, terminal := range []string{"ACTIVE", "ERROR", "404"} {
		for _, immediate := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/immediate=%t", terminal, immediate), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets atomic.Int32
				cloud.Mux.HandleFunc("GET /compute/servers/fixed", func(w http.ResponseWriter, r *http.Request) {
					if gets.Add(1) == 1 && !immediate {
						testcloud.JSON(w, 200, `{"server":{"id":"fixed","status":"BUILD","progress":17}}`)
						return
					}
					if terminal == "404" {
						http.Error(w, "missing", 404)
						return
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"fixed","status":%q,"progress":100}}`, terminal))
				})
				var progress []int
				_, err := servers.New(cloud.Client("compute", "/compute")).Resources.Wait(context.Background(), resource.ID("fixed"), "ACTIVE",
					resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
					resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
				wantGets := int32(2)
				wantProgress := []int{17}
				if immediate {
					wantGets, wantProgress = 1, nil
				}
				if gets.Load() != wantGets || !reflect.DeepEqual(progress, wantProgress) {
					t.Fatalf("gets=%d progress=%v", gets.Load(), progress)
				}
				if terminal == "ACTIVE" && err != nil || terminal == "ERROR" && !errors.Is(err, resource.ErrFailedState) || terminal == "404" && !errors.Is(err, resource.ErrNotFound) {
					t.Fatalf("terminal=%s error=%v", terminal, err)
				}
			})
		}
	}
}

func TestWaitProgressCallbackCancellationStopsFurtherGETs(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		t.Run(fmt.Sprintf("deletion=%t", deletion), func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/servers/fixed", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"server":{"id":"changed","status":"BUILD","progress":9}}`)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var progress []int
			options := []resource.WaitOption{resource.WithProgressCallback(func(value int) { progress = append(progress, value); cancel() }), resource.WithPollInterval(time.Hour)}
			collection := servers.New(cloud.Client("compute", "/compute")).Resources
			var err error
			if deletion {
				err = collection.WaitDeleted(ctx, resource.ID("fixed"), options...)
			} else {
				_, err = collection.Wait(ctx, resource.ID("fixed"), "ACTIVE", options...)
			}
			if !errors.Is(err, context.Canceled) || gets.Load() != 1 || !reflect.DeepEqual(progress, []int{9}) {
				t.Fatalf("gets=%d progress=%v error=%v", gets.Load(), progress, err)
			}
		})
	}
}

type waitAttributeEntry struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"`
	Nullable *string `json:"nullable"`
	Number   int     `json:"number"`
}

func TestWaitAttributesAndCallbackRejectInvalidPoliciesBeforeFetch(t *testing.T) {
	for _, test := range []struct {
		name   string
		option resource.WaitOption
		want   error
	}{
		{"unknown", resource.WithStatusAttribute("does_not_exist"), resource.ErrUnsupported},
		{"nonstring", resource.WithStatusAttribute("number"), resource.ErrUnsupported},
		{"empty", resource.WithStatusAttribute(" "), resource.ErrInvalidOption},
		{"nil callback", resource.WithProgressCallback(nil), resource.ErrInvalidOption},
	} {
		t.Run(test.name, func(t *testing.T) {
			validation := resource.ValidateWaitOptionsFor[waitAttributeEntry](test.option)
			if !errors.Is(validation, test.want) {
				t.Fatalf("validation error=%v", validation)
			}
			calls := 0
			collection := resource.NewCollection(resource.Adapter[waitAttributeEntry]{
				Kind: "entry", ID: func(value *waitAttributeEntry) string { return value.ID }, Status: func(value *waitAttributeEntry) string { return value.Status },
				Get: func(context.Context, string) (*waitAttributeEntry, error) {
					calls++
					return &waitAttributeEntry{ID: "fixed", Status: "ready"}, nil
				},
			})
			_, err := collection.Wait(context.Background(), resource.ID("fixed"), "ready", test.option)
			if calls != 0 || !errors.Is(err, test.want) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			if err := collection.WaitDeleted(context.Background(), resource.ID("fixed"), test.option); !errors.Is(err, test.want) || calls != 0 {
				t.Fatalf("deletion calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestWaitNullStatusAttributeReturnsError(t *testing.T) {
	calls := 0
	collection := resource.NewCollection(resource.Adapter[waitAttributeEntry]{
		Kind: "entry", ID: func(value *waitAttributeEntry) string { return value.ID }, Status: func(value *waitAttributeEntry) string { return value.Status },
		Get: func(context.Context, string) (*waitAttributeEntry, error) {
			calls++
			return &waitAttributeEntry{ID: "fixed", Status: "ready"}, nil
		},
	})
	_, err := collection.Wait(context.Background(), resource.ID("fixed"), "ready", resource.WithStatusAttribute("nullable"))
	if calls != 1 || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}

func TestWaitStatusAttributeSupportsStringPointersWithoutDefaultStatus(t *testing.T) {
	ready := "ready"
	calls := 0
	collection := resource.NewCollection(resource.Adapter[waitAttributeEntry]{
		Kind: "entry", ID: func(value *waitAttributeEntry) string { return value.ID },
		Get: func(context.Context, string) (*waitAttributeEntry, error) {
			calls++
			return &waitAttributeEntry{ID: "fixed", Nullable: &ready}, nil
		},
	})
	value, err := collection.Wait(context.Background(), resource.ID("fixed"), "READY", resource.WithStatusAttribute("Nullable"))
	if err != nil || value == nil || calls != 1 || value.Nullable == nil || *value.Nullable != ready {
		t.Fatalf("value=%v calls=%d error=%v", value, calls, err)
	}
}

func TestWaitProgressDefaultsToZeroWithoutNativeProgress(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
		state := "deploying"
		if gets.Add(1) > 1 {
			state = "active"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":"fixed","provision_state":%q}`, state))
	})
	var progress []int
	_, err := nodes.New(cloud.Client("baremetal", "/v1")).Resources.Wait(context.Background(), resource.ID("fixed"), "active",
		resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || gets.Load() != 2 || !reflect.DeepEqual(progress, []int{0}) {
		t.Fatalf("gets=%d progress=%v error=%v", gets.Load(), progress, err)
	}
}

func TestWaitStatusAttributeAlsoSelectsFailureState(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
		state := "power off"
		if gets.Add(1) > 1 {
			state = "broken"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":"changed","provision_state":"active","power_state":%q}`, state))
	})
	var progress []int
	_, err := nodes.New(cloud.Client("baremetal", "/v1")).Resources.Wait(context.Background(), resource.ID("fixed"), "power on",
		resource.WithStatusAttribute("power_state"), resource.WithFailureStates("BROKEN"),
		resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	var failure *resource.FailedStateError
	if !errors.As(err, &failure) || failure.ID != "fixed" || failure.Status != "broken" || gets.Load() != 2 || !reflect.DeepEqual(progress, []int{0}) {
		t.Fatalf("failure=%v gets=%d progress=%v error=%v", failure, gets.Load(), progress, err)
	}
}

func TestWaitDeletedStatusAttributeUsesNativeNodeField(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
		state := "power off"
		if gets.Add(1) > 1 {
			state = "DeLeTeD"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":"changed","provision_state":"deploying","power_state":%q}`, state))
	})
	var progress []int
	err := nodes.New(cloud.Client("baremetal", "/v1")).Resources.WaitDeleted(context.Background(), resource.ID("fixed"),
		resource.WithStatusAttribute("power_state"), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || gets.Load() != 2 || !reflect.DeepEqual(progress, []int{0}) {
		t.Fatalf("gets=%d progress=%v error=%v", gets.Load(), progress, err)
	}
}

func TestWaitDeletedProgressStopsAtDeletedStatusOr404(t *testing.T) {
	for _, terminal := range []string{"deleted", "404"} {
		t.Run(terminal, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/servers/fixed", func(w http.ResponseWriter, r *http.Request) {
				if gets.Add(1) == 1 {
					testcloud.JSON(w, 200, `{"server":{"id":"changed","status":"DELETING","progress":44}}`)
					return
				}
				if terminal == "404" {
					http.Error(w, "missing", 404)
					return
				}
				testcloud.JSON(w, 200, `{"server":{"id":"changed","status":"DeLeTeD","progress":100}}`)
			})
			var progress []int
			err := servers.New(cloud.Client("compute", "/compute")).Resources.WaitDeleted(context.Background(), resource.ID("fixed"),
				resource.WithProgressCallback(func(value int) { progress = append(progress, value) }),
				resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
			if err != nil || gets.Load() != 2 || !reflect.DeepEqual(progress, []int{44}) {
				t.Fatalf("gets=%d progress=%v error=%v", gets.Load(), progress, err)
			}
		})
	}
}

func TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback(t *testing.T) {
	calls, callbacks := 0, 0
	collection := resource.NewCollection(resource.Adapter[waitAttributeEntry]{
		Kind: "entry", Get: func(context.Context, string) (*waitAttributeEntry, error) { calls++; return nil, nil },
	})
	err := collection.WaitDeleted(context.Background(), resource.ID("fixed"), resource.WithProgressCallback(func(int) { callbacks++ }), resource.WithTimeout(time.Second))
	if err != nil || calls != 1 || callbacks != 0 {
		t.Fatalf("calls=%d callbacks=%d error=%v", calls, callbacks, err)
	}
}
