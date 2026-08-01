package response

import (
	"errors"
	"fmt"
	"testing"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestSentinelField covers the 4-state contract documented in
// CONTEXT.md D-03.4/D-03.5 and R-6 (zap.Field return type).
func TestSentinelField(t *testing.T) {
	t.Run("sentinel hit", func(t *testing.T) {
		field := SentinelField(apperrors.ErrTaskNotFound)
		assert.Equal(t, "sentinel_type", field.Key)
		assert.Equal(t, "ErrTaskNotFound", field.String)
	})

	t.Run("wrapped sentinel", func(t *testing.T) {
		field := SentinelField(fmt.Errorf("ctx: %w", apperrors.ErrUserDisabled))
		assert.Equal(t, "sentinel_type", field.Key)
		assert.Equal(t, "ErrUserDisabled", field.String)
	})

	t.Run("BusinessError", func(t *testing.T) {
		field := SentinelField(apperrors.NewBusinessError(apperrors.CodeNotFound, "missing", nil))
		assert.Equal(t, "sentinel_type", field.Key)
		assert.Equal(t, "BusinessError(code=NOT_FOUND)", field.String)
	})

	t.Run("unknown error", func(t *testing.T) {
		field := SentinelField(errors.New("random"))
		assert.Equal(t, "sentinel_type", field.Key)
		assert.Equal(t, "ad-hoc", field.String)
	})

	t.Run("nil", func(t *testing.T) {
		field := SentinelField(nil)
		assert.Equal(t, zapcore.SkipType, field.Type)
		assert.Equal(t, "", field.Key, "zap.Skip produces an empty Key")
	})

	t.Run("priority mirrors IsKnownError slice", func(t *testing.T) {
		// First hit wins — IsKnownError slice lists ErrNotFound before ErrTaskNotFound.
		multiWrapped := fmt.Errorf("combined: %w", errors.Join(apperrors.ErrTaskNotFound, apperrors.ErrNotFound))
		field := SentinelField(multiWrapped)
		assert.Equal(t, "ErrNotFound", field.String)
	})
}

// TestSentinelField_Encodable verifies the field renders cleanly through zap
// JSON encoding so it can ship in any handler logger call.
func TestSentinelField_Encodable(t *testing.T) {
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	entry := zapcore.Entry{Message: "smoke"}
	for _, err := range []error{
		apperrors.ErrTaskNotFound,
		apperrors.NewBusinessError(apperrors.CodeInvalidInput, "bad", nil),
		errors.New("unknown"),
		nil,
	} {
		fields := []zapcore.Field{SentinelField(err)}
		buf, encErr := encoder.EncodeEntry(entry, fields)
		assert.NoError(t, encErr)
		_ = buf.String()
	}
}
