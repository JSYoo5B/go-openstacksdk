package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func Example_securityGroupRuleList() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	group := flag.String("group", "", "server security_group_id filter")
	direction := flag.String("direction", "", "server direction filter")
	name := flag.String("name", "", "local inherited name filter")
	pageSize := flag.Int("page-size", -1, "server limit; -1 omits it")
	maxItems := flag.Int("max-items", 0, "raw cap before local filtering")
	single := flag.Bool("single-page", false, "read first page only")
	flag.Parse()
	if err := runSecurityGroupRuleList(*cloud, *group, *direction, *name, *pageSize, *maxItems, *single); err != nil {
		log.Fatal(err)
	}
}

func runSecurityGroupRuleList(cloud, group, direction, name string, pageSize, maxItems int, single bool) error {
	if pageSize < -1 || maxItems < 0 {
		return fmt.Errorf("invalid page-size or max-items")
	}
	ctx := context.Background()
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	options := []rules.RecordListOption{
		request.MaxItems(maxItems),
		request.Paginated(!single),
	}
	if pageSize >= 0 {
		options = append(options, request.Limit(pageSize))
	}
	for _, filter := range []struct{ key, value string }{
		{"security_group_id", group}, {"direction", direction}, {"name", name},
	} {
		if filter.value != "" {
			options = append(options, rules.WithRecordListQuery(filter.key, filter.value))
		}
	}
	out := json.NewEncoder(os.Stdout)
	for record, readErr := range network.SecurityRules.ListRecords(ctx, options...) {
		if readErr != nil {
			return readErr
		}
		if err := out.Encode(map[string]any{
			"resource":    record.Resource.Body,
			"wire":        record.Wire.Body,
			"envelope":    record.Envelope,
			"headers":     record.Header,
			"status_code": record.StatusCode,
		}); err != nil {
			return err
		}
	}
	return nil
}
