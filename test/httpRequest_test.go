package test

import (
	"github.com/TelephoneTan/GoHTTPRequest/net"
	"github.com/TelephoneTan/GoHTTPRequest/net/http"
	"github.com/TelephoneTan/GoHTTPRequest/net/http/method"
	"testing"
)

func TestGet(t *testing.T) {
	http.NewRequest(func(request http.Request) {
		request.Method = method.GET
		request.URL = "http://%E4%B8%96%E7%95%8C%E4%B8%8A%E5%8F%AA%E6%9C%89%E4%B8%80%E4%B8%AA%E4%BA%BA%20%20++%20++%20+%20%3F%20%5C%20%2F%20+%20=%20&%20:%E4%B8%96%E7%95%8C%E4%B8%8A%E5%8F%AA%E6%9C%89%E4%B8%80%E4%B8%AA%E4%BA%BA%20%20++%20++%20+%20%3F%20%5C%20%2F%20+%20=%20&%20@腾讯。中国:80/ nihao 你好 + ? a = %20 1 + 2 + 3 = ? \\ + = /?&  你好  ++ \\ / ? = ###123你好\\ %20 通用 +  = /?"
		request.Proxy = &net.Proxy{
			Type: net.HTTP,
			Host: "localhost",
			Port: net.Port(7892),
		}
	}).Send().Await()
}
