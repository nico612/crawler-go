package collect

// 动态规则引擎
// 动态规则带来的另一个好处是，降低了书写代码规则的门槛，它甚至可以让业务人员也能书写简单的规则。
// 说到在爬虫项目中实现动态规则的引擎，我们首先想到的就是使用 Javascript 虚拟机了。因为使用 JS 操作网页有天然的优势。

// TaskModel 动态规则任务模型
type TaskModel struct {
	Property
	Root  string      `json:"root_script"`
	Rules []RuleModel `json:"rule"`
}

// RuleModel 动态规则模型
type RuleModel struct {
	Name      string `json:"name"`
	ParseFunc string `json:"parse_script"`
}
