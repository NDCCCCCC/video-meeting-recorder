// Package response exposes the logging helper SentinelField which mirrors
// internal/errors.FirstKnownSentinelName. The function is placed in pkg/response
// (per user decision D-03.1) so handler/service call-sites can import it
// without introducing a new package boundary.
//
// R-6 (research §9) — return type is zap.Field (not a string) so future work
// on typed error kinds can swap zap.String for zap.Object without changing
// call-sites.
package response

import (
	"fmt"

	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// sentinelTypeKey is the structured-logging key emitted alongside err when
// handlers/services call zap.Error(err), response.SentinelField(err).
const sentinelTypeKey = "sentinel_type"

// SentinelField returns a zap.Field with key "sentinel_type" describing the
// recognition state of err.
//
// 4-state contract (CONTEXT.md D-03.4/D-03.5):
//
//	err == nil              → zap.Skip() (no field emitted)
//	*BusinessError          → zap.String("sentinel_type", "BusinessError(code=XXX)")
//	sentinel hit (IsKnown)  → zap.String("sentinel_type", "ErrXxx")
//	unknown error           → zap.String("sentinel_type", "ad-hoc")
//
// Priority delegates to apperrors.FirstKnownSentinelName, so the order
// matches internal/errors.IsKnownError (D-03.3).
func SentinelField(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	var be *apperrors.BusinessError
	if e := apperrors.As(err, &be); e {
		return zap.String(sentinelTypeKey, fmt.Sprintf("BusinessError(code=%s)", be.Code))
	}
	if name, ok := apperrors.FirstKnownSentinelName(err); ok {
		return zap.String(sentinelTypeKey, name)
	}
	return zap.String(sentinelTypeKey, "ad-hoc")
}
