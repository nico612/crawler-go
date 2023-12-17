package collect

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/nico612/crawler-go/limiter"
	"github.com/nico612/crawler-go/storage"
	"go.uber.org/zap"
	"math/rand"
	"regexp"
	"sync"
	"time"
)

// 用广度优先搜索实战爬虫

type Property struct {
	Name     string `json:"name"` // 任务名称，应保任务证唯一性
	Url      string `json:"url"`
	Cookie   string `json:"cookie"`
	WaitTime int64  `json:"wait_time"` // 随机休眠时间，秒
	Reload   bool   `json:"reload"`    // 网站是否可以重复爬取
	MaxDepth int64  `json:"max_depth"`
}

// Task 爬虫一个任务实例
type Task struct {
	Property
	Visited     map[string]bool // 是否访问过
	VisitedLock sync.Mutex
	Fetcher     Fetcher         // 请求处理
	Storage     storage.Storage // 储存
	Rule        RuleTree        // 规则条件中 Root 生成了初始化的爬虫任务。Trunk 为爬虫任务中的所有规则。
	Logger      *zap.Logger
	Limit       limiter.RateLimiter
}

type Request struct {
	unique   string
	Task     *Task
	Url      string // 请求url
	Method   string
	Depth    int64  // 爬取深度，默认为0
	Priority int64  // 优先级
	RuleName string // 规则名
	TmpData  *Temp  // 临时数据缓存
}

// ParseResult 解析结果包含了需要进一步获取数据的 Requesrts 和 本次获取到的结果 Items
type ParseResult struct {
	Requesrts []*Request    // 请求数组
	Items     []interface{} // 获取到的数据
}

func (r *Request) Check() error {
	if r.Depth > r.Task.MaxDepth {
		return errors.New("Max depth limit reached")
	}
	return nil
}

// Unique 请求的唯一识别码
func (r *Request) Unique() string {
	block := md5.Sum([]byte(r.Url + r.Method))
	return hex.EncodeToString(block[:])
}

// Fetch 请求数据
func (r *Request) Fetch() ([]byte, error) {
	if err := r.Task.Limit.Wait(context.Background()); err != nil {
		return nil, err
	}
	// 随机休眠，模拟人类行为
	sleeptime := rand.Int63n(r.Task.WaitTime * 1000)
	time.Sleep(time.Duration(sleeptime) * time.Millisecond)
	return r.Task.Fetcher.Get(r)
}

type Context struct {
	Body []byte
	Req  *Request
}

// GetRule 获取采集规则
func (c *Context) GetRule(ruleName string) *Rule {
	return c.Req.Task.Rule.Trunk[ruleName]
}

// Output 输出解析数据
func (c *Context) Output(data interface{}) *storage.DataCell {
	res := &storage.DataCell{}
	res.Data = make(map[string]interface{})
	res.Data["Task"] = c.Req.Task.Name
	res.Data["Rule"] = c.Req.RuleName
	res.Data["Data"] = data
	res.Data["Url"] = c.Req.Url
	res.Data["Time"] = time.Now().Format("2006-01-02 15:04:05")
	return res
}

func (c *Context) ParseJSReg(name string, reg string) ParseResult {
	re := regexp.MustCompile(reg)

	matches := re.FindAllSubmatch(c.Body, -1)
	result := ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requesrts = append(
			result.Requesrts, &Request{
				Method:   "GET",
				Task:     c.Req.Task,
				Url:      u,
				Depth:    c.Req.Depth + 1,
				RuleName: name,
			})
	}
	return result
}

func (c *Context) OutputJS(reg string) ParseResult {
	re := regexp.MustCompile(reg)
	ok := re.Match(c.Body)
	if !ok {
		return ParseResult{
			Items: []interface{}{},
		}
	}
	result := ParseResult{
		Items: []interface{}{c.Req.Url},
	}
	return result
}
