package engine

import (
	"github.com/nico612/crawler-go/collect"
	"go.uber.org/zap"
)

type options struct {
	WorkCount int             // 任务处理协程数量
	Fetcher   collect.Fetcher // 请求处理器
	Logger    *zap.Logger     // 日志
	Seeds     []*collect.Task // 任务列表
	scheduler Scheduler
}

type Option func(opts *options)

var defaultOptionss = options{
	Logger: zap.NewNop(),
}

func WithLogger(logger *zap.Logger) Option {
	return func(opts *options) {
		opts.Logger = logger
	}
}

func WithFetcher(fetcher collect.Fetcher) Option {
	return func(opts *options) {
		opts.Fetcher = fetcher
	}
}

func WithWorkCount(workCount int) Option {
	return func(opts *options) {
		opts.WorkCount = workCount
	}
}

func WithSeeds(seeds []*collect.Task) Option {
	return func(opts *options) {
		opts.Seeds = seeds
	}
}

func WithScheduler(scheduler Scheduler) Option {
	return func(opts *options) {
		opts.scheduler = scheduler
	}
}
