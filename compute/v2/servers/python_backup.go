package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// BackupActionRequest owns the createBackup body. Empty name/type and zero or
// negative rotation are sent literally; Nova validates its domain constraints.
type BackupActionRequest struct {
	Name       string `json:"name"`
	BackupType string `json:"backup_type"`
	Rotation   int    `json:"rotation"`
}
type BackupActionOption = request.Option[BackupActionRequest]

func WithBackupActionOptions(value BackupActionRequest) BackupActionOption {
	return request.WithOptions(value)
}
func WithBackupActionField(key string, value any) BackupActionOption {
	return request.WithField[BackupActionRequest](key, value)
}
func WithBackupActionHeader(key, value string) BackupActionOption {
	return request.WithHeader[BackupActionRequest](key, value)
}

// BackupActionResult retains the accepted action separately from discovery.
// ImageID alone does not establish that Glance has observed an active image.
type BackupActionResult struct {
	ImageID      string
	Microversion string
	Discovery    microversions.Discovery
	Body         json.RawMessage
	Header       http.Header
	StatusCode   int
}

func backupImageID(response *rest.Response) (string, error) {
	// Source tries response.json() at every version, then falls back to the last
	// slash-delimited Location component when there is no image_id key.
	var fields map[string]json.RawMessage
	var id string
	found := false
	if utf8.Valid(response.Body) && json.Unmarshal(response.Body, &fields) == nil && len(fields) != 0 {
		if raw, present := fields["image_id"]; present {
			found = true
			if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '"' || json.Unmarshal(raw, &id) != nil {
				return "", userActionInvalid("createBackup image_id must be a JSON string")
			}
		}
	}
	if !found {
		location, present := response.Header[http.CanonicalHeaderKey("Location")]
		if !present || len(location) == 0 {
			return "", userActionInvalid("createBackup response requires image_id or Location")
		}
		value := location[len(location)-1]
		// Do not resolve or follow the Location; only the returned identifier is
		// consumed by the separately selected Glance service.
		index := strings.LastIndex(value, "/")
		if index < 0 {
			return "", userActionInvalid("createBackup Location must contain a slash")
		}
		id = value[index+1:]
	}
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return "", userActionInvalid("createBackup image ID must be nonblank literal UTF-8 text")
	}
	return id, nil
}

// CreateBackupAction implements Server.backup, including operation-owned
// microversion discovery/selection and body-first/Location image ID extraction.
// Use compute.Service.BackupServer for the complete Proxy lookup/wait workflow.
func (a *API) CreateBackupAction(ctx context.Context, serverID string, input BackupActionRequest, options ...BackupActionOption) (*BackupActionResult, error) {
	const op = "CreateBackupAction"
	p, err := a.prepareUserAction(ctx, serverID)
	if err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	c, err := applyUserActionOptions(p, input, func(v BackupActionRequest) BackupActionRequest { return v }, true, options)
	if err == nil && (!utf8.ValidString(c.Options.Name) || !utf8.ValidString(c.Options.BackupType)) {
		err = userActionInvalid("backup name and type must be valid UTF-8")
	}
	if err == nil {
		for key := range c.Headers {
			if strings.EqualFold(key, "Accept") {
				err = userActionInvalid("createBackup owns the empty Accept header")
				break
			}
		}
	}
	body, bodyErr := request.MergeFieldsFor(map[string]any{"createBackup": map[string]any{"name": c.Options.Name, "backup_type": c.Options.BackupType, "rotation": c.Options.Rotation}}, c.Fields, c.Options)
	err = errors.Join(err, bodyErr, p.guard(ctx))
	if err != nil {
		return nil, userActionWrap(ctx, op, err)
	}
	result := &BackupActionResult{}
	result.Discovery, err = microversions.Read(ctx, p.source, microversions.NovaProfile)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return result, userActionWrap(ctx, op, err)
	}
	selected := p.source.Client.Microversion
	supports, err := microversions.Supports(result.Discovery.Maximum, result.Discovery.Minimum, selected, "2.45")
	if err == nil {
		if supports {
			selected = "2.45"
		} else if selected == "" {
			selected, err = microversions.SelectVersion(result.Discovery.Maximum, result.Discovery.Minimum, "2.100")
		}
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return result, userActionWrap(ctx, op, err)
	}
	result.Microversion = selected
	client := p.source.Client
	client.Microversion = selected
	client.MoreHeaders = maps.Clone(client.MoreHeaders)
	for key := range client.MoreHeaders {
		if strings.EqualFold(key, "Accept") || strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(client.MoreHeaders, key)
		}
	}
	headers := rest.RequestHeaderPolicy{Values: map[string]string{"Accept": ""}}
	if selected == "" {
		headers.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Nova-API-Version"}
	} else {
		headers.Values["OpenStack-API-Version"] = "compute " + selected
		headers.Values["X-OpenStack-Nova-API-Version"] = selected
	}
	accepted, rejected := make([]int, 200), make([]int, 200)
	for i := range accepted {
		accepted[i] = 200 + i
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &client, p.guard, http.MethodPost, client.ServiceURL("servers", url.PathEscape(serverID), "action"), body, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true}, accepted...)
	if response != nil {
		result.Body = bytes.Clone(response.Body)
		result.Header = response.Header.Clone()
		result.StatusCode = response.StatusCode
	}
	if err != nil {
		return result, userActionWrap(ctx, op, err)
	}
	result.ImageID, err = backupImageID(response)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return result, userActionWrap(ctx, op, response.Fail(err))
	}
	return result, nil
}
