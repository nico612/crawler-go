package limiter

import (
	"context"
	"golang.org/x/time/rate"
	"sort"
	"time"
)

// RateLimiter 限速器接口
type RateLimiter interface {
	// Wait 等待可用的令牌, 参数 ctx 可以设置超时退出的时间，它可以避免协程一直陷在堵塞状态中。
	Wait(ctx context.Context) error
	Limit() rate.Limit
}

// Per 计算速率
func Per(eventCount int, duration time.Duration) rate.Limit {
	return rate.Every(duration / time.Duration(eventCount))
}

// MultiLimiter 函数用于聚合多个 RateLimiter，并将速率由小到大排序。
// Wait 方法会循环遍历多层限速器 multiLimiter 中所有的限速器并索要令牌，只有当所有的限速器规则都满足后，才会正常执行后续的操作
func MultiLimiter(limiters ...RateLimiter) *multiLimiter {
	byLimit := func(i, j int) bool {
		return limiters[i].Limit() < limiters[j].Limit()
	}
	sort.Slice(limiters, byLimit)
	return &multiLimiter{limiters: limiters}
}

// 多层限速
type multiLimiter struct {
	limiters []RateLimiter
}

func (l *multiLimiter) Wait(ctx context.Context) error {
	for _, limiter := range l.limiters {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *multiLimiter) Limit() rate.Limit {
	return l.limiters[0].Limit()
}
