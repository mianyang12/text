package gee

import (
	"fmt"
	"strings"
)

type node struct {
	pattern  string
	part     string
	children []*node
	isWild   bool
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

// insert 方法将给定的模式字符串 pattern 插入到当前的 node 结构体中
func (n *node) insert(pattern string, parts []string, height int) {
	// 如果 parts 的长度等于 height，这意味着已经到达了树的底部，将 pattern 赋值给当前节点的 pattern 字段
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	// 获取当前层级的 part
	part := parts[height]
	// 在当前节点的子节点中查找匹配的 child
	child := n.matchChild(part)
	// 如果没有找到匹配的 child，创建一个新的 node 节点作为 child
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	// 递归地在 child 节点上调用 insert 方法，将 pattern 插入到子树中
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}

func (n *node) travel(list *([]*node)) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.travel(list)
	}
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}
