package collect

// 静态规则引擎, 静态规则处理的确定性强，它适合对性能要求高的爬虫任务。

// RuleTree 采集规则树
// 其实规则引擎就像一棵树。RuleTree.Root 是一个函数，用于生成爬虫的种子网站，
// RuleTree.Trunk 是一个规则哈希表，用于存储当前任务所有的规则，哈希表的 Key 为规则名，Value 为具体的规则。
type RuleTree struct {
	Root  func() ([]*Request, error) // 根节点（执行入口）
	Trunk map[string]*Rule           // 规则哈希表，爬虫任务中所有的规则
}

// Rule 采集规则节点
type Rule struct {
	ItemFields []string
	ParseFunc  func(*Context) (ParseResult, error) // 内容解析函数
}
