package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordCreateTaskPrepared(t *testing.T, ctx context.Context, client *gophercloud.ServiceClient) *preparedImageRecord {
	t.Helper()
	p, err := New(client).prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		return nil, check(opctx)
	})
	th.AssertNoErr(t, err)
	return p
}

func imageRecordCreateTaskSeed(t *testing.T, raw string) *ImageRecordCreateTask {
	t.Helper()
	var fields map[string]json.RawMessage
	th.AssertNoErr(t, json.Unmarshal([]byte(raw), &fields))
	body, err := normalizeImageRecordCreateTask(fields, json.RawMessage(raw))
	th.AssertNoErr(t, err)
	value, err := projectImageRecordCreateTask(body, json.RawMessage(`{"cloud":"seed"}`), nil)
	th.AssertNoErr(t, err)
	value.Envelope, value.Header, value.StatusCode = json.RawMessage(raw), http.Header{"X-Task-Proof": {"seed"}}, 201
	value.Resource.Header, value.Resource.StatusCode = value.Header.Clone(), value.StatusCode
	return value
}

func TestImageRecordCreateTaskNoWaitPreservesRawMissingAndNonstringDescriptors(t *testing.T) {
	for _, test := range []struct{ name, response string }{
		{"missing all", `{}`}, {"missing id", `{"status":"pending"}`}, {"missing status", `{"id":"t"}`},
		{"null all", `{"id":null,"status":null,"input":null,"result":null,"type":null}`},
		{"falsey all", `{"id":0,"status":false,"input":[],"result":"","type":0}`},
		{"raw all", `{"id":{"k":1},"status":["success"],"input":"opaque","result":900719925474099312345,"type":true}`},
		{"unknown status", `{"id":"t","status":"future"}`}, {"no active gate", `{"status":"failure","result":[1]}`},
		{"empty syntax", ``}, {"invalid syntax", `{"id":`}, {"opaque syntax", `opaque`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.TestMethod(t, req, http.MethodPost)
				if req.URL.EscapedPath() != "/reverse/glance/v2/tasks" || req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				imageRecordImportPayload(t, req, `{"type":null,"input":["raw",900719925474099312345]}`)
				return taskCoreJSON(req, 299, test.response), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			attrs := map[string]json.RawMessage{"type": json.RawMessage("null"), "input": json.RawMessage(`["raw",900719925474099312345]`)}
			got, receipt, err := createImageRecordTask(p, attrs)
			if err != nil || got == nil || receipt == nil || calls != 1 || got.StatusCode != 299 || len(got.Resource.Body) != 13 || string(receipt.Body) != test.response {
				t.Fatal(got, receipt, err, calls)
			}
			if !json.Valid([]byte(test.response)) && (string(got.Resource.Body["input"]) != string(attrs["input"]) || string(got.Resource.Body["status"]) != "null") {
				t.Fatal(got.Resource.Body)
			}
			if _, present := got.Resource.Body["tags"]; present {
				t.Fatal("Task has no tags descriptor")
			}
			got.Resource.Body["input"][0] = '!'
			if string(got.body["input"])[0] == '!' || string(attrs["input"])[0] == '!' {
				t.Fatal("raw descriptor aliases input")
			}
		})
	}
}

