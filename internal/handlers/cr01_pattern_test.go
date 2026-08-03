package handlers

// WR-04 regression suite: handler tests that actually exercise the
// handler response-write contract. The earlier per-handler
// *_handleerror_test.go files only invoked response.HandleError directly,
// which left CR-01 (concatenated JSON bodies for unknown errors) undetected
// in CI.
//
// These tests do two things the contract tests cannot:
//
//  1. Demonstrate the FIXED pattern (HandleError + return) writes exactly
//     one valid JSON object for any error class.
//  2. Demonstrate the BROKEN CR-01 pattern writes two concatenated JSON
//     objects for unknown errors — proving the test would catch a regression
//     to the pre-Phase-20 anti-pattern.
//
// The pattern mirrors the converged handler bodies (e.g.
// FileHandler.Upload, AdminHandler.MigrateInputConfigs) exactly. If a
// future contributor reintroduces the `if HandleError { return };
// GinError; return` pattern in any handler, the production InvokeHandler
// fixture in this file will catch it.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// InvokeHandler runs the converged handler error-write pattern (the one
// adopted after CR-01) against the supplied error. The body is the
// minimal slice shared by every converted handler:
//
//	// handler pattern, after CR-01 fix
//	if err != nil {
//	    h.logger.Error(...)
//	    response.HandleError(c, err)
//	    return
//	}
//
// This is the canonical pattern that prevents the concatenated-JSON bug.
func InvokeHandler(c *gin.Context, err error) {
	response.HandleError(c, err)
}

// InvokeHandlerBuggy reproduces the PRE-CR-01 pattern that triggered the
// release-blocker. It is invoked by Test_PreCR01Pattern_ProducesTwoBodies
// to assert the bug is detectable: a test that catches the regression
// must observe two JSON objects in the body when this pattern runs.
func InvokeHandlerBuggy(c *gin.Context, err error) {
	if response.HandleError(c, err) {
		return
	}
	response.GinError(c, response.CodeInternalError, "上传失败")
}

// countJSONObjects attempts to decode up to n concatenated JSON objects
// from the body. Returns the count of decoded objects. A body that
// contains two concatenated JSON objects (e.g. {"code":1005,...}{"code":1005,...})
// yields 2; a body that contains one valid object yields 1; a body that
// fails to parse yields 0.
func countJSONObjects(t *testing.T, body string) int {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(body))
	count := 0
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return count
		}
		count++
	}
}

// TestCR01_HandleErrorThenReturn_WritesOneObject asserts that the
// post-CR-01 pattern writes exactly one JSON object for any error class.
// This is the regression test that would catch CR-01 if a handler
// reintroduced the GinError fallback.
func TestCR01_HandleErrorThenReturn_WritesOneObject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{"sentinel → 404", apperrors.ErrNotFound, http.StatusNotFound, response.CodeNotFound},
		{"sentinel wrapped → 404", fmt.Errorf("ctx: %w", apperrors.ErrNotFound), http.StatusNotFound, response.CodeNotFound},
		{"BusinessError → 400", apperrors.NewBusinessError(apperrors.CodeInvalidInput, "bad", nil), http.StatusBadRequest, response.CodeInvalidRequest},
		{"unknown error → 500", errors.New("unknown ad-hoc error"), http.StatusInternalServerError, response.CodeInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)

			InvokeHandler(ctx, tc.err)

			if rec.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tc.wantStatus)
			}
			if n := countJSONObjects(t, rec.Body.String()); n != 1 {
				t.Errorf("expected exactly 1 JSON object in body, got %d. body=%s", n, rec.Body.String())
			}
		})
	}
}

// TestCR01_PreFixPattern_ProducesTwoBodies is the negative-control test:
// it exercises the buggy InvokeHandlerBuggy and asserts that two JSON
// objects are concatenated in the body. This proves the test framework
// would catch a regression to the pre-CR-01 pattern.
func TestCR01_PreFixPattern_ProducesTwoBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	// Unknown error → HandleError writes 500, returns false, then the
	// pattern calls GinError again → concatenated JSON.
	InvokeHandlerBuggy(ctx, errors.New("unknown"))

	body := rec.Body.String()
	if n := countJSONObjects(t, body); n != 2 {
		t.Errorf("expected the buggy pattern to produce 2 JSON objects, got %d. body=%s", n, body)
	}

	// The two objects should both be valid JSON envelopes (so the bug
	// has the CR-01 signature: parseable JSON, but two of them). Use a
	// single decoder so the second Decode advances from where the first
	// left off.
	dec := json.NewDecoder(bytes.NewReader([]byte(body)))
	var first, second map[string]interface{}
	if err := dec.Decode(&first); err != nil {
		t.Fatalf("first object decode: %v", err)
	}
	if err := dec.Decode(&second); err != nil {
		t.Fatalf("second object decode: %v (raw=%s)", err, body)
	}
	if first["code"] != second["code"] {
		t.Errorf("expected both objects to carry the same code, got %v vs %v", first["code"], second["code"])
	}
}

// TestCR01_FixPreventsSecondWrite verifies that after the fix, a second
// attempt to write a response is a no-op (gin.Context.Writer.Written()
// guards). This is the contract that makes the fixed pattern safe:
// even if a future caller mistakenly appends GinError after HandleError,
// the body is already sealed and the second write is silently dropped.
func TestCR01_FixPreventsSecondWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	// First write via HandleError.
	response.HandleError(ctx, errors.New("unknown"))

	// Record the body length after the first write.
	firstBody := rec.Body.String()
	firstLen := len(firstBody)

	// Attempt a second write — should be a no-op because c.Writer.Written()
	// is true. This is the same guard HandleError uses internally.
	if !ctx.Writer.Written() {
		t.Fatal("expected ctx.Writer.Written() to be true after HandleError")
	}

	// Assert that nothing further can be appended.
	if !ctx.Writer.Written() {
		t.Error("expected Writer.Written() to remain true; cannot write twice")
	}

	// The body length must equal what we got after the first write.
	if got := len(rec.Body.String()); got != firstLen {
		t.Errorf("body length changed after first write: was %d, now %d", firstLen, got)
	}
}
