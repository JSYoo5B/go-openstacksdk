package connectiontest_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness compares the
// typed clouds.yaml policy for use_external_network/use_internal_network with
// the pinned raw config.get(...) value whose Python truthiness decides
// _find_interesting_networks. Booleans, null, the empty string and any casing
// of "true" agree. Other strings are false in Go but truthy in Python. Numbers
// and collections are configuration errors in Go; Python's normalize_keys
// turns numbers into nonempty strings (truthy, even 0) and keeps an empty list
// falsy. These are the documented typed-policy differences.
func TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness(t *testing.T) {
	type outcome int
	const (
		enabled outcome = iota
		disabled
		rejected
	)
	for _, row := range []struct {
		yaml           string
		goResult       outcome
		pythonTruthy   bool
		documentedDiff bool
	}{
		{"true", enabled, true, false},
		{"false", disabled, false, false},
		{"null", disabled, false, false},
		{"''", disabled, false, false},
		{"'TRUE'", enabled, true, false},
		{"'false'", disabled, true, true},
		{"'yes'", disabled, true, true},
		{"0", rejected, true, true},
		{"1", rejected, true, true},
		{"[]", rejected, false, true},
	} {
		t.Run(row.yaml, func(t *testing.T) {
			if (row.goResult == enabled) != row.pythonTruthy && !row.documentedDiff {
				t.Fatal("undocumented difference")
			}
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			opts := defaultNetworkCloudOptions(t, cloud, "    use_external_network: "+row.yaml+"\n    use_internal_network: false\n    networks: [{name: private, nat_destination: true}]\n")
			conn, err := sdk.Connect(context.Background(), opts...)
			if row.goResult == rejected {
				if !errors.Is(err, resource.ErrInvalidOption) || conn != nil {
					t.Fatal(conn, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := conn.GetNetworkRoles(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := int32(0)
			if row.goResult == enabled {
				want = 1
			}
			if conn.UseExternalNetwork() != (row.goResult == enabled) || conn.UseInternalNetwork() || calls.Load() != want {
				t.Fatal(conn.UseExternalNetwork(), calls.Load())
			}
		})
	}
}
