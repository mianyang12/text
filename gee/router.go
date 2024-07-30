package gee

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// parsePattern 函数将模式字符串分割成部分，并返回一个字符串切片
func parsePattern(pattern string) []string {
	// 使用 strings.Split 函数将模式字符串按照 / 字符分割
	vs := strings.Split(pattern, "/")

	// 初始化一个空的字符串切片，用于保存分割后的结果
	parts := make([]string, 0)
	for _, item := range vs {
		// 如果分割后的字符串不为空，则将其添加到 parts 切片中
		if item != "" {
			parts = append(parts, item)
			// 如果字符串以 * 开头，则表示这是一个通配符模式，遇到通配符模式后，可以停止后续的分割操作
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)

	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

// router 结构体的 handle 方法用于处理传入的 HTTP 请求
func (r *router) handle(c *Context) {
	// 通过请求的 Method 和 Path 获取匹配的路由节点和路径参数
	n, params := r.getRoute(c.Method, c.Path)

	// 如果获取到符合的路由节点
	if n != nil {
		// 生成唯一的键
		key := c.Method + "-" + n.pattern
		// 设置上下文的路径参数
		c.Params = params
		// 添加路由节点对应的处理器到上下文的处理器列表中
		c.handlers = append(c.handlers, r.handlers[key])
		// 如果没有获取到符合的路由节点
	} else {
		// 添加一个匿名函数到上下文的处理器列表，用于处理 404 错误
		c.handlers = append(c.handlers, func(c *Context) {
			// 返回 404 状态码和错误信息
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}

	// 调用下一个处理器或执行中间件的逻辑
	c.Next()
}
