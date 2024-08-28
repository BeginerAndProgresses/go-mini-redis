# go-mini-redis
一个简单的redis实现，包含对于RESP协议的解析和封装，以及一些基本的命令实现。
使用该库时，需要先安装依赖库：

> go get -u github.com/BeginerAndProgresses/generalized-tools

当使用该包的RESP解析时，可参考以下代码:
```go
	resp := NewRESP()
	row := []byte("*2\r\n$3\r\nget\r\n$1\r\na\r\n")
	after, res := resp.Parse(row)
	if len(after) > 0 {
		t.Logf("after:%s", after)
	} else {
		for _, arr := range res.(Array) {
			t.Logf("%v", arr)
		}
	}
```
输出结果为
>
> resp_test.go:301: get
> 
> resp_test.go:301: a
>


