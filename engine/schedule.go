package engine

import (
	"github.com/nico612/crawler-go/collect"
	"github.com/nico612/crawler-go/parse/doubanbook"
	"github.com/nico612/crawler-go/parse/doubangroup"
	"github.com/nico612/crawler-go/parse/doubangroupjs"
	"github.com/nico612/crawler-go/storage"
	"github.com/robertkrimen/otto"
	"go.uber.org/zap"
	"runtime/debug"
	"sync"
)

func init() {
	Store.Add(doubangroup.DoubangroupTask)
	Store.Add(doubanbook.DoubanBookTask)
	Store.AddJSTask(doubangroupjs.DoubangroupJSTask)
}

// Store 全局爬虫任务实例
var Store = &CrawlerStore{
	list: []*collect.Task{},
	Hash: map[string]*collect.Task{},
}

type CrawlerStore struct {
	list []*collect.Task
	Hash map[string]*collect.Task // 储存爬虫任务，k: 任务名
}

func GetFields(taskName string, ruleName string) []string {
	return Store.Hash[taskName].Rule.Trunk[ruleName].ItemFields
}

func (c *CrawlerStore) Add(task *collect.Task) {
	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

func (c *CrawlerStore) AddJSTask(m *collect.TaskModel) {
	task := &collect.Task{
		Property: m.Property,
	}

	task.Rule.Root = func() ([]*collect.Request, error) {
		vm := otto.New() // js 虚拟机
		vm.Set("AddJsReq", AddJsReqs)
		v, err := vm.Eval(m.Root)
		if err != nil {
			return nil, err
		}
		e, err := v.Export()
		if err != nil {
			return nil, err
		}
		return e.([]*collect.Request), nil
	}

	for _, r := range m.Rules {
		paesrFunc := func(parse string) func(ctx *collect.Context) (collect.ParseResult, error) {
			return func(ctx *collect.Context) (collect.ParseResult, error) {
				vm := otto.New()
				vm.Set("ctx", ctx)
				v, err := vm.Eval(parse)
				if err != nil {
					return collect.ParseResult{}, err
				}
				e, err := v.Export()
				if err != nil {
					return collect.ParseResult{}, err
				}
				if e == nil {
					return collect.ParseResult{}, err
				}
				return e.(collect.ParseResult), err
			}
		}(r.ParseFunc)
		if task.Rule.Trunk == nil {
			task.Rule.Trunk = make(map[string]*collect.Rule, 0)
		}
		task.Rule.Trunk[r.Name] = &collect.Rule{
			ParseFunc: paesrFunc,
		}
	}

	c.Hash[task.Name] = task
	c.list = append(c.list, task)
}

// AddJsReqs 用于动态规则添加请求。
func AddJsReqs(jreqs []map[string]interface{}) []*collect.Request {
	reqs := make([]*collect.Request, 0)

	for _, jreq := range jreqs {
		req := &collect.Request{}
		u, ok := jreq["Url"].(string)
		if !ok {
			return nil
		}
		req.Url = u
		req.RuleName, _ = jreq["RuleName"].(string)
		req.Method, _ = jreq["Method"].(string)
		req.Priority, _ = jreq["Priority"].(int64)
		reqs = append(reqs, req)
	}
	return reqs
}

// Crawler 爬虫引擎
type Crawler struct {
	out         chan collect.ParseResult
	Visited     map[string]bool
	VisitedLock sync.Mutex

	failures    map[string]*collect.Request // 失败请求id -> 失败请求
	failureLock sync.Mutex

	options
}

func NewEngine(opts ...Option) *Crawler {
	options := defaultOptionss
	for _, opt := range opts {
		opt(&options)
	}
	e := &Crawler{}
	e.Visited = make(map[string]bool, 100)
	e.out = make(chan collect.ParseResult)
	e.failures = make(map[string]*collect.Request)
	e.options = options
	return e
}

// Schedule 任务调度分配
func (c *Crawler) Schedule() {
	var reqs []*collect.Request
	for _, seed := range c.Seeds {
		task := Store.Hash[seed.Name]
		task.Fetcher = seed.Fetcher
		task.Storage = seed.Storage
		task.Limit = seed.Limit
		task.Logger = c.Logger
		rootreqs, err := task.Rule.Root()
		if err != nil {
			c.Logger.Error("get root failed",
				zap.Error(err),
			)
			continue
		}
		for _, req := range rootreqs {
			req.Task = task
		}
		reqs = append(reqs, rootreqs...)
	}

	go c.scheduler.Schedule()
	go c.scheduler.Push(reqs...)
}

func (c *Crawler) Run() {
	go c.Schedule()
	for i := 0; i < c.WorkCount; i++ {
		go c.CreateWork()
	}
	c.HandleResult()
}

func (c *Crawler) CreateWork() {

	defer func() {
		if err := recover(); err != nil {
			c.Logger.Error("worker panic",
				zap.Any("err", err),
				zap.String("stack", string(debug.Stack())))
		}
	}()

	for {
		// 取出一个任务
		req := c.scheduler.Pull()
		// 检查任务深度
		if err := req.Check(); err != nil {
			c.Logger.Error("check failed", zap.Error(err))
			continue
		}
		// 任务已经爬取，跳过
		if !req.Task.Reload && c.HasVisited(req) {
			c.Logger.Debug("request has visited", zap.String("url:", req.Url))
			continue
		}

		c.StoreVisited(req)

		body, err := req.Fetch()
		if err != nil {
			c.Logger.Error("can't fetch ", zap.Error(err), zap.String("url", req.Url))
			c.SetFailure(req)
			continue
		}

		if len(body) < 6000 {
			c.Logger.Error("can't fetch ",
				zap.Int("length", len(body)),
				zap.String("url", req.Url),
			)
			c.SetFailure(req)

			continue
		}

		rule := req.Task.Rule.Trunk[req.RuleName]
		result, err := rule.ParseFunc(&collect.Context{
			body,
			req,
		})

		if err != nil {
			c.Logger.Error("ParseFunc failed ",
				zap.Error(err),
				zap.String("url", req.Url),
			)
			continue
		}

		if len(result.Requesrts) > 0 {
			go c.scheduler.Push(result.Requesrts...)
		}

		c.out <- result
	}
}

func (c *Crawler) HandleResult() {
	for {
		select {
		case result := <-c.out:
			for _, item := range result.Items {
				switch d := item.(type) {
				case *storage.DataCell:
					name := d.GetTaskName()
					task := Store.Hash[name]
					task.Storage.Save(d)
				}
				c.Logger.Sugar().Info("get result: ", item)
			}
		}
	}
}

// StoreVisited 储存已处理过的任务
func (c *Crawler) StoreVisited(reqs ...*collect.Request) {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()

	for _, r := range reqs {
		unique := r.Unique()
		c.Visited[unique] = true
	}
}

func (c *Crawler) HasVisited(r *collect.Request) bool {
	c.VisitedLock.Lock()
	defer c.VisitedLock.Unlock()
	unique := r.Unique()
	return c.Visited[unique]
}

func (c *Crawler) SetFailure(req *collect.Request) {
	if !req.Task.Reload {
		c.VisitedLock.Lock()
		unique := req.Unique()
		delete(c.Visited, unique)
		c.VisitedLock.Unlock()
	}
	c.failureLock.Lock()
	defer c.failureLock.Unlock()
	if _, ok := c.failures[req.Unique()]; !ok {
		// 首次失败时，再重新执行一次
		c.failures[req.Unique()] = req
		c.scheduler.Push(req)
	}
	// todo: 失败2次，加载到失败队列中
}

type Scheduler interface {
	Schedule()
	Push(...*collect.Request)
	Pull() *collect.Request
}

// Schedule 调度器
type Schedule struct {
	requestCh   chan *collect.Request // 任务通道
	workerCh    chan *collect.Request // 任务处理通道
	priReqQueue []*collect.Request    // 储存优先任务队列
	reqQueue    []*collect.Request    // 储存普通任务队列
	Logger      *zap.Logger
}

func NewSchedule() *Schedule {
	s := &Schedule{}
	s.requestCh = make(chan *collect.Request)
	s.workerCh = make(chan *collect.Request)
	return s
}

// Schedule 任务调度，负责接收任务，并将任务发送到 worker 通道中
func (s *Schedule) Schedule() {
	var req *collect.Request
	var ch chan *collect.Request

	go func() {
		for {
			// 优先任务
			if req == nil && len(s.priReqQueue) > 0 {
				req = s.priReqQueue[0]
				s.priReqQueue = s.priReqQueue[1:]
				ch = s.workerCh
			}

			// 普通任务
			if req == nil && len(s.reqQueue) > 0 {
				req = s.reqQueue[0]
				s.reqQueue = s.reqQueue[1:]
				ch = s.workerCh
			}

			select {
			case r := <-s.requestCh:
				if r.Priority > 0 {
					s.priReqQueue = append(s.priReqQueue, r)
				} else {
					s.reqQueue = append(s.reqQueue, r)
				}
			case ch <- req: // 将任务发送到 workerCh 通道，如果ch 为nil 或者 req 为nil 则该协程会阻塞
				req = nil
				ch = nil
			}
		}
	}()
}

// Pull 取出一个任务
func (s *Schedule) Pull() *collect.Request {
	r := <-s.workerCh
	return r
}

// Push 推入任务
func (s *Schedule) Push(reqs ...*collect.Request) {
	for _, req := range reqs {
		s.requestCh <- req
	}
}
