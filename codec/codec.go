package codec

// 导入 io 包
import (
	"io"
)

// Header 结构体定义了消息头的格式
type Header struct {
	ServiceMethod string // 格式为 "Service.Method"
	Seq           uint64 // 客户端选择的序列号
	Error         string
}

// Codec 接口定义了编解码器的行为
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFunc 是一个函数类型，用于创建新的编解码器实例
type NewCodecFunc func(io.ReadWriteCloser) Codec

// Type 定义了一种类型，用于标识编解码器的类型
type Type string

// 定义一些常量，用于表示不同的编解码器类型
const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // 未实现
)

// NewCodecFuncMap 是一个映射，用于存储不同类型的新编解码器函数
var NewCodecFuncMap map[Type]NewCodecFunc

// init 函数在包加载时执行，用于初始化 NewCodecFuncMap
func init() {
	// 初始化 NewCodecFuncMap
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	// 将 GobType 的新编解码器函数添加到 NewCodecFuncMap 中
	NewCodecFuncMap[GobType] = NewGobCodec
}
