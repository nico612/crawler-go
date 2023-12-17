package main

import (
	"context"
	"github.com/chromedp/chromedp"
	"log"
	"time"
)

// 远程与浏览器交互

func main() {

	// 1. 设置超时时间
	ctx, cancle := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancle()

	// 2. 初始化浏览器实例
	chromedp.NewContext(ctx)

	/// 3. 爬取页面，等待某一个元素出现，接着模拟鼠标点击，最后获取数据
	var example string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`https://pkg.go.dev/time`),           // 指定爬取指定的网址
		chromedp.WaitVisible(`body > footer`),                  // 等待当前标签可见，参数使用的是CSS选择器的形式
		chromedp.Click(`#example-After`, chromedp.NodeVisible), // 模拟对某一个标签的点击事件
		chromedp.Value(`#example-After textarea`, &example),    // 用于获取指定标签的数据
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Go's time.After example:\\n%s", example)

}
