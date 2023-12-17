package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// 反向代理
// 与正向代理不同的是，反向代理位于服务器的前方，客户端不能直接与服务器进行通信，需要通过反向代理。
// 我们比较熟悉的 Nginx 一般就是用于实现反向代理的。

func main() {
	// 初始化反向代理服务
	proxy, err := NewProxy()
	if err != nil {
		panic(err)
	}

	// 所有请求都由 ProxyRequestHandler 函数进行处理
	http.HandleFunc("/", ProxyRequestHandler(proxy))
	log.Fatal(http.ListenAndServe(":8000", nil))
}

// NewProxy 生成一个反向代理服务器
func NewProxy() (*httputil.ReverseProxy, error) {
	// 实际的后端服务器地址
	targetHost := "http://my-api-server.com"
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}

	// 内部封装了数据转发等操作。当客户端访问我们的代理服务器时，请求会被转发到对应的目标服务器中
	proxy := httputil.NewSingleHostReverseProxy(url)
	return proxy, nil
}

// ProxyRequestHandler 使用代理处理HTTP请求
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
