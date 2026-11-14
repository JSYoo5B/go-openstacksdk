package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"log"
	"os"
	"time"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
)

func Example_securityGroupList() {
	cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
	name := flag.String("name", "", "optional server name filter")
	limit := flag.Int("limit", -1, "page limit; -1 omits it")
	cap := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
	paginated := flag.Bool("paginated", true, "follow all retained pages")
	flag.Parse()
	if err := runSecurityGroupList(*cloud, *name, *limit, *cap, *paginated); err != nil {
		log.Fatal(err)
	}
}

func runSecurityGroupList(cloud, name string, limit, maxItems int, paginated bool) error {
	if limit < -1 {
		return fmt.Errorf("-limit must be -1 or nonnegative")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return err
	}
	options := []groups.RecordListOption{
		request.MaxItems(maxItems),
		request.Paginated(paginated),
		request.Headers(map[string]string{"X-Request-Source": "security-group-list-example"}),
	}
	if name != "" {
		options = append(options, groups.WithRecordListQuery("name", name))
	}
	if limit >= 0 {
		options = append(options, request.Limit(limit))
	}
	out := json.NewEncoder(os.Stdout)
	for record, readErr := range network.SecurityGroups.ListRecords(ctx, options...) {
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
