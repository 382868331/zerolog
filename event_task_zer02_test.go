package zerolog

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

// TestTaskZER02ErrNilStackMarshalerStillLogsError 验证：
// 当 ErrorStackMarshaler 返回 nil（错误无 stack trace，例如普通 fmt.Errorf 错误）时，
// Event.Err 仍必须写入基础 error 字段（错误消息不丢失），stack 字段可缺省。
func TestTaskZER02ErrNilStackMarshalerStillLogsError(t *testing.T) {
	original := ErrorStackMarshaler
	ErrorStackMarshaler = func(err error) interface{} { return nil }
	defer func() { ErrorStackMarshaler = original }()

	var buf bytes.Buffer
	log := New(&buf)
	serr := errors.New("boom")
	log.Log().Stack().Err(serr).Msg("test message")

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON %q: %v", buf.String(), err)
	}
	if got, ok := out["error"]; !ok || got != "boom" {
		t.Fatalf("error field = %v (present=%v), want \"boom\"; full output: %s", got, ok, buf.String())
	}
	if got := out["message"]; got != "test message" {
		t.Fatalf("message field = %v, want \"test message\"; full output: %s", got, buf.String())
	}
}
