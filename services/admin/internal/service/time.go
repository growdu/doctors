package service

import "time"

// nowFn 抽象时间来源，便于测试注入。
var nowFn = func() time.Time { return time.Now() }