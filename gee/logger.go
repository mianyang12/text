package gee

import (
	"log"
	"time"
)

// Logger 是一个 HTTP 中间件，用于记录请求的处理时间和状态码
func Logger() HandlerFunc {
	return func(c *Context) {
		// 记录开始时间
		t := time.Now()
		// 处理请求
		c.Next()
		// 记录处理时间和状态码
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
