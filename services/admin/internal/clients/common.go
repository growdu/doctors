package clients

import "strconv"

// fmtInt 把 int64 转 base-10 字符串。
func fmtInt(i int64) string { return strconv.FormatInt(i, 10) }