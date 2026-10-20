package containers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudContainerFaultLane struct {
	label  string
	phase  int
	prefix []int
	invoke func(context.Context, *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error)
}

func cloudContainerFaultLanes() []cloudContainerFaultLane {
	return []cloudContainerFaultLane{
		{"create initial HEAD", 0, nil, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.CreateContainer(ctx, containerMetadataName)
			if r == nil {
				return nil, nil, e
			}
			return r.Initial, r.Creation, e
		}},
		{"create PUT", 1, []int{404}, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.CreateContainer(ctx, containerMetadataName, objectstorage.WithContainerPublic(true))
			if r == nil {
				return nil, nil, e
			}
			return r.Creation, nil, e
		}},
		{"create final HEAD", 2, []int{404, 201}, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.CreateContainer(ctx, containerMetadataName)
			if r == nil {
				return nil, nil, e
			}
			return r.Refreshed, r.Creation, e
		}},
		{"set POST", 0, nil, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.SetContainerAccess(ctx, containerMetadataName, "private")
			if r == nil {
				return nil, nil, e
			}
			return r.Acknowledgement, nil, e
		}},
		{"set refresh HEAD", 1, []int{202}, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.SetContainerAccess(ctx, containerMetadataName, "public", objectstorage.WithContainerAccessRefresh(true))
			if r == nil {
				return nil, nil, e
			}
			return r.Refreshed, r.Acknowledgement, e
		}},
		{"access HEAD", 0, nil, func(ctx context.Context, c *objectstorage.Cloud) (*containers.ContainerRecord, *containers.ContainerRecord, error) {
			r, e := c.GetContainerAccess(ctx, containerMetadataName)
			if r == nil {
				return nil, nil, e
			}
			return r.Container, nil, e
		}},
	}
}

func TestSwiftCloudContainerPhysicalFaultsKeepEveryPhaseAndBlockCallbacks(t *testing.T) {
	for _, lane := range cloudContainerFaultLanes() {
		for _, status := range []int{203, 400, 401, 403, 404, 429, 503} {
			for _, fault := range []string{"read", "close", "both"} {
				t.Run(fmt.Sprint(lane.label, status, fault), func(t *testing.T) {
					readCause, closeCause := errors.New("actual read fault"), errors.New("actual close fault")
					readOnce := false
					reader := io.Reader(strings.NewReader("physical"))
					if fault != "close" {
						reader = containerMetadataReader(func(p []byte) (int, error) {
							if readOnce {
								return 0, io.EOF
							}
							readOnce = true
							return copy(p, "physical"), readCause
						})
					}
					body := &containerMetadataBody{Reader: reader}
					if fault != "read" {
						body.closeErr = closeCause
					}
					calls, retry, backoff, renew := 0, 0, 0, 0
					client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
						index := calls
						calls++
						if r.URL.String() != containerMetadataEndpoint || r.URL.RawQuery != "" || r.Body != nil {
							t.Error(r)
						}
						if index < lane.phase {
							return containerMetadataResponse(lane.prefix[index], "clean prefix"), nil
						}
						return containerMetadataWire(status, body), nil
					})
					client.ProviderClient.ReauthFunc = func(context.Context) error { renew++; return nil }
					client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
						// The clean404 prefix stops its native retry and permits
						// get-or-create; only hooks after the injected fault count.
						if calls <= lane.phase {
							return original
						}
						retry++
						return nil
					}
					client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
						backoff++
						return nil
					}
					client.ProviderClient.MaxBackoffRetries = 3
					record, ack, err := lane.invoke(context.Background(), objectstorage.New(client))
					if err == nil || record == nil || record.Resource != nil || record.StatusCode != status || string(record.Envelope) != "physical" || record.Header.Get("X-Trans-Id") != "actual-container" || calls != lane.phase+1 || body.closes.Load() != 1 || retry != 0 || backoff != 0 || renew != 0 {
						t.Fatalf("record=%+v ack=%+v err=%v HTTP=%d closes=%d hooks=%d/%d/%d", record, ack, err, calls, body.closes.Load(), retry, backoff, renew)
					}
					if fault != "close" && !errors.Is(err, readCause) {
						t.Fatal("lost read cause", err)
					}
					if fault != "read" && !errors.Is(err, closeCause) {
						t.Fatal("lost close cause", err)
					}
					if status < 400 {
						containerMetadataProof(t, err, status, "physical")
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if status == 401 {
							var actual *gophercloud.ErrUnableToReauthenticate
							if !errors.As(err, &actual) || !errors.As(actual.ErrOriginal, &native) {
								t.Fatal("lost native pointer 401", err)
							}
							if fault != "close" && !errors.Is(actual.ErrReauth, readCause) {
								t.Fatal(actual.ErrReauth)
							}
							if fault != "read" && !errors.Is(actual.ErrReauth, closeCause) {
								t.Fatal(actual.ErrReauth)
							}
						} else if !errors.As(err, &native) {
							t.Fatal(err)
						}
						if native.Actual != status || string(native.Body) != "physical" || native.ResponseHeader.Get("X-Trans-Id") != "actual-container" {
							t.Fatal(native)
						}
					}
					if lane.label == "create final HEAD" && (ack == nil || ack.StatusCode != 201) {
						t.Fatal(ack)
					}
					if lane.label == "set refresh HEAD" && (ack == nil || ack.StatusCode != 202) {
						t.Fatal(ack)
					}
				})
			}
		}
	}
}

