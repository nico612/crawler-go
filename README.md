# crawler

collect: 数据采集，负责网络请求，拿到结果

parse: 负责对collect请求的内容进行数据的解析，自定义解析规则

proxy: 代理，用轮询调度来实现对代理服务器的访问。

engine: 调度引擎，负责 接收任务、任务分配、结果处理工作