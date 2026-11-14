package serviceinfo_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func Example_records() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
	storeID := flag.String("store", "", "optional local store ID filter")
	limit := flag.Int("limit", -1, "server limit; -1 omits the option")
	maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
	paginated := flag.Bool("paginated", true, "follow store list continuations")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := runRecords(ctx, *cloud, *storeID, *limit, *maxItems, *paginated); err != nil {
		var response *resource.ResponseError
		if errors.As(err, &response) {
			fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
		}
		log.Fatal(err)
	}
}

func printRecordRecords(view, wire *resource.RawResource, statusCode int) error {
	value := map[string]any{"resource": view, "wire": wire, "status_code": statusCode}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runRecords(ctx context.Context, cloud, storeID string, limit, maxItems int, paginated bool) error {
	if limit < -1 {
		return fmt.Errorf("limit must be -1 or nonnegative")
	}
	conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.ImageV2(ctx)
	if err != nil {
		return err
	}
	info, err := service.ServiceInfo.GetImportInfoRecord(ctx,
		request.Header("X-Request-Source", "serviceinfo-records-example"))
	if err != nil {
		return err
	}
	if err := printRecordRecords(info.Resource, info.Wire, info.StatusCode); err != nil {
		return err
	}

	options := []serviceinfo.StoreRecordListOption{
		serviceinfo.WithStoreRecordListOpts(serviceinfo.StoreRecordListOpts{MaxItems: maxItems}),
		request.Header("X-Request-Source", "serviceinfo-records-example"),
		request.Paginated(paginated),
	}
	if limit >= 0 {
		options = append(options, request.Limit(limit))
	}
	if storeID != "" {
		options = append(options, serviceinfo.WithStoreRecordListFilter("id", storeID))
	}
	for record, readErr := range service.ServiceInfo.ListStoreRecords(ctx, options...) {
		if readErr != nil {
			return readErr
		}
		if err := printRecordRecords(record.Resource, record.Wire, record.StatusCode); err != nil {
			return err
		}
	}
	return nil
}
