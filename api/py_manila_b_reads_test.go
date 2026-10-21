package api_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/messages"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/replicas"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/schedulerstats"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetransfers"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetypes"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

const pyManilaBVersion = "2.80"

// Python Proxy._get issues one GET on the resource path with the session
// microversion. Go Get does the same with the client microversion and keeps a
// 404 as the native HTTP error instead of a NotFoundException.
func TestPyManilaBGetOperationsIssueOneFetchWithConfiguredMicroversion(t *testing.T) {
	t.Parallel()
	type getter struct {
		name, path, body string
		call             func(client *gophercloud.ServiceClient) (string, error)
	}
	ctx := context.Background()
	cases := []getter{
		{"get_share_network", "/v2/share-networks/sn-1", `{"share_network":{"id":"sn-1","name":"net","neutron_net_id":"n1"}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := sharenetworks.New(c).Get(ctx, "sn-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.NeutronNetID, nil
			}},
		{"get_share_replica", "/v2/share-replicas/rep-1", `{"share_replica":{"id":"rep-1","share_id":"sh-1","status":"available","replica_state":"in_sync"}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := replicas.New(c).Get(ctx, "rep-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.ShareID + "/" + v.State, nil
			}},
		{"get_share_snapshot", "/v2/snapshots/snap-1", `{"snapshot":{"id":"snap-1","share_id":"sh-1","size":2,"status":"available"}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := snapshots.New(c).Get(ctx, "snap-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.ShareID + "/" + v.Status, nil
			}},
		{"get_share_transfer", "/v2/share-transfers/tr-1", `{"transfer":{"id":"tr-1","resource_id":"sh-1","name":"hand-off"}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := sharetransfers.New(c).Get(ctx, "tr-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.ResourceID + "/" + v.Name, nil
			}},
		{"get_share_type", "/v2/types/ty-1", `{"share_type":{"id":"ty-1","name":"gold","extra_specs":{"snapshot_support":"True"}}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := sharetypes.New(c).Get(ctx, "ty-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.Name + "/" + fmt.Sprint(v.ExtraSpecs["snapshot_support"]), nil
			}},
		{"get_user_message", "/v2/messages/m-1", `{"message":{"id":"m-1","action_id":"001","message_level":"ERROR"}}`,
			func(c *gophercloud.ServiceClient) (string, error) {
				v, err := messages.New(c).Get(ctx, "m-1")
				if err != nil {
					return "", err
				}
				return v.ID + "/" + v.ActionID + "/" + v.MessageLevel, nil
			}},
	}
	want := map[string]string{
		"get_share_network":  "sn-1/n1",
		"get_share_replica":  "rep-1/sh-1/in_sync",
		"get_share_snapshot": "snap-1/sh-1/available",
		"get_share_transfer": "tr-1/sh-1/hand-off",
		"get_share_type":     "ty-1/gold/True",
		"get_user_message":   "m-1/001/ERROR",
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET " + tc.path: {status: 200, body: tc.body}})
			got, err := tc.call(cloud.client(pyManilaBVersion))
			if err != nil || got != want[tc.name] {
				t.Fatalf("got %q error %v", got, err)
			}
			cloud.expect(pyManilaBCall{method: "GET", path: tc.path, version: pyManilaBVersion})

			missing := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET " + tc.path: {status: 404, body: `{"itemNotFound":{"message":"gone"}}`}})
			_, err = tc.call(missing.client(pyManilaBVersion))
			if err == nil || !gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("404 error=%v", err)
			}
			missing.expect(pyManilaBCall{method: "GET", path: tc.path, version: pyManilaBVersion})
		})
	}
}

func pyManilaBIDs[T any](t *testing.T, seq func(yield func(*T, error) bool), id func(*T) string) []string {
	t.Helper()
	var ids []string
	for value, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id(value))
	}
	return ids
}

// Python Proxy list methods choose the /detail path when details=True (the
// default for shares, networks, snapshots and pools) and the summary path
// otherwise, and send the mapped filters as query parameters. Go exposes both
// routes as List and ListDetail with the same wire filters.
func TestPyManilaBListOperationsUsePythonRoutesAndFilterNames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	v := pyManilaBVersion
	sum := func(key string) string { return `{"` + key + `":[{"id":"a"},{"id":"b"}]}` }
	run := func(t *testing.T, path, body string, call func(*gophercloud.ServiceClient) []string, query string) {
		t.Helper()
		probe := ""
		if strings.Contains(path, "share-networks") {
			probe = `{"share_networks":[]}`
		}
		cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET " + path: {status: 200, body: body, probe: probe}})
		ids := call(cloud.client(v))
		if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
			t.Fatalf("ids=%v", ids)
		}
		want := []pyManilaBCall{{method: "GET", path: path, query: query, version: v}}
		if probe != "" {
			// The native share-network pager asks once more at the maximum offset when no limit is set.
			want = append(want, pyManilaBCall{method: "GET", path: path, query: strings.TrimPrefix(query+"&offset=9223372036854775807", "&"), version: v})
		}
		cloud.expect(want...)
	}
	shareID := func(s *shares.Share) string { return s.ID }
	t.Run("shares details=True", func(t *testing.T) {
		run(t, "/v2/shares/detail", sum("shares"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, shares.New(c).ListDetail(ctx, shares.WithListDetailOptions(shares.ListOpts{AllTenants: true, Name: "data", Status: "available", ProjectID: "p1"})), shareID)
		}, "all_tenants=true&name=data&status=available&project_id=p1")
	})
	t.Run("shares details=False", func(t *testing.T) {
		run(t, "/v2/shares", sum("shares"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, shares.New(c).List(ctx, shares.WithListOptions(shares.ListOpts{AllTenants: true, Name: "data", Status: "available", ProjectID: "p1"})), shareID)
		}, "all_tenants=true&name=data&status=available&project_id=p1")
	})
	t.Run("share_networks details=True", func(t *testing.T) {
		run(t, "/v2/share-networks/detail", sum("share_networks"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, sharenetworks.New(c).ListDetail(ctx, sharenetworks.WithListDetailOptions(sharenetworks.ListOpts{AllTenants: true, ProjectID: "p1", Name: "net"})), func(s *sharenetworks.ShareNetwork) string { return s.ID })
		}, "all_tenants=true&project_id=p1&name=net")
	})
	t.Run("share_networks details=False", func(t *testing.T) {
		run(t, "/v2/share-networks", sum("share_networks"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, sharenetworks.New(c).List(ctx, sharenetworks.WithListOptions(sharenetworks.ListOpts{AllTenants: true, ProjectID: "p1", Name: "net"})), func(s *sharenetworks.ShareNetwork) string { return s.ID })
		}, "all_tenants=true&project_id=p1&name=net")
	})
	t.Run("share_snapshots details=True", func(t *testing.T) {
		run(t, "/v2/snapshots/detail", sum("snapshots"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, snapshots.New(c).ListDetail(ctx, snapshots.WithListDetailQuery("snapshot_id", "snap-9")), func(s *snapshots.Snapshot) string { return s.ID })
		}, "snapshot_id=snap-9")
	})
	t.Run("share_snapshots details=False", func(t *testing.T) {
		run(t, "/v2/snapshots", sum("snapshots"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, snapshots.New(c).List(ctx, snapshots.WithListQuery("snapshot_id", "snap-9")), func(s *snapshots.Snapshot) string { return s.ID })
		}, "snapshot_id=snap-9")
	})
	t.Run("share_replicas", func(t *testing.T) {
		run(t, "/v2/share-replicas", sum("share_replicas"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, replicas.New(c).List(ctx), func(s *replicas.Replica) string { return s.ID })
		}, "")
	})
	t.Run("share_transfers details=False", func(t *testing.T) {
		run(t, "/v2/share-transfers", sum("transfers"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, sharetransfers.New(c).List(ctx, sharetransfers.WithListOptions(sharetransfers.ListOpts{Name: "hand-off", AllTenants: true})), func(s *sharetransfers.Transfer) string { return s.ID })
		}, "name=hand-off&all_tenants=true")
	})
	t.Run("share_transfers details=True", func(t *testing.T) {
		run(t, "/v2/share-transfers/detail", sum("transfers"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, sharetransfers.New(c).ListDetail(ctx, sharetransfers.WithListDetailOptions(sharetransfers.ListOpts{Name: "hand-off", AllTenants: true})), func(s *sharetransfers.Transfer) string { return s.ID })
		}, "name=hand-off&all_tenants=true")
	})
	t.Run("share_types with is_public and JSON extra_specs", func(t *testing.T) {
		run(t, "/v2/types", sum("share_types"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, sharetypes.New(c).List(ctx, sharetypes.WithListQuery("is_public", "True"), sharetypes.WithListQuery("extra_specs", `{"snapshot_support":"True"}`)), func(s *sharetypes.ShareType) string { return s.ID })
		}, "is_public=True&extra_specs="+`%7B%22snapshot_support%22%3A%22True%22%7D`)
	})
	t.Run("user_messages", func(t *testing.T) {
		run(t, "/v2/messages", sum("messages"), func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, messages.New(c).List(ctx, messages.WithListOptions(messages.ListOpts{ResourceID: "sh-1", ResourceType: "SHARE", MessageLevel: "ERROR", SortKey: "created_at", SortDir: "desc"})), func(s *messages.Message) string { return s.ID })
		}, "resource_id=sh-1&resource_type=SHARE&message_level=ERROR&sort_key=created_at&sort_dir=desc")
	})
	t.Run("services with Python name and availability_zone aliases", func(t *testing.T) {
		run(t, "/v2/services", `{"services":[{"id":1},{"id":2}]}`, func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, services.New(c).List(ctx, services.WithListQuery("binary", "manila-share"), services.WithListQuery("zone", "nova"), services.WithListQuery("host", "h1")), func(s *services.Service) string { return string(rune('a' + s.ID - 1)) })
		}, "binary=manila-share&zone=nova&host=h1")
	})
	t.Run("storage_pools details=True", func(t *testing.T) {
		run(t, "/v2/scheduler-stats/pools/detail", `{"pools":[{"name":"a"},{"name":"b"}]}`, func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, schedulerstats.New(c).ListDetail(ctx, schedulerstats.WithListDetailQuery("pool", "p"), schedulerstats.WithListDetailQuery("share_type", "gold")), func(s *schedulerstats.Pool) string { return s.Name })
		}, "pool=p&share_type=gold")
	})
	t.Run("storage_pools details=False", func(t *testing.T) {
		run(t, "/v2/scheduler-stats/pools", `{"pools":[{"name":"a"},{"name":"b"}]}`, func(c *gophercloud.ServiceClient) []string {
			return pyManilaBIDs(t, schedulerstats.New(c).List(ctx, schedulerstats.WithListQuery("backend", "b")), func(s *schedulerstats.Pool) string { return s.Name })
		}, "backend=b")
	})
}

// The summary lists keep the Gophercloud offset paging, reject nil options
// before HTTP and stop on cancellation without a request.
func TestPyManilaBSummaryListsFollowOffsetPagingAndRejectBadOptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	type lister struct {
		name, path, key string
		list            func(*gophercloud.ServiceClient, ...int) ([]string, error)
		nilOption       func(*gophercloud.ServiceClient) error
	}
	collect := func(seq func(yield func(string, error) bool)) ([]string, error) {
		var ids []string
		for id, err := range seq {
			if err != nil {
				return ids, err
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	cases := []lister{
		{"shares", "/v2/shares", "shares",
			func(c *gophercloud.ServiceClient, limit ...int) ([]string, error) {
				return collect(func(yield func(string, error) bool) {
					for v, err := range shares.New(c).List(ctx, shares.WithListOptions(shares.ListOpts{Limit: 1})) {
						var id string
						if v != nil {
							id = v.ID
						}
						if !yield(id, err) {
							return
						}
					}
				})
			},
			func(c *gophercloud.ServiceClient) error {
				for _, err := range shares.New(c).List(ctx, nil) {
					return err
				}
				return nil
			}},
		{"snapshots", "/v2/snapshots", "snapshots",
			func(c *gophercloud.ServiceClient, limit ...int) ([]string, error) {
				return collect(func(yield func(string, error) bool) {
					for v, err := range snapshots.New(c).List(ctx, snapshots.WithListOptions(snapshots.ListOpts{Limit: 1})) {
						var id string
						if v != nil {
							id = v.ID
						}
						if !yield(id, err) {
							return
						}
					}
				})
			},
			func(c *gophercloud.ServiceClient) error {
				for _, err := range snapshots.New(c).List(ctx, nil) {
					return err
				}
				return nil
			}},
		{"share networks", "/v2/share-networks", "share_networks",
			func(c *gophercloud.ServiceClient, limit ...int) ([]string, error) {
				return collect(func(yield func(string, error) bool) {
					for v, err := range sharenetworks.New(c).List(ctx, sharenetworks.WithListOptions(sharenetworks.ListOpts{Limit: 1})) {
						var id string
						if v != nil {
							id = v.ID
						}
						if !yield(id, err) {
							return
						}
					}
				})
			},
			func(c *gophercloud.ServiceClient) error {
				for _, err := range sharenetworks.New(c).List(ctx, nil) {
					return err
				}
				return nil
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET " + tc.path: {status: 200, bodies: []string{
				`{"` + tc.key + `":[{"id":"a"}]}`, `{"` + tc.key + `":[{"id":"b"}]}`, `{"` + tc.key + `":[]}`}}})
			ids, err := tc.list(cloud.client(pyManilaBVersion))
			if err != nil || len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
				t.Fatalf("ids=%v error=%v", ids, err)
			}
			cloud.expect(
				pyManilaBCall{method: "GET", path: tc.path, query: "limit=1", version: pyManilaBVersion},
				pyManilaBCall{method: "GET", path: tc.path, query: "limit=1&offset=1", version: pyManilaBVersion},
				pyManilaBCall{method: "GET", path: tc.path, query: "limit=1&offset=2", version: pyManilaBVersion},
			)
			rejected := newPyManilaBCloud(t, nil)
			if err := tc.nilOption(rejected.client(pyManilaBVersion)); err == nil {
				t.Fatal("nil option accepted")
			}
			rejected.expect()
		})
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	cloud := newPyManilaBCloud(t, nil)
	for _, err := range shares.New(cloud.client(pyManilaBVersion)).List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	}
	cloud.expect()
}
