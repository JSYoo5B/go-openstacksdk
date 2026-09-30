package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const pythonPin = "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe"
const gophercloudPin = "v2.15.0"

type sourcePins struct {
	Gophercloud string `json:"gophercloud"`
	Python      string `json:"openstacksdk"`
}

var pins = sourcePins{Gophercloud: gophercloudPin, Python: pythonPin}

type operationCatalog struct {
	Schema     int               `json:"schema_version"`
	Pins       sourcePins        `json:"source_pins"`
	Operations map[string]string `json:"operations"`
}

func loadInventory(root string) (operationCatalog, error) {
	result := operationCatalog{Schema: 1, Pins: pins, Operations: make(map[string]string)}
	add := func(id string, value map[string]any) error {
		if _, exists := result.Operations[id]; exists {
			return fmt.Errorf("duplicate inventory operation %q", id)
		}
		// Candidate Go packages, transport return policies and review markers are
		// SDK discovery data, not a Python/native source contract. Locations and
		// declared inputs are retained; the exact upstream revision pins the body.
		delete(value, "candidates")
		delete(value, "review")
		delete(value, "sdk_package")
		delete(value, "builder_free")
		delete(value, "return_policy")
		delete(value, "request_policy")
		delete(value, "issue")
		pin := pythonPin
		if strings.HasPrefix(id, "gophercloud:") {
			pin = gophercloudPin
		}
		fingerprint, err := sourceFingerprint(pin, value)
		if err != nil {
			return err
		}
		result.Operations[id] = fingerprint
		return nil
	}
	var native struct {
		Version    string           `json:"gophercloud_version"`
		Operations []map[string]any `json:"operations"`
	}
	if err := readJSON(filepath.Join(root, "api/gophercloud_inventory.json"), &native); err != nil {
		return result, err
	}
	if native.Version != gophercloudPin {
		return result, fmt.Errorf("Gophercloud inventory must be pinned to %s", gophercloudPin)
	}
	for _, op := range native.Operations {
		pkg, _ := op["package"].(string)
		name, _ := op["name"].(string)
		if pkg == "" || name == "" {
			return result, fmt.Errorf("native operation requires package and name")
		}
		if err := add("gophercloud:"+pkg+"."+name, op); err != nil {
			return result, err
		}
	}
	var manifest struct {
		Revision string `json:"revision"`
		Proxy    int    `json:"proxy_methods"`
		Cloud    int    `json:"cloud_workflows"`
		Connect  int    `json:"connection_methods"`
		Proxies  []struct {
			File    string `json:"file"`
			Methods int    `json:"methods"`
		} `json:"proxies"`
	}
	pythonRoot := filepath.Join(root, "api/openstacksdk")
	if err := readJSON(filepath.Join(pythonRoot, "manifest.json"), &manifest); err != nil {
		return result, err
	}
	if manifest.Revision != pythonPin {
		return result, fmt.Errorf("Python inventory must be pinned to %s", pythonPin)
	}
	proxyCount := 0
	files := map[string]bool{}
	loadPython := func(file, kind string, expected int) error {
		if files[file] {
			return fmt.Errorf("duplicate Python inventory file %q", file)
		}
		files[file] = true
		path, err := localFile(pythonRoot, file)
		if err != nil {
			return err
		}
		var inventory struct {
			Revision   string           `json:"revision"`
			Operations []map[string]any `json:"operations"`
		}
		if err := readJSON(path, &inventory); err != nil {
			return err
		}
		if inventory.Revision != pythonPin || len(inventory.Operations) != expected {
			return fmt.Errorf("Python inventory %s has inconsistent revision or method count", file)
		}
		for _, op := range inventory.Operations {
			id, _ := op["id"].(string)
			if id == "" || op["kind"] != kind {
				return fmt.Errorf("Python operation in %s requires id and kind %s", file, kind)
			}
			if err := add("python:"+id, op); err != nil {
				return err
			}
		}
		return nil
	}
	for _, proxy := range manifest.Proxies {
		if err := loadPython(proxy.File, "proxy", proxy.Methods); err != nil {
			return result, err
		}
		proxyCount += proxy.Methods
	}
	if proxyCount != manifest.Proxy {
		return result, fmt.Errorf("Python manifest proxy total does not match its groups")
	}
	if err := loadPython("cloud.json", "workflow", manifest.Cloud); err != nil {
		return result, err
	}
	if err := loadPython("connection.json", "connection", manifest.Connect); err != nil {
		return result, err
	}
	return result, nil
}

func sourceFingerprint(pin string, declaration map[string]any) (string, error) {
	data, err := json.Marshal(map[string]any{"source_pin": pin, "declaration": declaration})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// Reject duplicate JSON keys rather than accepting encoding/json's last value.
// Otherwise a duplicated review or catalog ID could hide contradictory evidence.
func readJSON(path string, target any) error {
	return readJSONPolicy(path, target, false)
}

func readStrictJSON(path string, target any) error {
	return readJSONPolicy(path, target, true)
}

func readJSONPolicy(path string, target any, strict bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value func() error
	value = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate or invalid JSON key %q", key)
				}
				seen[name] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delim)
		}
		_, err = decoder.Token()
		return err
	}
	if err := value(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("%s: trailing JSON content", path)
	}
	decoder = json.NewDecoder(bytes.NewReader(data))
	if strict {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func localFile(root, name string) (string, error) {
	if name == "" || !filepath.IsLocal(name) || strings.Contains(name, "\\") {
		return "", fmt.Errorf("invalid local evidence path %q", name)
	}
	path := filepath.Join(root, name)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("missing evidence file %q", name)
	}
	return path, nil
}
