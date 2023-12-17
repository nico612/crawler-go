package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// HTTP 隧道代理
// 在一些更复杂的情况下，客户端希望与服务器进行 HTTPS 通信和 HTTP 隧道技术（HTTP Tunnel）形式的通信
// ，防止中间人攻击并隐藏 HTTP 的特征。

// 在 HTTP 隧道技术中，客户端会在第一次连接代理服务器时给代理服务器发送一个指令，通常是一个 HTTP 请求。
// 这里我们可以将 HTTP 请求头中的 method 设置为 CONNECT。

// 代理服务器收到该指令后，将与目标服务器建立 TCP 连接。连接建立后，代理服务器会将之后收到的请求通过 TCP 连接转发给目标服务器。
// 因此，只有初始连接请求是 HTTP， 之后，代理服务器将不再嗅探到任何数据，
// 它只是完成一个转发的动作。现在如果我们去查看其他开源的代理库，就会明白为什么会对 CONNECT 方法进行单独的处理了，这是业内通用的一种标准。

func main() {
	server := &http.Server{
		Addr: ":9981",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
	}

	log.Fatal(server.ListenAndServe())
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	// 拿到客户端与代理服务器之间的底层 TCP 连接
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
