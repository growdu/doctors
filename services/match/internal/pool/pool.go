// Package pool 实现抢单池（Redis ZSET）。
//
// 设计要点：
//   - 每个订单一个 ZSET key：`match:order:<id>`，member=escort_id，score=计算得分。
//   - Peek 取 TopN 候选（按 score DESC）。
//   - Pop 用 Lua 原子：ZREM 候选 + 写一个 "taken" 集合，避免同一 escort 重复占用。
//   - 整个 key 有 TTL（expire），过期表示订单已被消费或已超时。
//   - 单元测试用 NopPool（内存 map）；集成测试用 go-redis 真客户端。
package pool

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Pool 是抢单池抽象。
type Pool interface {
	Push(ctx context.Context, orderID int64, candidates []Candidate, expire time.Duration) error
	Pop(ctx context.Context, orderID, escortID int64) (bool, error)
	Peek(ctx context.Context, orderID int64, topN int) ([]int64, error)
}

// Candidate 是推送池的一个候选（escort + 得分）。
type Candidate struct {
	EscortID int64
	Score    float64
}

// ---------- NopPool（内存实现，单元测试用） ----------

// NopPool 用 map 模拟；不持久化，进程退出即清空。
// 单线程顺序执行时不保证并发安全；测试只验证语义。
type NopPool struct {
	data map[string]map[int64]float64
}

// NewNopPool 构造一个 NopPool。
func NewNopPool() *NopPool {
	return &NopPool{data: map[string]map[int64]float64{}}
}

func key(orderID int64) string {
	return fmt.Sprintf("match:order:%d", orderID)
}

// Push 用 candidate list 覆盖订单 key。
func (p *NopPool) Push(ctx context.Context, orderID int64, candidates []Candidate, _ time.Duration) error {
	m := map[int64]float64{}
	for _, c := range candidates {
		m[c.EscortID] = c.Score
	}
	p.data[key(orderID)] = m
	return nil
}

// Pop 用 escortID 在池中找到并删除；返回是否真的被 pop。
func (p *NopPool) Pop(ctx context.Context, orderID, escortID int64) (bool, error) {
	m, ok := p.data[key(orderID)]
	if !ok {
		return false, nil
	}
	if _, ok := m[escortID]; !ok {
		return false, nil
	}
	delete(m, escortID)
	return true, nil
}

// Peek 按 score DESC 取前 N 个 escortID。
func (p *NopPool) Peek(ctx context.Context, orderID int64, topN int) ([]int64, error) {
	m, ok := p.data[key(orderID)]
	if !ok {
		return nil, nil
	}
	type kv struct {
		id  int64
		sco float64
	}
	pairs := make([]kv, 0, len(m))
	for id, s := range m {
		pairs = append(pairs, kv{id, s})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].sco > pairs[j].sco })
	out := make([]int64, 0, topN)
	for i := 0; i < len(pairs) && i < topN; i++ {
		out = append(out, pairs[i].id)
	}
	return out, nil
}

// ---------- RedisPool（集成测试 / 生产） ----------

// RedisPool 用 Redis ZSET 实现 Pool；Pop 走 Lua 脚本保证原子。
type RedisPool struct {
	rdb *redis.Client
	// popScript 是 ZREM + SADD "taken" 组合；KEYS=order_key, ARGV=escort_id。
	popScript *redis.Script
}

// NewRedisPool 构造一个 RedisPool。
func NewRedisPool(rdb *redis.Client) *RedisPool {
	return &RedisPool{
		rdb: rdb,
		popScript: redis.NewScript(`
			local removed = redis.call('ZREM', KEYS[1], ARGV[1])
			if removed == 0 then return 0 end
			redis.call('SADD', KEYS[1] .. ':taken', ARGV[1])
			return 1
		`),
	}
}

// Push 把候选列表 ZADD 到 order_key；并 EXPIRE。
func (p *RedisPool) Push(ctx context.Context, orderID int64, candidates []Candidate, expire time.Duration) error {
	if expire <= 0 {
		expire = 5 * time.Minute
	}
	k := key(orderID)
	pipe := p.rdb.TxPipeline()
	pipe.Del(ctx, k)
	if len(candidates) > 0 {
		members := make([]redis.Z, 0, len(candidates))
		for _, c := range candidates {
			members = append(members, redis.Z{Score: c.Score, Member: c.EscortID})
		}
		pipe.ZAdd(ctx, k, members...)
	}
	pipe.Expire(ctx, k, expire)
	_, err := pipe.Exec(ctx)
	return err
}

// Pop 走 Lua 原子脚本。
func (p *RedisPool) Pop(ctx context.Context, orderID, escortID int64) (bool, error) {
	k := key(orderID)
	res, err := p.popScript.Run(ctx, p.rdb, []string{k}, escortID).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}
	return res == 1, nil
}

// Peek ZREVRANGE 取 TopN escort_id。
func (p *RedisPool) Peek(ctx context.Context, orderID int64, topN int) ([]int64, error) {
	k := key(orderID)
	res, err := p.rdb.ZRevRange(ctx, k, 0, int64(topN-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(res))
	for _, s := range res {
		id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}