package main

import (
	"io"
	"log"
	"net/http"
)

// 正向代理
// 用 Go 实现的一个简单的 HTTP 正向代理服务如下所示。
// 在这个例子中，代理服务器接受来自客户端的 HTTP 请求，并通过 handleHTTP 函数对请求进行处理。
// 处理的方式也比较简单，当前代理服务器获取客户端的请求，并用自己的身份发送请求到服务器。
// 代理服务器获取到服务器的回复后，会再次利用 io.Copy 将回复发送回客户端。

func main() {
	server := &http.Server{
		Addr:    ":8888",
		Handler: http.HandlerFunc(handleHTTP),
	}

	log.Fatal(server.ListenAndServe())
}

// 对请求进行处理
func handleHTTP(w http.ResponseWriter, req *http.Request) {

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
