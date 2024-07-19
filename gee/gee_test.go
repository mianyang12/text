package gee

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {
	engine := New()

	// 断言 Engine 对象不为 nil
	assert.NotNil(t, engine)

	// 断言 router 字段已初始化
	assert.NotNil(t, engine.router)
	assert.Empty(t, engine.router)
}
