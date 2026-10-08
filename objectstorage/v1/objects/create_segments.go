package objects

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type objectCreateManifestEntry struct {
	Path string  `json:"path"`
	Size int64   `json:"size_bytes"`
	ETag *string `json:"etag,omitempty"`
}

func (p *preparedCreateObject) planSegments(ctx context.Context, result *CreateObjectResult, source *objectCreateSource, size int64) ([]string, error) {
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	count := (source.size-1)/size + 1
	if count <= 0 || uint64(count) > uint64(^uint(0)>>1) {
		return nil, metadataInvalid("object segment count overflows local indexing")
	}
	var entropy [16]byte
	_, err := rand.Read(entropy[:])
	if err = joinMetadataErrors(err, p.note(ctx)); err != nil {
		return nil, err
	}
	result.SegmentPrefix = p.metadata.object + "/.go-openstacksdk-upload-" + hex.EncodeToString(entropy[:]) + "/"
	width := len(strconv.FormatInt(count-1, 10))
	if width < 6 {
		width = 6
	}
	segments, targets := make([]ObjectCreateSegmentResult, int(count)), make([]string, int(count))
	for index := int64(0); index < count; index++ {
		// index*size is bounded by the source length because index<count.
		offset := index * size
		length := size
		if source.size-offset < length {
			length = source.size - offset
		}
		name := result.SegmentPrefix + fmt.Sprintf("%0*d", width, index)
		target, err := p.target(name)
		if err != nil {
			return nil, err
		}
		segments[index] = ObjectCreateSegmentResult{Name: name, Index: index, Offset: offset, Size: length}
		targets[index] = target
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	result.Segments = segments
	return targets, nil
}

func appendObjectCreatePhase(dst **ObjectCreatePhaseResult, src *ObjectCreatePhaseResult) {
	if *dst == nil {
		*dst = &ObjectCreatePhaseResult{}
	}
	(*dst).Attempts = append((*dst).Attempts, src.Attempts...)
	if src.Acknowledgement != nil {
		(*dst).Acknowledgement = cloneObjectCreateResponse(src.Acknowledgement)
	}
}

func (p *preparedCreateObject) segmentRound(ctx context.Context, result *CreateObjectResult, source *objectCreateSource, targets []string, headers map[string]string, indexes []int, logical int, failures []error, eligible []bool) {
	jobs := make(chan int)
	var workers sync.WaitGroup
	count := 5
	if len(indexes) < count {
		count = len(indexes)
	}
	for worker := 0; worker < count; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				segment := &result.Segments[index]
				out := p.exchange(ctx, http.MethodPut, targets[index], "segment", &objectCreatePayload{source: source, offset: segment.Offset, size: segment.Size}, headers, logical, http.StatusCreated, http.StatusAccepted)
				appendObjectCreatePhase(&segment.Upload, out.phase)
				segment.Ambiguous = segment.Ambiguous || out.ambiguous
				response := out.phase.Acknowledgement
				if out.err == nil && response != nil {
					if response.StatusCode == http.StatusAccepted {
						segment.Ambiguous = true
						out.err = objectCreateResponseError(response, &ObjectCreateUnconfirmedSegmentError{Name: segment.Name})
					} else {
						etag, projectionErr := objectCreateHeaderValue(response.Header, "ETag")
						guardErr := p.guard(ctx)
						if guardErr == nil {
							segment.Created = true
						}
						if guardErr != nil {
							segment.Ambiguous = true
						}
						if err := joinMetadataErrors(projectionErr, guardErr); err != nil {
							out.err = objectCreateResponseError(response, err)
						} else if etag != nil {
							value := *etag
							if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
								value = value[1 : len(value)-1]
							}
							segment.ETag = &value
						}
					}
				}
				if out.err != nil && response != nil && len(segment.Upload.Attempts) != 0 {
					last := &segment.Upload.Attempts[len(segment.Upload.Attempts)-1]
					last.Error = joinMetadataErrors(last.Error, cloneObjectCreateError(out.err))
				}
				failures[index], eligible[index] = out.err, out.retry
			}
		}()
	}
	for _, index := range indexes {
		if p.guard(ctx) != nil {
			break
		}
		select {
		case jobs <- index:
		case <-ctx.Done():
		}
	}
	close(jobs)
	workers.Wait()
}