func TestImageRecordCreateTaskConstructorAndSparseTranslationUseTaskDescriptors(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		th.TestHeader(t, req, "X-Auth-Token", "first-token")
		if calls == 1 {
			th.TestHeader(t, req, "Accept", "application/json")
			th.TestHeader(t, req, "Content-Type", "application/json")
			imageRecordImportPayload(t, req, `{"id":["seed"],"name":false,"created_at":7,"expires_at":{},"input":null,"message":[],"owner":"alias wins","result":true,"schema":1e9999,"status":"pending","type":{},"updated_at":"literal"}`)
			return taskCoreJSON(req, 203, `{"id":"actual","owner":"first","owner_id":"last","result":[1,null],"self":"foreign","unknown":900719925474099312345,"location":{"cloud":"wire"},"tags":["ignored"]}`), nil
		}
		if calls != 2 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/tasks/actual" {
			t.Fatal(calls, req.Method, req.URL)
		}
		return taskCoreJSON(req, 202, `{"status":"success","input":"new raw","owner":null}`), nil
	})
	p := imageRecordCreateTaskPrepared(t, context.Background(), client)
	p.location = json.RawMessage(`{"cloud":"captured"}`)
	attrs := map[string]json.RawMessage{"id": json.RawMessage(`["seed"]`), "name": json.RawMessage("false"), "created_at": json.RawMessage("7"), "expires_at": json.RawMessage("{}"), "input": json.RawMessage("null"), "message": json.RawMessage("[]"), "owner": json.RawMessage(`"wire first"`), "owner_id": json.RawMessage(`"alias wins"`), "result": json.RawMessage("true"), "schema": json.RawMessage("1e9999"), "status": json.RawMessage(`"pending"`), "type": json.RawMessage("{}"), "updated_at": json.RawMessage(`"literal"`), "unknown": json.RawMessage(`[false]`), "location": json.RawMessage(`{"cloud":"constructor"}`), "tags": json.RawMessage(`["discard"]`)}
	created, createReceipt, err := createImageRecordTask(p, attrs)
	th.AssertNoErr(t, err)
	if created == nil || createReceipt == nil || string(created.Resource.Body["owner_id"]) != `"last"` || string(created.Resource.Body["location"]) != `{"cloud":"captured"}` || len(created.body) != 12 {
		t.Fatal(created)
	}
	got, getReceipt, err := getCreatedImageRecordTask(p, created)
	th.AssertNoErr(t, err)
	if got == nil || getReceipt == nil || got.StatusCode != 202 || string(got.Resource.Body["result"]) != `[1,null]` || string(got.Resource.Body["owner_id"]) != "null" || string(got.Resource.Body["status"]) != `"success"` || string(got.Resource.Body["schema"]) != "1e9999" {
		t.Fatal(got)
	}
	for _, key := range []string{"unknown", "self", "tags", "owner"} {
		if _, present := got.Resource.Body[key]; present {
			t.Fatal("unexpected descriptor", key)
		}
	}
	createReceipt.Body[0], getReceipt.Body[0], got.Envelope[0] = '!', '!', '!'
	got.Resource.Body["result"][1] = '9'
	got.Resource.Header.Set("X-Task-Proof", "view edited")
	got.Header.Set("X-Task-Proof", "receipt edited")
	if string(created.Resource.Body["result"]) != `[1,null]` || string(got.body["result"]) != `[1,null]` || string(created.Envelope)[0] != '{' || got.Resource.Header.Get("X-Task-Proof") != "view edited" || getReceipt.Header.Get("X-Task-Proof") != "actual" {
		t.Fatal("receipt/view/raw channels alias")
	}
}

func TestImageRecordCreateTaskConstructorHookAndControlBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, hook string
		invalid    bool
	}{
		{"object override", `{"input":[1],"type":null,"owner_id":false}`, false},
		{"false hook", `false`, false}, {"null hook", `null`, false}, {"empty list hook", `[]`, false},
		{"truthy list has no items", `[["type","x"]]`, true}, {"truthy string", `"x"`, true},
		{"inner source control", `{"connection":null}`, true}, {"inner self", `{"self":"x"}`, true},
		{"inner fixed target remains ignored extension", `{"base_path":"/foreign","resource_type":"future"}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				body := taskCorePayload(t, req)
				if _, ok := body["__conflicting_attrs"]; ok {
					t.Fatal(body)
				}
				if test.name == "object override" && (string(body["type"]) != "null" || string(body["input"]) != "[1]" || string(body["owner"]) != "false") {
					t.Fatal(body)
				}
				return taskCoreJSON(req, 201, `{}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			got, receipt, err := createImageRecordTask(p, map[string]json.RawMessage{"type": json.RawMessage(`"import"`), "input": json.RawMessage("{}"), "__conflicting_attrs": json.RawMessage(test.hook)})
			if test.invalid {
				if got != nil || receipt != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(got, receipt, err, calls)
				}
			} else if err != nil || got == nil || receipt == nil || calls != 1 {
				t.Fatal(got, receipt, err, calls)
			}
		})
	}
	for _, key := range []string{"self", "connection", "_synchronized", "microversion", "base_path", "resource_type"} {
		t.Run("top "+key, func(t *testing.T) {
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("unsafe constructor sent HTTP"); return nil, nil })
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			_, receipt, err := createImageRecordTask(p, map[string]json.RawMessage{key: json.RawMessage("null")})
			if receipt != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(receipt, err)
			}
		})
	}
}

