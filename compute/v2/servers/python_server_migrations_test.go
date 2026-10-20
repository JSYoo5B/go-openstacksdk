package servers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func serverMigrationField(t *testing.T, row *servers.ServerMigrationRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}
func migrationVersion() servers.ServerMigrationOption {
	return servers.WithServerMigrationMicroversion("2.80")
}

func TestPythonServerMigrationListIsLazyAndKeepsUntypedProjection(t *testing.T) {
	var calls []nativeServerCall
	body := `{"migrations":[{"uuid":"uuid-only","status":false,"created_at":"literal-date","memory_total_bytes":"large","disk_total_bytes":9007199254740993,"x_unknown":1,"server_id":"forged"},{"id":null,"uuid":"fallback"}]}`
	api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Method != "GET" || r.URL.Path != "/nova/v2.1/servers/server/migrations" || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "compute 2.80" {
			t.Fatal(r.Method, r.URL, r.Header)
		}
		return nativeServerWire(203, body)
	})
	seq := api.ListServerMigrationRecords(context.Background(), "server", migrationVersion())
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	var rows []*servers.ServerMigrationRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 2 || len(calls) != 1 {
		t.Fatal(rows, calls)
	}
	for k, v := range map[string]string{"id": `"uuid-only"`, "uuid": `"uuid-only"`, "status": "false", "created_at": `"literal-date"`, "memory_total_bytes": `"large"`, "disk_total_bytes": "9007199254740993"} {
		serverMigrationField(t, rows[0], k, v)
	}
	serverMigrationField(t, rows[1], "id", "null")
	if rows[0].Resource.Body["x_unknown"] != nil || rows[0].Resource.Body["server_id"] != nil || string(rows[0].Wire.Body["server_id"]) != `"forged"` || string(rows[0].Envelope) != body || rows[0].Header.Get("X-Server-Proof") != "actual" || rows[0].StatusCode != 203 {
		t.Fatal(rows)
	}
}

func TestPythonServerMigrationListFollowsCompletePagesAndHonorsEarlyBreak(t *testing.T) {
	for _, early := range []bool{false, true} {
		t.Run(map[bool]string{false: "all pages", true: "early break"}[early], func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeServerWire(200, `{"migrations":[{"id":"first"}],"next":"?marker=first"}`)
				}
				if r.URL.Query().Get("marker") != "first" {
					t.Fatal(r.URL)
				}
				return nativeServerWire(200, `{"migrations":[{"id":"last"}]}`)
			})
			if early {
				for row, err := range api.ListServerMigrationRecords(context.Background(), "server", migrationVersion()) {
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
					break
				}
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				return
			}
			rows, err := api.AllServerMigrationRecords(context.Background(), "server", migrationVersion())
			if err != nil || len(rows) != 2 || len(calls) != 2 {
				t.Fatal(rows, err, calls)
			}
			serverMigrationField(t, rows[1], "id", `"last"`)
		})
	}
}

func TestPythonServerMigrationGetOwnsParentURIAndPassiveResponseID(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "default ignore missing", true: "explicit false"}[strict], func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != "GET" || r.URL.EscapedPath() != "/nova/v2.1/servers/server%2Fid/migrations/migration%2Fid" || r.URL.RawQuery != "" {
					t.Fatal(r.Method, r.URL)
				}
				return nativeServerWire(200, `{"migration":{"id":null,"uuid":"fallback","status":7}}`)
			})
			opts := []servers.ServerMigrationOption{migrationVersion()}
			if strict {
				opts = append(opts, servers.WithServerMigrationIgnoreMissing(false))
			}
			row, err := api.GetServerMigrationRecord(context.Background(), "server/id", "migration/id", opts...)
			if err != nil || row == nil || len(calls) != 1 {
				t.Fatal(row, err, calls)
			}
			serverMigrationField(t, row, "id", "null")
			serverMigrationField(t, row, "status", "7")
		})
	}
}