func (p *preparedCreateObject) segmented(ctx context.Context, result *CreateObjectResult, source *objectCreateSource, size int64, headers map[string]string) error {
	targets, err := p.planSegments(ctx, result, source, size)
	if err != nil {
		return err
	}
	failures, eligible := make([]error, len(targets)), make([]bool, len(targets))
	indexes := make([]int, len(targets))
	for index := range indexes {
		indexes[index] = index
	}
	p.segmentRound(ctx, result, source, targets, headers, indexes, 1, failures, eligible)
	retries := make([]int, 0)
	for index := range eligible {
		if failures[index] != nil && eligible[index] {
			retries = append(retries, index)
		}
	}
	if len(retries) != 0 && p.guard(ctx) == nil {
		p.segmentRound(ctx, result, source, targets, headers, retries, 2, failures, eligible)
	}
	err = joinMetadataErrors(failures...)
	err = joinMetadataErrors(err, p.guard(ctx))
	if err == nil {
		for index := range result.Segments {
			if result.Segments[index].Upload == nil || result.Segments[index].Upload.Acknowledgement == nil {
				err = metadataInvalid("object segment was not confirmed")
				break
			}
		}
	}
	if err == nil {
		err = p.publishManifest(ctx, result, headers)
	}
	if err != nil {
		err = joinMetadataErrors(err, p.cleanupSegments(ctx, result, targets, headers))
	}
	return err
}

func objectCreateDLOQuoted(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func (p *preparedCreateObject) publishManifest(ctx context.Context, result *CreateObjectResult, headers map[string]string) error {
	if err := p.guard(ctx); err != nil {
		return err
	}
	target := p.metadata.target
	var data []byte
	if result.Mode == "slo" {
		entries := make([]objectCreateManifestEntry, len(result.Segments))
		for index, segment := range result.Segments {
			entries[index] = objectCreateManifestEntry{Path: "/" + p.metadata.container + "/" + segment.Name, Size: segment.Size, ETag: segment.ETag}
		}
		var err error
		data, err = json.Marshal(entries)
		if err != nil {
			return err
		}
		parsed, _ := url.Parse(target)
		parsed.RawQuery = url.Values{"multipart-manifest": {"put"}}.Encode()
		target = parsed.String()
	} else {
		data = make([]byte, 0)
		headers = cloneMetadataHeaders(headers)
		headers["X-Object-Manifest"] = objectCreateDLOQuoted(p.metadata.container) + "/" + objectCreateDLOQuoted(result.SegmentPrefix)
	}
	source := &objectCreateSource{data: data, size: int64(len(data))}
	for logical := 1; logical <= 3; logical++ {
		out := p.exchange(ctx, http.MethodPut, target, result.Mode, &objectCreatePayload{source: source, size: source.size}, headers, logical, http.StatusCreated, http.StatusAccepted)
		appendObjectCreatePhase(&result.Manifest, out.phase)
		result.ManifestAmbiguous = result.ManifestAmbiguous || out.ambiguous
		if out.err == nil {
			return nil
		}
		if !out.retry || logical == 3 {
			return out.err
		}
	}
	return nil
}

func (p *preparedCreateObject) cleanupSegments(ctx context.Context, result *CreateObjectResult, targets []string, headers map[string]string) error {
	if result.ManifestAmbiguous || p.guard(ctx) != nil {
		return nil
	}
	// Cleanup carries ordinary upload headers, without logical object metadata
	// or the DLO publication header. Each exact name has its own evidence.
	cleanupHeaders := cloneMetadataHeaders(headers)
	for key := range cleanupHeaders {
		if strings.HasPrefix(strings.ToLower(key), "x-object-meta-") || strings.EqualFold(key, "X-Object-Manifest") {
			delete(cleanupHeaders, key)
		}
	}
	var failures []error
	for index, segment := range result.Segments {
		if !segment.Created || segment.Ambiguous {
			continue
		}
		if err := p.guard(ctx); err != nil {
			failures = append(failures, err)
			break
		}
		out := p.exchange(ctx, http.MethodDelete, targets[index], "cleanup", nil, cleanupHeaders, 1, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound)
		result.Cleanup = append(result.Cleanup, ObjectCreateCleanupResult{Name: segment.Name, Deletion: out.phase})
		if out.err != nil {
			failures = append(failures, out.err)
		}
		if err := p.guard(ctx); err != nil {
			failures = append(failures, err)
			break
		}
	}
	return joinMetadataErrors(failures...)
}
