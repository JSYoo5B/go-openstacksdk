package cloudvendors

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"regexp"
	"testing"
)

// TestEmbeddedFilesMatchREADMEHashes keeps the unmodified pinned copies and
// their documented provenance hashes in sync.
func TestEmbeddedFilesMatchREADMEHashes(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := regexp.MustCompile("(?m)^\\| `([^`]+)` \\| `([0-9a-f]{64})` \\|$").FindAllStringSubmatch(string(readme), -1)
	entries, err := files.ReadDir("data")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(entries) {
		t.Fatal(len(rows), len(entries))
	}
	for _, row := range rows {
		data, err := files.ReadFile("data/" + row[1])
		if err != nil {
			t.Fatal(err)
		}
		if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != row[2] {
			t.Fatal(row[1])
		}
	}
}

func TestProfilesResolveStatusNestedAndRemoteNames(t *testing.T) {
	if names := Names(); len(names) != 25 || names[0] != "auro" {
		t.Fatal(names)
	}
	rackspace, found, err := Profile("rackspace")
	if err != nil || !found || rackspace["has_network"] != false || rackspace["floating_ip_source"] != "None" || rackspace["secgroup_source"] != "None" {
		t.Fatal(rackspace, found, err)
	}
	rackspace["has_network"] = true
	if again, _, _ := Profile("rackspace"); again["has_network"] != false {
		t.Fatal("profile copy is not owned")
	}
	limestone, found, err := Profile("limestonenetworks")
	if err != nil || !found || limestone["regions"] == nil {
		t.Fatal(limestone, found, err)
	}
	// Deprecated profiles load without status or message keys.
	deprecated, found, err := Profile("fuga")
	if err != nil || !found || deprecated["status"] != nil || deprecated["message"] != nil {
		t.Fatal(deprecated, found, err)
	}
	if _, found, err := Profile("betacloud"); !found || !errors.Is(err, ErrShutdown) {
		t.Fatal(found, err)
	}
	// The ovh profile nests a remote URL that Python would fetch.
	if _, _, err := Profile("ovh"); !errors.Is(err, ErrRemoteProfile) {
		t.Fatal(err)
	}
	if _, _, err := Profile("https://example.invalid"); !errors.Is(err, ErrRemoteProfile) {
		t.Fatal(err)
	}
	if profile, found, err := Profile("absent"); profile != nil || found || err != nil {
		t.Fatal(profile, found, err)
	}
}