func TestSwiftCloudContainerFaultyOrExpanded404NeverEstablishesAbsence(t *testing.T) {
	for _, mode := range []string{"expanded", "callback error"} {
		t.Run(mode, func(t *testing.T) {
			calls, hooks := 0, 0
			cause := errors.New("retry policy rejected")
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return containerMetadataResponse(404, "physical missing"), nil
			})
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				if mode == "callback error" {
					return cause
				}
				opts.OkCodes = append(opts.OkCodes, 404)
				return nil
			}
			result, err := objectstorage.New(client).CreateContainer(context.Background(), containerMetadataName)
			if err == nil || result == nil || result.Container != nil || result.Creation != nil || result.Initial == nil || result.Initial.StatusCode != 404 {
				t.Fatal(result, err)
			}
			if mode == "expanded" {
				if calls != 2 || hooks != 1 || !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(err, calls, hooks)
				}
			} else if calls != 1 || hooks != 1 || !errors.Is(err, cause) {
				t.Fatal(err, calls, hooks)
			}
		})
	}
}

func TestSwiftCloudContainerCancellationKeepsActualAcknowledgementWithoutReplay(t *testing.T) {
	for _, lane := range cloudContainerFaultLanes() {
		t.Run(lane.label, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cause := errors.New("cancelled at actual Close")
			calls := 0
			body := &containerMetadataBody{Reader: strings.NewReader("actual"), onClose: func() { cancel(cause) }}
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				index := calls
				calls++
				if index < lane.phase {
					return containerMetadataResponse(lane.prefix[index], "prefix"), nil
				}
				return containerMetadataWire(203, body), nil
			})
			record, ack, err := lane.invoke(ctx, objectstorage.New(client))
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || record == nil || record.StatusCode != 203 || record.Resource != nil || string(record.Envelope) != "actual" || calls != lane.phase+1 || body.closes.Load() != 1 {
				t.Fatal(record, ack, err, calls)
			}
			containerMetadataProof(t, err, 203, "actual")
		})
	}
}

func TestSwiftCloudContainerSourceDriftIsStickyAcrossReadsAndClose(t *testing.T) {
	for _, lane := range cloudContainerFaultLanes() {
		t.Run(lane.label, func(t *testing.T) {
			calls := 0
			var client *gophercloud.ServiceClient
			readOnce := false
			reader := containerMetadataReader(func(p []byte) (int, error) {
				if readOnce {
					return 0, io.EOF
				}
				readOnce = true
				client.ResourceBase = containerMetadataBase
				return copy(p, "actual"), nil
			})
			body := &containerMetadataBody{Reader: reader, onClose: func() { client.ResourceBase = "" }}
			client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
				index := calls
				calls++
				if index < lane.phase {
					return containerMetadataResponse(lane.prefix[index], "prefix"), nil
				}
				return containerMetadataWire(203, body), nil
			})
			record, _, err := lane.invoke(context.Background(), objectstorage.New(client))
			if !errors.Is(err, resource.ErrInvalidOption) || record == nil || record.Resource != nil || calls != lane.phase+1 || body.closes.Load() != 1 || client.ResourceBase != "" {
				t.Fatal(record, err, calls)
			}
			containerMetadataProof(t, err, 203, "actual")
		})
	}
}

