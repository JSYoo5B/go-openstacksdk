package quotas_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonRejudgeQuotaReply struct {
	status int
	body   string
}

func pythonRejudgeQuotaScope(t *testing.T, replies ...pythonRejudgeQuotaReply) (*quotas.ProjectScope, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	var seen []string
	cloud.Mux.HandleFunc("/key-manager/v1/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = append(seen, r.Method+" "+r.URL.RequestURI()+" "+string(raw))
		reply := replies[len(seen)-1]
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, reply.body)
	})
	scope, err := quotas.New(cloud.Client("key-manager", "/key-manager/v1/")).InProject(context.Background(), resource.ID("project-1"))
	if err != nil {
		t.Fatal(err)
	}
	return scope, &seen
}

// Python get_project_quota fetches project-quotas/{project} and exposes the
// project_quotas values.
func TestPythonRejudgeKeyManagerProjectQuotaGetRequest(t *testing.T) {
	scope, seen := pythonRejudgeQuotaScope(t, pythonRejudgeQuotaReply{200, `{"project_quotas":{"secrets":10,"orders":20,"containers":-1,"consumers":5,"cas":0}}`})
	value, err := scope.Get(context.Background())
	if err != nil || string(value.Secrets) != "10" || string(value.Orders) != "20" || string(value.Containers) != "-1" || string(value.Consumers) != "5" || string(value.CAs) != "0" {
		t.Fatal(value, err)
	}
	if len(*seen) != 1 || (*seen)[0] != "GET /key-manager/v1/project-quotas/project-1 " {
		t.Fatalf("%q", *seen)
	}
}

// Python delete_project_quota defaults to ignore_missing=True: 204 and 404
// both return nothing, while ignore_missing=False surfaces the 404.
func TestPythonRejudgeKeyManagerProjectQuotaDeleteRequest(t *testing.T) {
	scope, seen := pythonRejudgeQuotaScope(t, pythonRejudgeQuotaReply{204, ""}, pythonRejudgeQuotaReply{404, `{"code":404}`}, pythonRejudgeQuotaReply{404, `{"code":404}`})
	if err := scope.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(false)); err == nil {
		t.Fatal("ignore_missing=False must surface 404")
	}
	want := "DELETE /key-manager/v1/project-quotas/project-1 "
	if len(*seen) != 3 || (*seen)[0] != want || (*seen)[1] != want || (*seen)[2] != want {
		t.Fatalf("%q", *seen)
	}
}