func TestImageRecordCreateTaskGetStringifiesRawIDsOnlyWhenFetching(t *testing.T) {
	for _, test := range []struct {
		name, raw, identity string
		invalid             bool
	}{
		{"literal slash", `"a/b %?#한"`, "a/b %?#한", false}, {"integer", `900719925474099312345`, "900719925474099312345", false},
		{"bool", `true`, "True", false}, {"list", `[1,"x"]`, "[1, 'x']", false}, {"object", `{"k":null}`, "{'k': None}", false},
		{"null", `null`, "", true}, {"false", `false`, "", true}, {"zero", `0`, "", true}, {"empty string", `""`, "", true}, {"empty list", `[]`, "", true},
		{"dot", `".."`, "", true}, {"control", `"a\n"`, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/tasks/"+url.PathEscape(test.identity) || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL, test.identity)
				}
				return taskCoreJSON(req, 299, `{"status":"success"}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			seed := imageRecordCreateTaskSeed(t, `{"id":`+test.raw+`,"status":"pending","input":false}`)
			// Public descriptor edits cannot rewrite the internally captured GET ID.
			seed.Resource.Body["id"] = json.RawMessage(`"decoy"`)
			got, receipt, err := getCreatedImageRecordTask(p, seed)
			if test.invalid {
				if got != nil || receipt != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(got, receipt, err, calls)
				}
			} else if got == nil || receipt == nil || err != nil || calls != 1 || string(got.Resource.Body["id"]) != test.raw {
				t.Fatal(got, receipt, err, calls)
			}
		})
	}
}

func TestImageRecordCreateTaskAcceptedFaultsRetainOnlyActualReceipts(t *testing.T) {
	for _, operation := range []string{"create", "get"} {
		for _, fault := range []string{"read", "close", "valid nonobject", "source changed while read"} {
			t.Run(operation+" "+fault, func(t *testing.T) {
				cause := errors.New("physical task cause")
				calls := 0
				var client *gophercloud.ServiceClient
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					reader := &taskCoreReader{body: `{"id":"actual","status":"success"}`}
					body := &taskCoreBody{reader: reader}
					switch fault {
					case "read":
						reader.err = cause
					case "close":
						body.closeErr = cause
					case "valid nonobject":
						reader.body = `["array"]`
					case "source changed while read":
						reader.action = func() { client.Endpoint = "https://foreign.test/v2/" }
					}
					return taskCoreHTTP(req, 203, body), nil
				})
				p := imageRecordCreateTaskPrepared(t, context.Background(), client)
				var got *ImageRecordCreateTask
				var receipt *ImageUploadResponse
				var err error
				if operation == "create" {
					got, receipt, err = createImageRecordTask(p, map[string]json.RawMessage{"type": json.RawMessage(`"import"`), "input": json.RawMessage("[1]")})
				} else {
					got, receipt, err = getCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":"pending","input":[1]}`))
				}
				if got == nil || receipt == nil || got.StatusCode != 203 || receipt.StatusCode != 203 || err == nil || calls != 1 || string(got.Resource.Body["input"]) != "[1]" {
					t.Fatal(got, receipt, err, calls)
				}
				var proof *resource.ResponseError
				var failed *ImageRecordCreateTaskFailureError
				if !errors.As(err, &proof) || errors.As(err, &failed) || string(proof.Body) != string(receipt.Body) || proof.Header.Get("X-Task-Proof") != "actual" {
					t.Fatal(err, proof)
				}
				if fault == "read" || fault == "close" {
					if !errors.Is(err, cause) {
						t.Fatal(err)
					}
				}
				if string(got.Resource.Body["status"]) == `"success"` {
					t.Fatal("handling failure translated response", got.Resource.Body)
				}
			})
		}
	}
}