func TestSwiftCloudContainerClean401RenewalKeepsMethodACLAndLiveToken(t *testing.T) {
	for _, operation := range []string{"create", "set", "get"} {
		t.Run(operation, func(t *testing.T) {
			calls, renew := 0, 0
			var methods []string
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				methods = append(methods, r.Method)
				if calls == 1 {
					return containerMetadataResponse(401, "clean expired"), nil
				}
				if r.Header.Get("X-Auth-Token") != "renewed-token" {
					t.Error(r.Header)
				}
				if operation == "create" && r.Method == "HEAD" && calls == 2 {
					return containerMetadataResponse(404, "absent"), nil
				}
				if operation == "create" && r.Method == "PUT" && r.Header.Get("X-Container-Read") != ".r:*,.rlistings" {
					t.Error(r.Header)
				}
				if operation == "set" {
					values, present := r.Header["X-Container-Read"]
					if !present || !reflect.DeepEqual(values, []string{""}) {
						t.Error(r.Header)
					}
				}
				return containerMetadataResponse(203, ""), nil
			})
			client.ProviderClient.ReauthFunc = func(context.Context) error { renew++; client.ProviderClient.SetToken("renewed-token"); return nil }
			cloud := objectstorage.New(client)
			var err error
			switch operation {
			case "create":
				_, err = cloud.CreateContainer(context.Background(), containerMetadataName, objectstorage.WithContainerPublic(true))
			case "set":
				_, err = cloud.SetContainerAccess(context.Background(), containerMetadataName, "private")
			case "get":
				_, err = cloud.GetContainerAccess(context.Background(), containerMetadataName)
			}
			want := []string{"HEAD", "HEAD", "PUT", "HEAD"}
			if operation == "set" {
				want = []string{"POST", "POST"}
			} else if operation == "get" {
				want = []string{"HEAD", "HEAD"}
			}
			if err != nil || renew != 1 || !reflect.DeepEqual(methods, want) {
				t.Fatal(err, renew, methods)
			}
		})
	}
}

func TestSwiftCloudContainerHeadAndPostIgnoreJSONBodiesButRetainPhysicalEnvelope(t *testing.T) {
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		response := containerMetadataResponse(203, `{"name":"unrelated","read_ACL":"fake","foreign":true}`)
		if r.Method == "HEAD" {
			response.Header.Set("X-Container-Read", ".r:*,.rlistings")
		}
		return response, nil
	})
	cloud := objectstorage.New(client)
	found, err := cloud.GetContainerAccess(context.Background(), containerMetadataName)
	if err != nil || found.Access != "public" || string(found.Container.Resource.Body["name"]) == `"unrelated"` || string(found.Container.Envelope) != `{"name":"unrelated","read_ACL":"fake","foreign":true}` || len(found.Container.Wire.Body) != 0 {
		t.Fatal(found, err)
	}
	updated, err := cloud.SetContainerAccess(context.Background(), containerMetadataName, "public")
	if err != nil || string(updated.Container.Resource.Body["read_ACL"]) != "null" || len(updated.Container.Wire.Body) != 0 {
		t.Fatal(updated, err)
	}
}

func TestSwiftCloudContainerDescriptorFailuresKeepActualHeadAndPreventCreate(t *testing.T) {
	calls := 0
	client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
		calls++
		response := containerMetadataResponse(203, "physical")
		// Source's int descriptor maps ordinary non-digit strings to zero.
		// Superscript two is isdigit=True but int("²") raises ValueError.
		response.Header.Set("X-Container-Object-Count", "²")
		return response, nil
	})
	result, err := objectstorage.New(client).CreateContainer(context.Background(), containerMetadataName)
	if err == nil || result == nil || result.Creation != nil || result.Initial == nil || result.Initial.Resource != nil || result.Initial.StatusCode != 203 || result.Initial.Header.Get("X-Container-Object-Count") != "²" || calls != 1 {
		t.Fatal(result, err, calls)
	}
	containerMetadataProof(t, err, 203, "physical")
}

func TestSwiftCloudContainerCleanRetryCannotChangeSelectedACLOrBodylessFraming(t *testing.T) {
	for _, operation := range []string{"create public ACL", "set private ACL", "get body"} {
		t.Run(operation, func(t *testing.T) {
			calls, hooks := 0, 0
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if operation == "create public ACL" && calls == 1 {
					return containerMetadataResponse(404, "absent"), nil
				}
				return containerMetadataResponse(503, "actual rejection"), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, c *gophercloud.RequestOpts, original error, _ uint) error {
				if operation == "create public ACL" && calls == 1 {
					return original
				}
				hooks++
				if operation == "get body" {
					c.RawBody = strings.NewReader("changed framing")
				} else {
					c.MoreHeaders["X-Container-Read"] = "changed ACL"
				}
				return nil
			}
			cloud := objectstorage.New(client)
			var record *containers.ContainerRecord
			var err error
			switch operation {
			case "create public ACL":
				r, e := cloud.CreateContainer(context.Background(), containerMetadataName, objectstorage.WithContainerPublic(true))
				record, err = r.Creation, e
			case "set private ACL":
				r, e := cloud.SetContainerAccess(context.Background(), containerMetadataName, "private")
				record, err = r.Acknowledgement, e
			case "get body":
				r, e := cloud.GetContainerAccess(context.Background(), containerMetadataName)
				record, err = r.Container, e
			}
			expected := 1
			if operation == "create public ACL" {
				expected = 2
			}
			if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || record == nil || record.Resource != nil || record.StatusCode != 503 || string(record.Envelope) != "actual rejection" || calls != expected || hooks != 1 {
				t.Fatal(record, err, calls, hooks)
			}
		})
	}
}
