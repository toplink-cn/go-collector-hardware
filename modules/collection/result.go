package collection

import (
	"errors"
)

const (
	Success     = "success"
	Partial     = "partial"
	Failed      = "failed"
	Unsupported = "unsupported"
)

var ErrUnsupported = errors.New("metric unsupported")

type Quality struct {
	CollectedAt int64  `json:"collected_at,omitempty"`
	State       string `json:"state"`
	Reason      string `json:"reason,omitempty"`
}

type Result[T any] struct {
	Data    T
	Quality map[string]Quality
}

func FromError(err error, hasData bool) Quality {
	if err == nil {
		return Quality{State: Success}
	}
	state := Failed
	if hasData {
		state = Partial
	} else if errors.Is(err, ErrUnsupported) {
		state = Unsupported
	}
	return Quality{State: state, Reason: err.Error()}
}