func TestImageRecordCreateTaskNativeRejectionsPreserveStatusWithoutResourceFailure(t *testing.T) {
	for _, code := range []int{400, 401, 404, 409, 503, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				return taskCoreJSON(req, code, `{"error":"native"}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			result, err := waitCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":"pending"}`), json.RawMessage("null"))
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			var failed *ImageRecordCreateTaskFailureError
			if result == nil || result.Task == nil || result.Fetches != 0 || result.Recreations != 0 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"error":"native"}` || errors.As(err, &proof) || errors.As(err, &failed) {
				t.Fatal(result, err, native)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitSeedSuccessPrecedesIDAndTimeoutValidation(t *testing.T) {
	for _, test := range []struct{ name, raw, timeout string }{
		{"missing id", `{"status":"SuCcEsS","result":null}`, `0`}, {"raw id", `{"id":[],"status":"success","result":[1]}`, `"bad timeout"`},
		{"null id", `{"id":null,"status":"SUCCESS"}`, `-1`}, {"raw result", `{"id":false,"status":"success","result":900719925474099312345}`, `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("seeded success sent GET"); return nil, nil })
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			seed := imageRecordCreateTaskSeed(t, test.raw)
			got, err := waitCreatedImageRecordTask(p, seed, json.RawMessage(test.timeout))
			if err != nil || got == nil || got.Task == seed || got.Fetches != 0 || got.Recreations != 0 || !reflect.DeepEqual(got.Task, seed) {
				t.Fatal(got, err)
			}
			got.Task.Resource.Body["status"][1] = '!'
			got.Task.Header.Set("X-Task-Proof", "changed")
			if string(seed.Resource.Body["status"])[1] == '!' || seed.Header.Get("X-Task-Proof") != "seed" {
				t.Fatal("seeded success aliases")
			}
		})
	}
}

