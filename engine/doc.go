package engine

// 在真实的实践场景中，我们常常需要爬取多个初始网站，
// 我们希望能够同时爬取这些网站。这就需要合理调度和组织爬虫任务了。
// 因此，任务调度的高并发模型，使资源得到充分的利用。

// 调度引擎主要目标是完成下面几个功能：
// 1. 创建调度程序，接收任务并将任务存储起来；
// 2. 执行调度任务，通过一定的调度算法将任务调度到合适的 worker 中执行；
// 3. 创建指定数量的 worker，完成实际任务的处理；
// 4. 创建数据处理协程，对爬取到的数据进行进一步处理。
