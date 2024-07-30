package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
)

// trace 函数用于打印当前函数调用栈的回溯信息，包括文件名和行号
func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

// Recovery 是一个中间件，用于从 panic 中恢复
func Recovery() HandlerFunc {
	// 创建返回函数
	return func(c *Context) {
		// 延迟执行匿名函数
		defer func() {
			// 尝试从 panic 中恢复
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				// 打印跟踪信息
				log.Printf("%s\n\n", trace(message))
				// 返回错误
				c.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		// 执行下一个Handler
		c.Next()
	}
}
