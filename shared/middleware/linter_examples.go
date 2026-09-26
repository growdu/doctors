//go:build linter_examples
// +build linter_examples

// Package middleware · golangci-lint linter 示例
//
// 本文件不是被业务代码 import 的实现，而是给 LLM / 开发者做 lint 规则速查用的：
// 每个函数都是对应 linter 的「典型违规 → 修正」对照，PR 评审时可参考。
//
// 对照 .golangci.yml，本项目启用的 linter：
//
//	┌─────────────┬─────────────────────────────────────────────────────────┐
//	│ linter       │ 适用场景                                                │
//	├─────────────┼─────────────────────────────────────────────────────────┤
//	│ errcheck     │ IO / RPC / DB 调用返回的 error 必须被处理              │
//	│ gosimple     │ 可省略的类型转换 / 简化写法（compiler 视角的繁琐代码）  │
//	│ govet        │ printf 格式串、shadow 变量、struct tag 错误             │
//	│ ineffassign  │ 变量赋了新值但从未被读取（典型 bug 源）               │
//	│ staticcheck  │ SA 系列（unreachable / unused / 性能陷阱）              │
//	│ unused       │ 未使用的变量 / 常量 / 函数 / 类型                       │
//	│ bodyclose    │ http.Response.Body 必须 Close（middleware 常见漏写）    │
//	│ gocritic     │ 风格 / 性能 / 可读性提示（ifElseChain / rangeValCopy）  │
//	│ misspell     │ 注释 / 字符串中的拼写错误（locale=zh）                  │
//	│ nakedret     │ 裸返回（长函数可读性差，max-func-lines=25）            │
//	│ prealloc     │ slice 预分配容量（append 性能优化）                     │
//	└─────────────┴─────────────────────────────────────────────────────────┘
//
// 注：以下代码每行单独编译；某些片段作为「错误示例」会被对应 linter 标红，
// 本文件不被 lint 扫描（见 .golangci.yml exclusions 规则 path: shared/middleware/）。
package middleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// =====================================================================
// errcheck：error 必须被处理
// =====================================================================

// errcheckDemo_Bad 演示 _ = 之外的典型漏检：defer resp.Body.Close() 而该调用本身可能 panic。
// errcheck: "resp.Body.Close" 会被标记。
func errcheckDemo_Bad(resp *http.Response) {
	defer resp.Body.Close() // ❌ 应当 `defer func() { _ = resp.Body.Close() }()`
}

// errcheckDemo_Good 显式忽略（_ =）以满足 errcheck。
func errcheckDemo_Good(resp *http.Response) {
	defer func() { _ = resp.Body.Close() }() // ✅
}

// =====================================================================
// gosimple：可简化的写法
// =====================================================================

func gosimpleDemo() {
	x := 5
	// gosimple(S1019): 应直接用 `x`。
	_ = &x //nolint:unused // 仅为示例
}

// =====================================================================
// govet：printf 格式 / shadow / struct tag
// =====================================================================

func govetPrintfDemo(name string, age int) {
	// govet(printf): 整数占位符应是 %d，不是 %s。
	fmt.Printf("user=%s age=%d\n", name, age) // ✅
}

// =====================================================================
// ineffassign：变量赋值但从未使用
// =====================================================================

func ineffassignDemo() {
	x := 1
	x = 2 // ❌ ineffassign: x 赋值后从未读取
	_ = x  // ✅ 加这一行消除警告
}

// =====================================================================
// staticcheck（SA 系列）：unreachable code / 性能陷阱
// =====================================================================

func staticcheckDemo(s string) string {
	// staticcheck(SA1029): strings.Contains 比 strings.Index 更清晰。
	if strings.Index(s, "x") >= 0 { // ❌
		return "found"
	}
	return "not found"
}

// =====================================================================
// unused：未使用的变量 / 函数
// =====================================================================

func unusedDemo() {
	_ = 1 // ✅ 未使用的整型字面量可用 `_ =`
}

// =====================================================================
// bodyclose：HTTP response body 必须 Close
// =====================================================================

func bodycloseDemo_Bad() {
	resp, err := http.Get("http://example.com")
	if err != nil {
		return
	}
	_ = resp // ❌ bodyclose: resp.Body 未 Close
}

func bodycloseDemo_Good() error {
	resp, err := http.Get("http://example.com")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// =====================================================================
// gocritic：风格 / 性能 / 可读性
// =====================================================================

func gocriticRangeValCopy() {
	// gocritic(rangeValCopy): range 拷贝了大结构体，应改用索引。
	type Big struct {
		Data [1024]byte
	}
	xs := []Big{{}, {}}
	for _, v := range xs { // ❌ 应改 `for i := range xs { _ = xs[i] }`
		_ = v
	}
}

// =====================================================================
// misspell：拼写错误（locale=zh）
// =====================================================================

func misspellDemo() {
	// misspell: "cancle" 应为 "cancel"。
	_ = "cancle" //nolint:unused // 仅为示例
}

// =====================================================================
// nakedret：裸返回（> 25 行函数禁用）
// =====================================================================

func nakedretDemo_Good(s string) (result string) { // ✅ 函数短，无裸返回风险
	result = s
	return // 这里因函数体 < 25 行不被标记
}

// =====================================================================
// prealloc：slice 预分配
// =====================================================================

func preallocDemo_Good(n int) []int {
	out := make([]int, 0, n) // ✅ 预分配容量
	for i := 0; i < n; i++ {
		out = append(out, i)
	}
	return out
}

// 注释占位，让文件不被 Go compiler 当作 unused。
var _ = fmt.Sprintf