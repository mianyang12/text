package test

// 导入 codec 包
import (
	"distributed/codec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

// 定义一个常量作为 MagicNumber，以标记这是一个特定的 RPC 请求
const MagicNumber = 0x3bef5c

// Option 结构体，包含了客户端与服务器端通信时所需的所有选项
type Option struct {
	MagicNumber    int           // MagicNumber 标记这是一个geerpc请求
	CodecType      codec.Type    // 客户端可能选择不同的Codec类型来编码body
	ConnectTimeout time.Duration // 0 代表没有限制
	HandleTimeout  time.Duration
}

// 设置默认选项
var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}

// Server 结构代表一个 RPC 服务器，管理服务和处理连接
type Server struct {
	serviceMap sync.Map // 存储服务名和服务实例的映射
}

// NewServer 返回一个新的 Server 实例
func NewServer() *Server {
	return &Server{}
}

// 默认服务器实例，用于简化调用
var DefaultServer = NewServer()

// 处理单个连接，接收并解析请求，分派到相应的服务
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	// 根据选项中的 CodecType 字段选择对应的 NewCodecFunc
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	// 用选择的编解码器函数处理连接和选项
	server.serveCodec(f(conn), &opt)
}

// 一个空结构体，作为错误时响应的占位符
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec, opt *Option) {
	// 初始化一个互斥锁，确保能够发送完整的响应。
	sending := new(sync.Mutex)
	// 初始化一个等待组，用于等待所有请求处理完毕。
	wg := new(sync.WaitGroup)
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // 如果请求解析失败，且请求为空，表明出错，直接关闭连接，结束循环
			}
			// 其他错误时，设置相应的错误信息。
			req.h.Error = err.Error()
			// 使用互斥锁发送出错时的响应信息。
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 处理正常请求，启动一个新的 Goroutine。
		go server.handleRequest(cc, req, sending, wg, opt.HandleTimeout)
	}
	// 等待组等待所有请求处理完毕。
	wg.Wait()
	// 关闭编解码器，释放资源。
	_ = cc.Close()
}

// request 结构体，存储一个调用的所有信息。
type request struct {
	h            *codec.Header // 请求头
	argv, replyv reflect.Value // 请求和响应的实际参数
	mtype        *methodType   // 请求相关的方法类型
	svc          *service      // 请求相关的服务
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// 确保 argvi 是一个指针，因为 ReadBody 需要指针参数
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	// 读取请求体
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}

	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()

	// 调用服务方法，记录请求处理完成
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		// 通知处理完成
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			// 通知响应发送完成
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		// 通知响应发送完成
		sent <- struct{}{}
	}()
	if timeout == 0 {
		// 等待调用完成和发送完成
		<-called
		<-sent
		return
	}
	// 使用 select 等待事件发生，或超时
	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		server.sendResponse(cc, req.h, invalidRequest, sending)
	case <-called:
		// 调用完成，等待响应发送完成
		<-sent
	}
}

// Accept 函数用于接受网络连接，并为每个连接启动一个 Goroutine 来处理请求
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		// 为每个连接创建一个新的 Goroutine 来服务
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// Register 函数用于向 RPC 服务器注册服务
func (server *Server) Register(rcvr interface{}) error {
	// 创建服务实例
	s := newService(rcvr)
	// 尝试加载或存储服务到同步映射
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

const (
	connected        = "200 Connected to Gee RPC"
	defaultRPCPath   = "/_geeprc_"
	defaultDebugPath = "/debug/geerpc"
)

// ServeHTTP 用于处理 HTTP 请求，支持 RPC 通过 HTTP 连接传输
func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		// 设置错误的请求方法
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT\n")
		return
	}
	// 获取 HTTP 连接
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		// 记录错误信息
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	// 返回连接成功信息
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	// 使用连接来服务 RPC 请求
	server.ServeConn(conn)
}

// HandleHTTP 函数用于注册 RPC 服务的 HTTP 处理程序，包括默认的 RPC 路径和调试路径
func (server *Server) HandleHTTP() {
	// 注册默认的 RPC 路径处理程序
	http.Handle(defaultRPCPath, server)
	// 注册调试路径处理程序
	http.Handle(defaultDebugPath, debugHTTP{server})
	// 记录调试路径信息
	log.Println("rpc server debug path:", defaultDebugPath)
}

func HandleHTTP() {
	// 默认的服务器处理程序注册 HTTP 处理程序
	DefaultServer.HandleHTTP()
}
