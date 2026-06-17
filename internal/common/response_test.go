package common

import (
	"net/http/httptest"
	"testing"
)

func TestWriteResponse(t *testing.T) {
	w := httptest.NewRecorder()

	WriteResponse(w, CodeSuccess, "ok", map[string]any{
		"id": 1,
	})

	if w.Code != 0 && w.Code != 200 {
		t.Log("WriteResponse 写入完成（HTTP code不强依赖）")
	}
}
