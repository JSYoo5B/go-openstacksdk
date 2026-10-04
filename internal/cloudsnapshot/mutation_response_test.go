package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func TestSnapshotMutationObjectDistinguishesToleranceFromActualEmptyAndRejectsWrongShape(t *testing.T) {
	for _, body := range []json.RawMessage{nil, json.RawMessage(""), json.RawMessage(" \n\t"), json.RawMessage("{"), json.RawMessage("not-json"), json.RawMessage("{} {}"), json.RawMessage(`{"snapshot":`)} {
		wire := &rest.Response{Body: body, Header: http.Header{"X-Actual": []string{"malformed"}}, StatusCode: http.StatusAccepted}
		raw, actual, err := mutationObject(wire)
		if err != nil || raw != nil || actual != nil {
			t.Fatalf("tolerated body manufactured an actual object: %q %s %#v %v", body, raw, actual, err)
		}
		proof := mutationProof(wire)
		if proof == nil || proof.StatusCode != http.StatusAccepted || !bytes.Equal(proof.Body, body) || proof.Header.Get("X-Actual") != "malformed" {
			t.Fatalf("tolerance discarded actual response evidence: %#v", proof)
		}
	}
	for _, body := range []string{"{}", `{"snapshot":{}}`} {
		wire := &rest.Response{Body: json.RawMessage(body), Header: http.Header{"X-Actual": []string{"parsed-empty"}}, StatusCode: http.StatusNoContent}
		raw, actual, err := mutationObject(wire)
		if err != nil || actual == nil || actual.Body == nil || len(actual.Body) != 0 || string(raw) != "{}" || actual.StatusCode != http.StatusNoContent {
			t.Fatalf("parsed empty lost actual-object identity: %s %s %#v %v", body, raw, actual, err)
		}
		if actual.Header.Get("X-Actual") != "parsed-empty" {
			t.Fatal("actual empty object lost metadata")
		}
	}
	wrong := []json.RawMessage{json.RawMessage("null"), json.RawMessage("false"), json.RawMessage("7"), json.RawMessage(`"text"`), json.RawMessage("[]"), json.RawMessage("[{}]"), json.RawMessage(`{"snapshot":null}`), json.RawMessage(`{"snapshot":false}`), json.RawMessage(`{"snapshot":7}`), json.RawMessage(`{"snapshot":"text"}`), json.RawMessage(`{"snapshot":[]}`), json.RawMessage{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}}
	for _, body := range wrong {
		t.Run(string(body), func(t *testing.T) {
			wire := &rest.Response{Body: json.RawMessage(bytes.Clone(body)), Header: http.Header{"X-Actual": []string{"accepted-bad-object"}}, StatusCode: http.StatusAccepted}
			raw, actual, err := mutationObject(wire)
			var physical *resource.ResponseError
			if raw != nil || actual != nil || !errors.As(err, &physical) || physical.Cause == nil {
				t.Fatalf("accepted invalid object lost phase failure: %s %#v %v", raw, actual, err)
			}
			if physical.StatusCode != http.StatusAccepted || !bytes.Equal(physical.Body, body) || physical.Header.Get("X-Actual") != "accepted-bad-object" {
				t.Fatalf("decode failure borrowed/changed response evidence: %#v", physical)
			}
			for i := range wire.Body {
				wire.Body[i] = ' '
			}
			wire.Header["X-Actual"][0] = "mutated"
			if !bytes.Equal(physical.Body, body) || physical.Header.Get("X-Actual") != "accepted-bad-object" {
				t.Fatal("accepted failure evidence aliases its physical input")
			}
		})
	}
}

