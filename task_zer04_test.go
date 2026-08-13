package zerolog_test

import (
	"bytes"
	"io"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestTaskZER04DictArrayPoolReturn 验证 Array.Dict 消费的临时字典 Event 会归还对象池。
//
// 现象：Array.Dict 消费 CreateDict 生成的临时字典 Event 后未归还对象池
// （Event.Dict 会调用 putEvent 归还，Array.Dict 不会）。持续循环写入字典数组时，
// 每次迭代都从空对象池新建字典 Event（结构体 + 500B 缓冲区 ≈ 2 次分配）；
// 修复后字典 Event 被复用，分配数趋近于零。
//
// 验证方式：以"数组内放普通字符串"为对照基线，用 testing.AllocsPerRun 分别测量
// 写入字典数组与普通数组的单次分配数；若前者显著高于后者（+1 以上），
// 说明对象池归还遗漏导致池化失效。
func TestTaskZER04DictArrayPoolReturn(t *testing.T) {
	// 功能前置核验：字典数组输出必须正确（Before/Gold 均满足，防止回归）
	var buf bytes.Buffer
	logw := zerolog.New(&buf)
	fe := logw.Info()
	fa := fe.CreateArray()
	fd := fe.CreateDict()
	fd.Str("k", "v")
	fa.Dict(fd)
	fe.Array("arr", fa)
	fe.Msg("")
	if !strings.Contains(buf.String(), `"arr":[{"k":"v"}]`) {
		t.Fatalf("字典数组输出异常: %q", buf.String())
	}

	// 关闭自动 GC，避免测量期间 GC 清空对象池或回收未归还对象造成抖动
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	log := zerolog.New(io.Discard)

	withDict := func() {
		e := log.Info()
		a := e.CreateArray()
		d := e.CreateDict()
		d.Str("k", "v")
		a.Dict(d)
		e.Array("arr", a)
		e.Msg("")
	}

	withoutDict := func() {
		e := log.Info()
		a := e.CreateArray()
		a.Str("v")
		e.Array("arr", a)
		e.Msg("")
	}

	avgWithDict := testing.AllocsPerRun(1000, withDict)
	avgWithoutDict := testing.AllocsPerRun(1000, withoutDict)

	t.Logf("withDict allocs/run=%v withoutDict allocs/run=%v", avgWithDict, avgWithoutDict)
	if avgWithDict > avgWithoutDict+1.0 {
		t.Fatalf("Array.Dict 未归还字典 Event 到对象池：持续写入字典数组时每次迭代额外分配（withDict=%v > withoutDict=%v+1）",
			avgWithDict, avgWithoutDict)
	}
}
