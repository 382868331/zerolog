package zerolog_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// This file reproduces a serialization defect: nested Object/EmbedObject
// serialization does not carry the parent event's context, hooks and stack
// settings over to the child object, so nested error fields behave differently
// from top-level ones.

type taskZER05CtxKey string

const taskZER05TraceKey taskZER05CtxKey = "taskZER05_trace_id"

// taskZER05TraceMarshaler reads the Go context available on the *Event handed
// to MarshalZerologObject and renders the trace id (or "missing"). Top-level
// Object/EmbedObject serialize on the parent event, so they see the parent
// context; nested objects (Objects/Fields/Context.Object) historically got a
// detached temp event whose context was lost.
type taskZER05TraceMarshaler struct{}

func (taskZER05TraceMarshaler) MarshalZerologObject(e *zerolog.Event) {
	if v := e.GetCtx().Value(taskZER05TraceKey); v != nil {
		e.Str("trace", v.(string))
	} else {
		e.Str("trace", "missing")
	}
}

// taskZER05ErrMarshaler serializes an error through e.Err so that the "stack"
// field is only emitted when the event carrying it has stack support enabled.
type taskZER05ErrMarshaler struct {
	err error
}

func (m taskZER05ErrMarshaler) MarshalZerologObject(e *zerolog.Event) {
	e.Err(m.err)
}

func taskZER05TraceLogger(out *bytes.Buffer) zerolog.Logger {
	ctx := context.WithValue(context.Background(), taskZER05TraceKey, "abc123")
	return zerolog.New(out).With().Ctx(ctx).Logger()
}

func TestTaskZER05CtxPropagationToNestedObject(t *testing.T) {
	out := &bytes.Buffer{}
	logger := taskZER05TraceLogger(out)
	m := taskZER05TraceMarshaler{}

	logger.Log().Object("top", m).Msg("")
	top := out.String()
	out.Reset()

	logger.Log().Objects("nested", []zerolog.LogObjectMarshaler{m, m}).Msg("")
	nested := out.String()
	out.Reset()

	if !strings.Contains(top, `"trace":"abc123"`) {
		t.Fatalf("top-level object should see the parent context, got: %s", top)
	}
	if !strings.Contains(nested, `"trace":"abc123"`) {
		t.Fatalf("nested object (Event.Objects) should inherit the parent context, got: %s", nested)
	}

	logger.Log().Fields(map[string]interface{}{"obj": m}).Msg("")
	fields := out.String()
	out.Reset()

	if !strings.Contains(fields, `"trace":"abc123"`) {
		t.Fatalf("nested object (Event.Fields) should inherit the parent context, got: %s", fields)
	}

	ctxLogger := zerolog.New(out).With().Ctx(context.WithValue(context.Background(), taskZER05TraceKey, "abc123")).Object("obj", m).Logger()
	ctxLogger.Log().Msg("")
	ctxOut := out.String()
	out.Reset()

	if !strings.Contains(ctxOut, `"trace":"abc123"`) {
		t.Fatalf("nested object (Context.Object) should inherit the logger context, got: %s", ctxOut)
	}
}

func TestTaskZER05StackPropagationToNestedObject(t *testing.T) {
	origStackMarshaler := zerolog.ErrorStackMarshaler
	zerolog.ErrorStackMarshaler = func(err error) interface{} { return err }
	defer func() { zerolog.ErrorStackMarshaler = origStackMarshaler }()

	out := &bytes.Buffer{}
	logger := zerolog.New(out).With().Stack().Logger()
	m := taskZER05ErrMarshaler{err: errors.New("boom")}

	logger.Log().Object("top", m).Msg("")
	top := out.String()
	out.Reset()

	logger.Log().Objects("nested", []zerolog.LogObjectMarshaler{m, m}).Msg("")
	nested := out.String()
	out.Reset()

	if !strings.Contains(top, `"stack":`) {
		t.Fatalf("top-level object should emit the error stack field, got: %s", top)
	}
	if !strings.Contains(nested, `"stack":`) {
		t.Fatalf("nested object should inherit the parent stack setting and emit the error stack field, got: %s", nested)
	}
}
