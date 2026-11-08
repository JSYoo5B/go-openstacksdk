package objects

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type preparedObjectWait struct {
	request         *preparedCreateObject
	options         ObjectWaitOpts
	timeout         time.Duration
	failures        []string
	callback        func(int) error
	progress        chan<- int
	pacer           *pollschedule.Pacer
	header, literal string
	literalSelected bool
}

func validateObjectWaitHeaders(values map[string]string) (map[string]string, error) {
	headers, err := validateCreateObjectHeaders(values)
	if err != nil {
		return nil, err
	}
	for key := range headers {
		name := strings.ToLower(key)
		if name == "range" || strings.HasPrefix(name, "if-") {
			return nil, metadataInvalid("conditional object waiter header %q is not allowed", key)
		}
	}
	return headers, nil
}
func objectWaitText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func (a *API) prepareObjectWait(ctx context.Context, container, object, status string, deletion bool, options []ObjectWaitOption) (*preparedObjectWait, error) {
	p, err := a.captureCreateObject(ctx, container, object)
	if err != nil {
		return nil, err
	}
	p.headerPolicy = func(values map[string]string) error { _, err := validateObjectWaitHeaders(values); return err }
	if err = p.guard(ctx); err != nil {
		return nil, err
	}
	if !objectWaitText(status) {
		return nil, metadataInvalid("invalid object target status")
	}
	cfg, err := p.applyObjectWaitOptions(ctx, options)
	if err != nil {
		return nil, metadataContextError(ctx, err)
	}
	cfg.Headers, err = validateObjectWaitHeaders(cfg.Headers)
	defaults := resource.WaitDefaults{Unlimited: true, Strategy: resource.FixedWait(2 * time.Second), Supports: objectWaitSupport}
	if deletion {
		defaults = resource.WaitDefaults{Timeout: 120 * time.Second, Strategy: resource.FixedWait(2 * time.Second), Supports: objectWaitSupport}
	}
	var settings resource.WaitSettings
	if err == nil {
		settings, err = resource.ResolveWait(defaults, cfg.Wait...)
	}
	attribute := settings.StatusAttribute
	if err == nil && attribute != "" && (!metadataToken(attribute) || strings.ContainsAny(attribute, ". /\\")) {
		err = metadataInvalid("object status attribute must be a single field name")
	}
	if err == nil && cfg.StatusHeader != "" && !metadataToken(cfg.StatusHeader) {
		err = metadataInvalid("object status header must be a single HTTP token")
	}
	if err == nil && cfg.StatusHeader != "" {
		at := min(max(cfg.statusHeaderAt, 0), len(cfg.Wait))
		if objectWaitSelectsAttribute(cfg.Wait[at:]) {
			err = metadataInvalid("object status attribute and header conflict")
		}
		attribute = ""
	}
	if err == nil {
		for _, failure := range settings.FailureStates {
			if !objectWaitText(failure) {
				err = metadataInvalid("invalid object failure state")
				break
			}
		}
	}
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	w := &preparedObjectWait{request: p, options: cfg, timeout: settings.Timeout, callback: settings.ProgressCallback, progress: settings.Progress}
	w.pacer = pollschedule.NewPacer(settings.Strategy.Delay)
	if deletion {
		return w, nil
	}
	w.failures = []string{"ERROR"}
	if settings.FailureStatesSet {
		w.failures = append([]string{}, settings.FailureStates...)
	}
	if cfg.StatusHeader != "" {
		w.header = http.CanonicalHeaderKey(cfg.StatusHeader)
		return w, nil
	}
	if attribute == "" {
		attribute = "status"
	}
	switch attribute {
	case "name":
		w.literal, w.literalSelected = object, true
	case "container":
		w.literal, w.literalSelected = container, true
	default:
		w.header = objectWaitAttributeHeader(attribute)
		if w.header == "" {
			return nil, fmt.Errorf("%w: Swift object has no supported string attribute %q", resource.ErrUnsupported, attribute)
		}
	}
	return w, nil
}
func objectWaitAttributeHeader(attribute string) string {
	switch attribute {
	case "content_type":
		return "Content-Type"
	case "etag":
		return "ETag"
	case "content_encoding":
		return "Content-Encoding"
	case "content_disposition":
		return "Content-Disposition"
	case "manifest", "object_manifest":
		return "X-Object-Manifest"
	case "timestamp":
		return "X-Timestamp"
	case "last_modified_at", "updated_at":
		return "Last-Modified"
	case "delete_at":
		return "X-Delete-At"
	case "accept_ranges":
		return "Accept-Ranges"
	case "access_control_allow_origin":
		return "Access-Control-Allow-Origin"
	case "expires_at":
		return "Expires"
	case "signature":
		return "Signature"
	}
	return ""
}
func (w *preparedObjectWait) selected(headers http.Header) (*string, error) {
	if w.literalSelected {
		value := w.literal
		return &value, nil
	}
	value, err := objectCreateHeaderValue(headers, w.header)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: object response lacks selected status header %q", resource.ErrUnsupported, w.header)
	}
	return value, nil
}
func (w *preparedObjectWait) pause(ctx context.Context) error {
	if err := w.request.guard(ctx); err != nil {
		return err
	}
	timer := time.NewTimer(w.pacer.Next())
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
	return w.request.guard(ctx)
}
