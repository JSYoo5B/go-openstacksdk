package neutronfind

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"slices"
	"testing"
)

func TestSourceVersionHeaderPolicyIsolatesServiceRoles(t *testing.T) {
	for _, role := range []string{"compute", "volume", "network", "identity", "baremetal"} {
		for _, version := range []string{"", "3.100"} {
			t.Run(role+"/"+version, func(t *testing.T) {
				e := &Engine{profile: Profile{ServiceRole: role, SourceFindPolicy: true}}
				source := &cloudread.Source{}
				source.Client.Microversion = version
				policy := e.headerPolicy(source, false, nil)
				for header, owner := range map[string]string{"X-OpenStack-Nova-API-Version": "compute", "X-OpenStack-Volume-API-Version": "volume", "X-OpenStack-Ironic-API-Version": "baremetal"} {
					if version != "" && owner == role {
						if policy.Values[header] != version || slices.Contains(policy.Absent, header) {
							t.Fatal(header, policy)
						}
					} else if !slices.Contains(policy.Absent, header) {
						t.Fatal(header, policy)
					}
				}
			})
		}
	}
}