func TestImageRecordCreateTaskWaitTimeoutDomainAndFixedBudget(t *testing.T) {
	for _, test := range []struct {
		name, raw                     string
		seconds                       time.Duration
		immediate, invalid, unlimited bool
	}{
		{"default", "", 3600 * time.Second, false, false, false}, {"null unlimited", "null", 0, false, false, true},
		{"fraction", "0.5", 500 * time.Millisecond, false, false, false}, {"bool true", "true", time.Second, false, false, false},
		{"exponent fraction", "5e-1", 500 * time.Millisecond, false, false, false},
		{"integer", "12", 12 * time.Second, false, false, false}, {"zero", "0", 0, true, false, false}, {"negative", "-0.5", 0, true, false, false},
		{"bool false", "false", 0, true, false, false}, {"numeric string", `"12"`, 0, false, true, false}, {"list", `[]`, 0, false, true, false},
		{"float underflow", "1e-9999", 0, true, false, false},
		{"object", `{}`, 0, false, true, false}, {"invalid JSON", `1 2`, 0, false, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				deadline, present := req.Context().Deadline()
				if test.unlimited {
					if present {
						t.Fatal("null timeout added a deadline")
					}
				} else {
					remaining := time.Until(deadline)
					if !present || remaining > test.seconds || remaining < test.seconds-200*time.Millisecond {
						t.Fatal(deadline, remaining, test.seconds)
					}
				}
				return taskCoreJSON(req, 200, `{"status":"success"}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			var raw json.RawMessage
			if test.raw != "" {
				raw = json.RawMessage(test.raw)
			}
			got, err := waitCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":"pending"}`), raw)
			if got == nil || got.Task == nil {
				t.Fatal(got, err)
			}
			if test.immediate {
				if !errors.Is(err, context.DeadlineExceeded) || calls != 0 || got.Fetches != 0 {
					t.Fatal(got, err, calls)
				}
			} else if test.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(got, err, calls)
				}
			} else if err != nil || calls != 1 || got.Fetches != 1 || len(got.FetchResponses) != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitInitialFailureAlwaysFetchesBeforeFailureDecision(t *testing.T) {
	for _, state := range []string{"failure", "pending", "future", "FAILURE"} {
		t.Run(state, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, 203, `{"status":"SuCcEsS","result":"passive"}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			got, err := waitCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":"`+state+`","message":"failure"}`), json.RawMessage("null"))
			if got == nil || err != nil || calls != 1 || got.Fetches != 1 || got.Recreations != 0 || string(got.Task.Resource.Body["result"]) != `"passive"` {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitOnlyActualNon396FailureTriggersDiagnosticClass(t *testing.T) {
	for _, message := range []string{`"ordinary"`, `null`, `false`, `[]`, `"Image cannot be imported. Error code: '396' "`} {
		t.Run(message, func(t *testing.T) {
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				return taskCoreJSON(req, 203, `{"id":"fresh","status":"FaIlUrE","message":`+message+`}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			got, err := waitCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"original","status":"pending"}`), json.RawMessage("null"))
			var failed *ImageRecordCreateTaskFailureError
			if got == nil || got.Fetches != 1 || got.Recreations != 0 || !errors.As(err, &failed) || !errors.Is(err, resource.ErrFailedState) || failed.ID != "original" || failed.Status != "FaIlUrE" || string(failed.Task.Resource.Body["id"]) != `"fresh"` {
				t.Fatal(got, err, failed)
			}
			taskCoreProof(t, err, 203, string(got.Task.Envelope))
			failed.Task.Resource.Body["id"][1] = '!'
			if string(got.Task.body["id"]) != `"fresh"` {
				t.Fatal("failure evidence aliases latest Task")
			}
		})
	}
}

func TestImageRecordCreateTaskWaitExact396RecreatesRawInputsAndGetsRecreatedSuccess(t *testing.T) {
	for _, test := range []struct{ name, input, kind string }{
		{"object and string", `{"k":[1]}`, `"import"`}, {"nulls", `null`, `null`}, {"raw nonobjects", `[false,1]`, `true`},
		{"empty raw", `""`, `0`}, {"huge number", `900719925474099312345`, `{"type":"raw"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls, pauses := 0, 0
			var firstDeadline time.Time
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				deadline, ok := req.Context().Deadline()
				if !ok {
					t.Fatal("missing budget")
				}
				if calls == 1 {
					firstDeadline = deadline
				} else if !deadline.Equal(firstDeadline) {
					t.Fatal("recreation restarted budget", deadline, firstDeadline)
				}
				switch calls {
				case 1:
					if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/tasks/original" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 203, `{"status":"failure","message":"Image cannot be imported. Error code: '396'","input":`+test.input+`,"type":`+test.kind+`}`), nil
				case 2:
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/tasks" {
						t.Fatal(req.Method, req.URL)
					}
					body := taskCorePayload(t, req)
					if len(body) != 2 || string(body["input"]) != test.input || string(body["type"]) != test.kind {
						t.Fatal(body)
					}
					return taskCoreJSON(req, 202, `{"id":"recreated","status":"success","result":{"image_id":"cached"}}`), nil
				case 3:
					if pauses != 1 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/tasks/recreated" {
						t.Fatal(pauses, req.Method, req.URL)
					}
					return taskCoreJSON(req, 299, `{"status":"success","result":{"image_id":"fresh"}}`), nil
				default:
					t.Fatal("extra request", calls)
					return nil, nil
				}
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			got, err := waitCreatedImageRecordTaskWithPause(p, imageRecordCreateTaskSeed(t, `{"id":"original","status":"pending"}`), json.RawMessage("30"), func(ctx context.Context) error { pauses++; return ctx.Err() })
			if got == nil || err != nil || calls != 3 || pauses != 1 || got.Fetches != 2 || got.Recreations != 1 || len(got.FetchResponses) != 2 || len(got.RecreationResponses) != 1 || string(got.Task.Resource.Body["result"]) != `{"image_id":"fresh"}` {
				t.Fatal(got, err, calls, pauses)
			}
			got.FetchResponses[1].Body[0], got.RecreationResponses[0].Body[0] = '!', '!'
			if string(got.Task.Envelope)[0] != '{' {
				t.Fatal("wait receipts alias latest Task")
			}
		})
	}
}

func TestImageRecordCreateTaskWaitMissingRecreationFieldsBecomeExplicitNull(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return taskCoreJSON(req, 200, `{"status":"failure","message":"Image cannot be imported. Error code: '396'"}`), nil
		}
		if calls == 2 {
			imageRecordImportPayload(t, req, `{"input":null,"type":null}`)
			return taskCoreJSON(req, 201, `{"id":"new","status":"pending"}`), nil
		}
		if calls != 3 {
			t.Fatal(calls)
		}
		return taskCoreJSON(req, 200, `{"status":"success"}`), nil
	})
	p := imageRecordCreateTaskPrepared(t, context.Background(), client)
	got, err := waitCreatedImageRecordTaskWithPause(p, imageRecordCreateTaskSeed(t, `{"id":"original","status":"pending"}`), json.RawMessage("null"), func(ctx context.Context) error { return ctx.Err() })
	if got == nil || err != nil || got.Recreations != 1 || got.Fetches != 2 || calls != 3 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateTaskWaitRecreationFailureAndPausePreserveLatestEvidence(t *testing.T) {
	for _, mode := range []string{"accepted read", "valid nonobject", "native rejection", "canceled pause", "source guard pause"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("recreation reader")
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"status":"failure","message":"Image cannot be imported. Error code: '396'","input":null,"type":false}`), nil
				}
				if calls != 2 {
					t.Fatal("GET after recreation/pause failure", calls)
				}
				switch mode {
				case "accepted read":
					return taskCoreHTTP(req, 202, &taskCoreBody{reader: &taskCoreReader{body: `{"id":"new"}`, err: cause}}), nil
				case "valid nonobject":
					return taskCoreJSON(req, 202, `[]`), nil
				case "native rejection":
					return taskCoreJSON(req, 503, `{"error":"post"}`), nil
				default:
					return taskCoreJSON(req, 202, `{"id":"new","status":"success"}`), nil
				}
			})
			p := imageRecordCreateTaskPrepared(t, ctx, client)
			got, err := waitCreatedImageRecordTaskWithPause(p, imageRecordCreateTaskSeed(t, `{"id":"original","status":"pending"}`), json.RawMessage("null"), func(ctx context.Context) error {
				if mode == "canceled pause" {
					cancel()
					return ctx.Err()
				}
				if mode == "source guard pause" {
					client.Endpoint = "https://foreign.test/"
					return nil
				}
				t.Fatal("unexpected pause", mode)
				return nil
			})
			if got == nil || err == nil || calls != 2 || got.Fetches != 1 {
				t.Fatal(got, err, calls)
			}
			var failed *ImageRecordCreateTaskFailureError
			if errors.As(err, &failed) {
				t.Fatal("non ResourceFailure requested diagnostic", err)
			}
			if mode == "native rejection" {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				if got.Recreations != 0 || !errors.As(err, &native) || native.Actual != 503 || errors.As(err, &proof) {
					t.Fatal(got, err)
				}
			} else {
				if got.Recreations != 1 || len(got.RecreationResponses) != 1 || got.Task.StatusCode != 202 {
					t.Fatal(got, err)
				}
				taskCoreProof(t, err, 202, string(got.RecreationResponses[0].Body))
			}
			if mode == "accepted read" && !errors.Is(err, cause) || mode == "canceled pause" && !errors.Is(err, context.Canceled) || mode == "source guard pause" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(mode, err)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitRawStatusFailuresDoNotTriggerTaskFailureDiagnostics(t *testing.T) {
	for _, mode := range []string{"initial null", "initial number", "fetched null", "fetched list", "fetched missing overlays pending"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				switch mode {
				case "fetched null":
					return taskCoreJSON(req, 203, `{"status":null}`), nil
				case "fetched list":
					return taskCoreJSON(req, 203, `{"status":["success"]}`), nil
				case "fetched missing overlays pending":
					return taskCoreJSON(req, 203, `{"input":true}`), nil
				default:
					t.Fatal("initial invalid status sent HTTP")
					return nil, nil
				}
			})
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			status := `"pending"`
			if mode == "initial null" {
				status = "null"
			} else if mode == "initial number" {
				status = "1"
			}
			pauseCause := errors.New("stop after missing overlay")
			got, err := waitCreatedImageRecordTaskWithPause(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":`+status+`}`), json.RawMessage("null"), func(context.Context) error { return pauseCause })
			var failed *ImageRecordCreateTaskFailureError
			if got == nil || err == nil || errors.As(err, &failed) {
				t.Fatal(got, err)
			}
			if strings.HasPrefix(mode, "initial") {
				if calls != 0 || got.Fetches != 0 || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err, calls)
				}
			} else if calls != 1 || got.Fetches != 1 {
				t.Fatal(got, err, calls)
			}
			if mode == "fetched missing overlays pending" && (!errors.Is(err, pauseCause) || string(got.Task.Resource.Body["status"]) != `"pending"`) {
				t.Fatal(got, err)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitDefaultPauseHonorsParentAndFractionalBudget(t *testing.T) {
	for _, mode := range []string{"fractional timeout", "parent canceled", "parent deadline and null budget"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if mode == "parent deadline and null budget" {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
			}
			defer cancel()
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					t.Fatal("default interval did not pause", calls)
				}
				if mode == "parent canceled" {
					return taskCoreHTTP(req, 203, &taskCoreBody{reader: &taskCoreReader{body: `{"status":"pending"}`, action: cancel}}), nil
				}
				return taskCoreJSON(req, 203, `{"status":"pending"}`), nil
			})
			p := imageRecordCreateTaskPrepared(t, ctx, client)
			timeout := json.RawMessage("null")
			if mode == "fractional timeout" {
				timeout = json.RawMessage("0.02")
			}
			start := time.Now()
			got, err := waitCreatedImageRecordTask(p, imageRecordCreateTaskSeed(t, `{"id":"seed","status":"pending"}`), timeout)
			if got == nil || got.Fetches != 1 || calls != 1 || time.Since(start) > time.Second {
				t.Fatal(got, err, calls)
			}
			if mode == "parent canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordCreateTaskNativeRetryFreezesRawSubmissionAndRetainsLiveAuthentication(t *testing.T) {
	for _, mode := range []string{"ordinary retry", "body changed", "endpoint changed", "auth changed"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				imageRecordImportPayload(t, req, `{"input":[1],"type":false}`)
				if calls == 1 {
					return taskCoreJSON(req, 503, `{"error":"retry"}`), nil
				}
				if mode == "auth changed" {
					th.TestHeader(t, req, "X-Auth-Token", "updated-token")
				}
				return taskCoreJSON(req, 201, `{"id":"actual"}`), nil
			})
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if retries != 1 {
					return original
				}
				switch mode {
				case "body changed":
					opts.JSONBody = json.RawMessage(`{"input":[],"type":true}`)
				case "endpoint changed":
					client.Endpoint = "https://foreign.test/v2/"
				case "auth changed":
					client.ProviderClient.SetToken("updated-token")
				case "ordinary retry":
					opts.MoreHeaders["X-Retry"] = "ordinary"
				}
				return nil
			}
			p := imageRecordCreateTaskPrepared(t, context.Background(), client)
			got, receipt, err := createImageRecordTask(p, map[string]json.RawMessage{"input": json.RawMessage("[1]"), "type": json.RawMessage("false")})
			if mode == "body changed" || mode == "endpoint changed" {
				if got != nil || receipt != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, receipt, err, calls)
				}
			} else if got == nil || receipt == nil || err != nil || calls != 2 || retries != 1 {
				t.Fatal(got, receipt, err, calls, retries)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitPreservesOriginalDiagnosticObjectAcrossRecreations(t *testing.T) {
	calls, pauses := 0, 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return taskCoreJSON(req, 203, `{"id":"first-overlay","status":"failure","message":"Image cannot be imported. Error code: '396'","input":true,"type":null}`), nil
		case 2:
			return taskCoreJSON(req, 201, `{"id":"replacement-1","status":"pending","input":[]}`), nil
		case 3:
			if req.URL.EscapedPath() != "/reverse/glance/v2/tasks/replacement-1" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 299, `{"id":"second-overlay","status":"failure","message":"Image cannot be imported. Error code: '396'","type":{}}`), nil
		case 4:
			imageRecordImportPayload(t, req, `{"input":[],"type":{}}`)
			return taskCoreJSON(req, 202, `{"id":"replacement-2","status":"pending"}`), nil
		case 5:
			if req.URL.EscapedPath() != "/reverse/glance/v2/tasks/replacement-2" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 203, `{"id":"final-overlay","status":"failure","message":null}`), nil
		default:
			t.Fatal("extra request", calls)
			return nil, nil
		}
	})
	p := imageRecordCreateTaskPrepared(t, context.Background(), client)
	seed := imageRecordCreateTaskSeed(t, `{"id":"original","status":"pending"}`)
	got, err := waitCreatedImageRecordTaskWithPause(p, seed, json.RawMessage("30"), func(ctx context.Context) error { pauses++; return ctx.Err() })
	var failed *ImageRecordCreateTaskFailureError
	if got == nil || !errors.As(err, &failed) || calls != 5 || pauses != 2 || got.Fetches != 3 || got.Recreations != 2 || string(got.OriginalTask.Resource.Body["id"]) != `"first-overlay"` || string(got.Task.Resource.Body["id"]) != `"final-overlay"` || failed.ID != "original" {
		t.Fatal(got, err, calls, pauses)
	}
	got.OriginalTask.Resource.Body["id"][1] = '!'
	if string(got.OriginalTask.body["id"]) != `"first-overlay"` || string(seed.Resource.Body["id"]) != `"original"` || string(got.Task.Resource.Body["id"]) != `"final-overlay"` {
		t.Fatal("diagnostic object aliases")
	}
}
