package proxy

import (
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
)

// 轮询调度（RR，Round-robin）是最简单的调度策略，轮询调度的意思是让每一个代理服务器都能够按顺序获得相同的负载。
// 下面让我们在项目中用轮询调度来实现对代理服务器的访问。

type ProxyFunc func(r *http.Request) (*url.URL, error)

// RoundRobinProxySwitcher creates a proxy switcher function which rotates
// ProxyURLs on every request.
// The proxy type is determined by the URL scheme. "http", "https"
// and "socks5" are supported. If the scheme is empty,
// "http" is assumed.
func RoundRobinProxySwitcher(proxyURLs ...string) (ProxyFunc, error) {
	if len(proxyURLs) < 1 {
		return nil, errors.New("Proxy URL list is  empty")
	}

	urls := make([]*url.URL, len(proxyURLs))
	for i, u := range proxyURLs {
		parsedU, err := url.Parse(u)
		if err != nil {
			return nil, err
		}

		urls[i] = parsedU
	}

	rs := &roundRobinSwitcher{
		proxyURLs: urls,
		index:     0,
	}

	return rs.GetProxy, nil
}

type roundRobinSwitcher struct {
	proxyURLs []*url.URL
	index     uint32
}

// GetProxy  取余算法实现轮询调度
func (r *roundRobinSwitcher) GetProxy(pr *http.Request) (*url.URL, error) {
	if len(r.proxyURLs) == 0 {
		return nil, errors.New("empty proxy urls")
	}
	// 对r.index进行自增，这里 -1 是获取自增前的值
	index := atomic.AddUint32(&r.index, 1) - 1
	u := r.proxyURLs[index%uint32(len(r.proxyURLs))]

	return u, nil
}
