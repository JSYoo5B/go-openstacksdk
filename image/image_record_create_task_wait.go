package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const imageRecordCreateTaskError396 = "Image cannot be imported. Error code: '396'"
const imageRecordCreateTaskInterval = 2 * time.Second

// The Source timeout is arithmetic, not a float()/truthiness conversion:
// numeric strings and containers fail, bool is numeric, null means forever.
// Positive values beyond Go's timer range remain bounded by caller context.
func imageRecordCreateTaskBudget(raw json.RawMessage) (time.Duration, bool, error) {
	if raw == nil {
		return 3600 * time.Second, false, nil
	}
	if !json.Valid(raw) {
		return 0, false, uploadInvalid("task timeout must be complete JSON")
	}
	text := string(bytes.TrimSpace(raw))
	switch text {
	case "null":
		return 0, false, nil
	case "true":
		return time.Second, false, nil
	case "false":
		return 0, true, nil
	}
	if len(text) == 0 || text[0] != '-' && (text[0] < '0' || text[0] > '9') {
		return 0, false, uploadInvalid("task timeout must be a JSON number, bool, or null")
	}
	var number *big.Rat
	if strings.ContainsAny(text, ".eE") {
		// Python JSON loads a fractional/exponent number into binary64. Preserve
		// its rounding and positive infinity without constructing an enormous
		// arbitrary-precision exponent merely to set a timer.
		value, err := strconv.ParseFloat(text, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return 0, false, uploadInvalid("task timeout number cannot be represented")
		}
		if value <= 0 {
			return 0, true, nil
		}
		if math.IsInf(value, 1) {
			return time.Duration(1<<63 - 1), false, nil
		}
		number = new(big.Rat).SetFloat64(value)
	} else {
		var ok bool
		number, ok = new(big.Rat).SetString(text)
		if !ok {
			return 0, false, uploadInvalid("task timeout number cannot be represented")
		}
	}
	if number.Sign() <= 0 {
		return 0, true, nil
	}
	nanoseconds := new(big.Rat).Mul(number, big.NewRat(int64(time.Second), 1))
	whole := new(big.Int).Quo(nanoseconds.Num(), nanoseconds.Denom())
	if !whole.IsInt64() {
		return time.Duration(1<<63 - 1), false, nil
	}
	duration := time.Duration(whole.Int64())
	if duration == 0 {
		duration = time.Nanosecond
	}
	return duration, false, nil
}

func imageRecordCreateTaskPause(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(imageRecordCreateTaskInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

func imageRecordCreateTaskStatus(task *ImageRecordCreateTask) (string, error) {
	return decodeImageRecordString(imageRecordCreateTaskRaw(task, "status"), "task wait status")
}

func imageRecordCreateTaskWaitError(result *ImageRecordCreateTaskWaitResult, err error) (*ImageRecordCreateTaskWaitResult, error) {
	// A later context/source/status failure retains the latest actual accepted
	// evidence. HTTP rejections and accepted physical errors already carry their
	// own newer receipt and must not be masked by an older Task's response.
	var accepted *resource.ResponseError
	if result.Task != nil && result.Task.StatusCode != 0 && !errors.As(err, &accepted) {
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) {
			err = (&rest.Response{Body: bytes.Clone(result.Task.Envelope), Header: result.Task.Header.Clone(), StatusCode: result.Task.StatusCode}).Fail(err)
		}
	}
	return result, err
}

func waitCreatedImageRecordTask(p *preparedImageRecord, task *ImageRecordCreateTask, timeout json.RawMessage) (*ImageRecordCreateTaskWaitResult, error) {
	return waitCreatedImageRecordTaskWithPause(p, task, timeout, imageRecordCreateTaskPause)
}

// A private pause argument gives focused tests deterministic polling boundaries
// without exporting a clock/builder policy or changing the Source interval.
func waitCreatedImageRecordTaskWithPause(p *preparedImageRecord, task *ImageRecordCreateTask, timeout json.RawMessage, pause func(context.Context) error) (*ImageRecordCreateTaskWaitResult, error) {
	result := &ImageRecordCreateTaskWaitResult{Task: cloneImageRecordCreateTask(task), OriginalTask: cloneImageRecordCreateTask(task)}
	if err := p.check(p.ctx); err != nil {
		return imageRecordCreateTaskWaitError(result, err)
	}
	if _, err := imageRecordCreateTaskBody(result.Task); err != nil {
		return result, err
	}
	initial, err := imageRecordCreateTaskStatus(result.Task)
	if err != nil {
		return imageRecordCreateTaskWaitError(result, err)
	}
	// Source checks seeded success before touching timeout or an HTTP identity.
	if cloudfilter.PythonLower(initial) == "success" {
		if err := p.check(p.ctx); err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		return result, nil
	}
	budget, expired, err := imageRecordCreateTaskBudget(bytes.Clone(timeout))
	if err != nil {
		return imageRecordCreateTaskWaitError(result, err)
	}
	if expired {
		return imageRecordCreateTaskWaitError(result, context.DeadlineExceeded)
	}
	ctx, cancel := context.WithCancel(p.ctx)
	if budget > 0 {
		cancel()
		ctx, cancel = context.WithTimeout(p.ctx, budget)
	}
	defer cancel()
	owned := *p
	owned.ctx = ctx
	originalID, err := cloudfilter.PythonString(imageRecordCreateTaskRaw(result.Task, "id"))
	if err != nil {
		return imageRecordCreateTaskWaitError(result, err)
	}
	originalObject := true
	for {
		if err := owned.check(ctx); err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		fresh, receipt, err := getCreatedImageRecordTask(&owned, result.Task)
		if receipt != nil {
			result.Fetches++
			result.FetchResponses = append(result.FetchResponses, receipt)
		}
		if fresh != nil {
			result.Task = fresh
			if originalObject {
				result.OriginalTask = cloneImageRecordCreateTask(fresh)
			}
		}
		if err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		state, err := imageRecordCreateTaskStatus(result.Task)
		if err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		switch cloudfilter.PythonLower(state) {
		case "success":
			if err := owned.check(ctx); err != nil {
				return imageRecordCreateTaskWaitError(result, err)
			}
			return result, nil
		case "failure":
			message := imageRecordCreateTaskRaw(result.Task, "message")
			var exact string
			// A nonstring/null message simply fails equality with Source's string.
			if json.Unmarshal(message, &exact) == nil && exact == imageRecordCreateTaskError396 {
				attrs := map[string]json.RawMessage{"input": imageRecordCreateTaskRaw(result.Task, "input"), "type": imageRecordCreateTaskRaw(result.Task, "type")}
				recreated, receipt, err := createImageRecordTask(&owned, attrs)
				if receipt != nil {
					result.Recreations++
					result.RecreationResponses = append(result.RecreationResponses, receipt)
				}
				if recreated != nil {
					result.Task = recreated
				}
				if err != nil {
					return imageRecordCreateTaskWaitError(result, err)
				}
				originalObject = false
			} else {
				return imageRecordCreateTaskWaitError(result, &ImageRecordCreateTaskFailureError{Task: cloneImageRecordCreateTask(result.Task), ID: originalID, Status: state})
			}
		}
		// Even a recreated success response is followed by this pause and fresh
		// GET; recreation never restarts the original timeout budget.
		if err := owned.check(ctx); err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		if err := pause(ctx); err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
		if err := owned.check(ctx); err != nil {
			return imageRecordCreateTaskWaitError(result, err)
		}
	}
}
