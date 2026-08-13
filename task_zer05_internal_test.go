package zerolog

import (
	"context"
	"io"
	"testing"
)

// taskZER05CaptureMarshaler records the *Event that the serializer hands to a
// nested LogObjectMarshaler so the test can inspect whether the parent event's
// settings (ctx, stack, hooks) were carried over to the child object.
type taskZER05CaptureMarshaler struct {
	got *Event
}

func (m *taskZER05CaptureMarshaler) MarshalZerologObject(e *Event) {
	m.got = e
}

func TestTaskZER05NestedEventInheritsSettings(t *testing.T) {
	const key = "taskZER05-inherit-key"
	ctx := context.WithValue(context.Background(), key, "v")
	hook := HookFunc(func(e *Event, level Level, message string) {})
	logger := New(io.Discard).With().Ctx(ctx).Stack().Logger().Hook(hook)

	var m taskZER05CaptureMarshaler
	logger.Log().Objects("list", []LogObjectMarshaler{&m})

	if m.got == nil {
		t.Fatal("Objects did not invoke the nested marshaler")
	}
	if m.got.ctx != ctx {
		t.Errorf("nested object event ctx not inherited: got %v, want %v", m.got.ctx, ctx)
	}
	if v := m.got.GetCtx().Value(key); v != "v" {
		t.Errorf("nested object event GetCtx() lost the parent context value: got %v, want %v", v, "v")
	}
	if !m.got.stack {
		t.Error("nested object event stack setting not inherited")
	}
	if len(m.got.ch) != 1 {
		t.Errorf("nested object event hooks not inherited: got %d hooks, want 1", len(m.got.ch))
	}
}