func TestSnapshotMutationObjectOwnsActualFieldsAndMetadataSeparatelyFromLogicalView(t *testing.T) {
	body := json.RawMessage(`{"snapshot":{"id":0,"force":"FALSE","size":"٧","metadata":"opaque","unknown":{"n":900719925474099312345678901},"location":"wire","self":"wire-self","connection":true},"id":"outer-is-not-a-member"}`)
	before := json.RawMessage(bytes.Clone(body))
	wire := &rest.Response{Body: body, Header: http.Header{"X-Actual": []string{"one", "two"}}, StatusCode: http.StatusCreated}
	raw, actual, err := mutationObject(wire)
	if err != nil || actual == nil {
		t.Fatalf("physical object: %s %#v %v", raw, actual, err)
	}
	rawBefore := json.RawMessage(bytes.Clone(raw))
	if actual.StatusCode != http.StatusCreated || len(actual.Body) != 8 || !bytes.Equal(body, before) {
		t.Fatalf("actual member fields/status changed: %#v", actual)
	}
	snapshotMutationModelRaw(t, actual.Body["id"], "0")
	snapshotMutationModelRaw(t, actual.Body["force"], `"FALSE"`)
	snapshotMutationModelRaw(t, actual.Body["size"], `"٧"`)
	snapshotMutationModelRaw(t, actual.Body["metadata"], `"opaque"`)
	snapshotMutationModelRaw(t, actual.Body["unknown"], `{"n":900719925474099312345678901}`)
	_, state := snapshotMutationModelCreateBody(t, "seed volume", CreateOptions{Attributes: CreateAttributes{Name: snapshotMutationModelString("seed name")}})
	if err := state.overlay(raw); err != nil {
		t.Fatal(err)
	}
	logical := snapshotMutationModelView(t, &state, resource.CloudLocation{})
	snapshotMutationModelRaw(t, logical["id"], "0")
	snapshotMutationModelRaw(t, logical["name"], `"seed name"`)
	snapshotMutationModelRaw(t, logical["volume_id"], `"seed volume"`)
	snapshotMutationModelRaw(t, logical["is_forced"], "false")
	snapshotMutationModelRaw(t, logical["size"], "7")
	snapshotMutationModelRaw(t, logical["metadata"], "{}")
	for _, key := range []string{"name", "volume_id", "is_forced"} {
		if _, present := actual.Body[key]; present {
			t.Fatalf("synthetic logical field %q entered actual RawResource", key)
		}
	}
	for i := range wire.Body {
		wire.Body[i] = ' '
	}
	wire.Header["X-Actual"][0] = "wire changed"
	if !bytes.Equal(raw, rawBefore) || actual.Header["X-Actual"][0] != "one" {
		t.Fatal("physical resource and selected bytes borrow wire storage")
	}
	for i := range raw {
		raw[i] = ' '
	}
	snapshotMutationModelRaw(t, actual.Body["unknown"], `{"n":900719925474099312345678901}`)
	for i := range actual.Body["force"] {
		actual.Body["force"][i] = ' '
	}
	actual.Body["id"] = json.RawMessage("null")
	actual.Header["X-Actual"][1] = "actual changed"
	logical = snapshotMutationModelView(t, &state, resource.CloudLocation{})
	snapshotMutationModelRaw(t, logical["id"], "0")
	snapshotMutationModelRaw(t, logical["is_forced"], "false")
	snapshotMutationModelRaw(t, state.id(), "0")

	flat := &rest.Response{Body: json.RawMessage(`{"id":"flat","snapshotish":{"id":"nested"},"status":null}`), StatusCode: http.StatusOK}
	raw, actual, err = mutationObject(flat)
	if err != nil || actual == nil || !bytes.Equal(raw, flat.Body) {
		t.Fatalf("flat actual member was incorrectly enveloped: %s %#v %v", raw, actual, err)
	}
	snapshotMutationModelRaw(t, actual.Body["id"], `"flat"`)
	snapshotMutationModelRaw(t, actual.Body["snapshotish"], `{"id":"nested"}`)
}

func TestSnapshotMutationProofKeepsOpaqueBytesAndEveryPhaseOwnsItsCopy(t *testing.T) {
	if mutationProof(nil) != nil || cloneMutationProof(nil) != nil {
		t.Fatal("nil physical phase invented proof")
	}
	for _, status := range []int{http.StatusAccepted, http.StatusNoContent, http.StatusNotFound} {
		opaque := json.RawMessage{0xff, 'o', 'p', 'a', 'q', 'u', 'e'}
		wire := &rest.Response{Body: opaque, Header: http.Header{"X-Phase": []string{"one", "two"}}, StatusCode: status}
		first := mutationProof(wire)
		second := cloneMutationProof(first)
		third := cloneMutationProof(first)
		if first == nil || second == nil || third == nil || first == second || second == third || first.StatusCode != status {
			t.Fatalf("phase proof not independently owned: %#v %#v %#v", first, second, third)
		}
		wire.Body[0] = 'x'
		wire.Header["X-Phase"][0] = "wire changed"
		wire.StatusCode = http.StatusInternalServerError
		if first.Body[0] != 0xff || first.Header["X-Phase"][0] != "one" || first.StatusCode != status {
			t.Fatal("opaque actual phase aliases its input")
		}
		first.Body[1] = 'x'
		first.Header["X-Phase"][1] = "first changed"
		first.StatusCode = http.StatusInternalServerError
		if second.Body[1] != 'o' || third.Body[1] != 'o' || second.Header["X-Phase"][1] != "two" || third.Header["X-Phase"][1] != "two" || second.StatusCode != status || third.StatusCode != status {
			t.Fatal("earlier phase mutation changed another phase")
		}
		second.Body[2] = 'x'
		second.Header["X-Phase"][0] = "second changed"
		if third.Body[2] != 'p' || third.Header["X-Phase"][0] != "one" {
			t.Fatal("result phase siblings share retained bytes/headers")
		}
	}
}