func TestPythonServerMigrationGet404IsStrictForBothMissingFlags(t *testing.T) {
	for _, ignore := range []bool{true, false} {
		t.Run(map[bool]string{true: "true", false: "false"}[ignore], func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				return nativeServerWire(404, `{"error":"actual missing"}`)
			})
			_, err := api.GetServerMigrationRecord(context.Background(), "server", "migration", migrationVersion(), servers.WithServerMigrationIgnoreMissing(ignore))
			var native gophercloud.ErrUnexpectedResponseCode
			if len(calls) != 1 || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"actual missing"}` {
				t.Fatal(err, native, calls)
			}
		})
	}
}

func TestPythonServerMigrationAbortMissingDefaultAndStrictPolicies(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "ignored missing", true: "strict missing"}[strict], func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != "DELETE" || r.URL.Path != "/nova/v2.1/servers/server/migrations/migration" || r.URL.RawQuery != "" {
					t.Fatal(r.Method, r.URL)
				}
				return nativeServerWire(404, `{"error":"abort missing"}`)
			})
			opts := []servers.ServerMigrationOption{migrationVersion()}
			if strict {
				opts = append(opts, servers.WithServerMigrationIgnoreMissing(false))
			}
			result, err := api.AbortServerMigration(context.Background(), "server", "migration", opts...)
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			if strict {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal(result, err, native)
				}
				return
			}
			if err != nil || result == nil || !result.IgnoredMissing || result.Completed || result.Response == nil || result.Response.StatusCode != 404 || string(result.Response.Envelope) != `{"error":"abort missing"}` {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonServerMigrationActionsOwnLiteralForceCompleteAndReceipts(t *testing.T) {
	for _, op := range []string{"abort", "force"} {
		t.Run(op, func(t *testing.T) {
			var calls []nativeServerCall
			body := "not JSON: action receipt"
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				call := calls[len(calls)-1]
				if op == "force" {
					if call.method != "POST" || call.path != "/nova/v2.1/servers/server/migrations/migration/action" || call.body != `{"force_complete":null}` {
						t.Fatal(call)
					}
				} else if call.method != "DELETE" || call.body != "" {
					t.Fatal(call)
				}
				return nativeServerWire(202, body)
			})
			var result *servers.ServerMigrationActionResult
			var err error
			if op == "force" {
				result, err = api.ForceCompleteServerMigration(context.Background(), "server", "migration", migrationVersion())
			} else {
				result, err = api.AbortServerMigration(context.Background(), "server", "migration", migrationVersion())
			}
			if err != nil || result == nil || !result.Completed || result.IgnoredMissing || result.Response == nil || len(calls) != 1 {
				t.Fatal(result, err, calls)
			}
			row := result.Response
			if row.Resource != nil || row.Wire != nil || row.StatusCode != 202 || row.Header.Get("X-Server-Proof") != "actual" || string(row.Envelope) != body {
				t.Fatal(row)
			}
		})
	}
}

func TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls(t *testing.T) {
	headers := map[string]string{"X-Owned": "original"}
	version := "2.80"
	ignore := false
	option := servers.WithServerMigrationOpts(servers.ServerMigrationOpts{Headers: headers, Microversion: &version, IgnoreMissing: &ignore})
	headers["X-Owned"] = "changed"
	version = "2.1"
	ignore = true
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "compute 2.80" || r.URL.RawQuery != "" {
			t.Fatal(r.URL, r.Header)
		}
		return nativeServerWire(200, `{"migration":{"id":"actual"}}`)
	})
	for i := 0; i < 2; i++ {
		row, err := api.GetServerMigrationRecord(context.Background(), "server", "migration", option)
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	t.Run("discovery receipts owned per yielded row", func(t *testing.T) {
		var observed []nativeServerCall
		discoveryBody := ""
		discovered, _ := serverFindAPI(t, &observed, func(r *http.Request) *http.Response {
			if r.URL.Path == "/nova/v2.1/servers/server/migrations" {
				return nativeServerWire(200, `{"migrations":[{"id":"one"},{"id":"two"}]}`)
			}
			discoveryBody = `{"version":{"id":"v2.1","min_version":"2.1","version":"2.95","links":[{"rel":"self","href":"http://` + r.Host + `/nova/v2.1/"}]}}`
			return nativeServerWire(200, discoveryBody)
		})
		rows := 0
		for row, err := range discovered.ListServerMigrationRecords(context.Background(), "server", servers.WithServerMigrationMicroversion("")) {
			if err != nil || row == nil || len(row.Discovery.Responses) != 1 || row.Discovery.Responses[0] == nil {
				t.Fatal(row, err)
			}
			proof := row.Discovery.Responses[0]
			if string(proof.Body) != discoveryBody || proof.Header.Get("X-Server-Proof") != "actual" {
				t.Fatal(row.Discovery)
			}
			rows++
			if rows == 1 {
				proof.Body[0] = '['
				proof.Header.Set("X-Server-Proof", "caller")
				row.Discovery.Responses[0] = nil
			}
		}
		if rows != 2 || len(observed) != 2 {
			t.Fatal(rows, observed)
		}
	})

}

func TestPythonServerMigrationPreflightAndOptionGuardsStopRequests(t *testing.T) {
	for _, mode := range []string{"nil option", "callback error", "API changed", "cancel", "invalid server", "invalid migration", "forbidden header"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativeServerCall
			later := 0
			cause := errors.New("migration preflight")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, cloud := serverFindAPI(t, &calls, func(r *http.Request) *http.Response { return nativeServerWire(200, `{}`) })
			var option servers.ServerMigrationOption
			if mode != "nil option" {
				option = func(*servers.ServerMigrationOpts) error {
					switch mode {
					case "callback error":
						return cause
					case "API changed":
						*api = *servers.New(cloud.Client("compute", "/replacement/"))
					case "cancel":
						cancel(cause)
					}
					return nil
				}
			}
			if mode == "forbidden header" {
				option = servers.WithServerMigrationHeaders(map[string]string{"X-Auth-Token": "foreign"})
			}
			serverID, migrationID := "server", "migration"
			if mode == "invalid server" {
				serverID = ".."
			}
			if mode == "invalid migration" {
				migrationID = ".."
			}
			_, err := api.GetServerMigrationRecord(ctx, serverID, migrationID, option, func(*servers.ServerMigrationOpts) error { later++; return nil })
			if err == nil || len(calls) != 0 {
				t.Fatal(err, calls)
			}
			if mode == "nil option" || mode == "callback error" || mode == "API changed" || mode == "cancel" {
				if later != 0 {
					t.Fatal(later)
				}
			}
			if mode == "callback error" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay(t *testing.T) {
	for _, op := range []string{"get", "abort", "force"} {
		for _, mode := range []string{"read", "close", "cancel"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				var calls []nativeServerCall
				cause := errors.New("migration physical " + mode)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				fault := &serverFindFaultBody{data: []byte(`{"error":"physical"}`)}
				if mode == "read" {
					fault.readErr = cause
				}
				if mode == "close" {
					fault.closeErr = cause
				}
				if mode == "cancel" {
					fault.onClose = func() { cancel(cause) }
				}
				api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
					response := nativeServerWire(404, `{}`)
					response.Body = fault
					return response
				})
				var err error
				switch op {
				case "get":
					_, err = api.GetServerMigrationRecord(ctx, "server", "migration", migrationVersion())
				case "abort":
					_, err = api.AbortServerMigration(ctx, "server", "migration", migrationVersion())
				case "force":
					_, err = api.ForceCompleteServerMigration(ctx, "server", "migration", migrationVersion())
				}
				if !errors.Is(err, cause) || len(calls) != 1 || fault.closed != 1 {
					t.Fatal(err, calls, fault.closed)
				}
			})
		}
	}
}

func TestPythonServerMigrationListMalformedAcceptedResponseKeepsActualProof(t *testing.T) {
	var calls []nativeServerCall
	body := `{"migrations":[7]}`
	api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response { return nativeServerWire(203, body) })
	rows, err := api.AllServerMigrationRecords(context.Background(), "server", migrationVersion())
	var proof *resource.ResponseError
	if rows != nil || len(calls) != 1 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Server-Proof") != "actual" {
		t.Fatal(rows, err, proof, calls)
	}
}
