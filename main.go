package test

import (
	"context"
	"distributed/registry"
	"distributed/xclient"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// 定义一个名为 Foo 的类型
type Foo int

// 定义一个名为 Args 的结构体，包含两个字段：Num1 和 Num2
type Args struct {
	Num1, Num2 int
}

// 为 Foo 类型实现 Sum 方法。接收者是 Foo 类型，方法入参是一个 Args 类型的指针，返回值是一个 error 类型的指针
func (f Foo) Sum(args Args, reply *int) error {
	// 计算参数的和
	*reply = args.Num1 + args.Num2
	return nil
}

// 为 Foo 类型实现 Sleep 方法。接收者是 Foo 类型，方法入参是一个 Args 类型的指针，返回值是一个 error 类型的指针
func (f Foo) Sleep(args Args, reply *int) error {
	// 根据入参 args 的 Num1 字段的值来决定睡眠时间
	time.Sleep(time.Second * time.Duration(args.Num1))
	// 计算参数的和
	*reply = args.Num1 + args.Num2
	return nil
}

// 启动注册中心的函数。接收者是一个 sync.WaitGroup 类型的指针
func startRegistry(wg *sync.WaitGroup) {
	// 在本地监听一个 TCP 端口 9999
	l, _ := net.Listen("tcp", ":9999")
	// 启动注册表的 HTTP 处理程序，用于处理服务发现
	registry.HandleHTTP()
	// 标记 WaitGroup 任务完成
	wg.Done()
	// 启动 HTTP 服务，用于注册和发现
	_ = http.Serve(l, nil)
}

// 启动服务器的函数。接收者是注册中心的地址和一个 sync.WaitGroup 类型的指针
func startServer(registryAddr string, wg *sync.WaitGroup) {
	// 声明一个 Foo 类型的变量
	var foo Foo
	// 在本地监听一个 TCP 端口，端口号由系统自动选择
	l, _ := net.Listen("tcp", ":0")
	// 创建新的服务器
	server := NewServer()
	// 将 foo 变量注册到服务器上
	_ = server.Register(&foo)
	// 定时向注册中心发送心跳，维持注册信息
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	// 标记 WaitGroup 任务完成
	wg.Done()
	// 服务器开始接收请求
	server.Accept(l)
} ////////////

// 处理 RPC 请求的函数。接收者是 xclient 结构体类型的指针，上下文信息，RPC 调用的类型、服务方法名和入参信息
func foo(xc *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	// 声明一个用于接收返回结果的变量
	var reply int
	// 声明一个用于接收错误信息的变量
	var err error
	// 根据入参 typ 的值进行条件判断
	switch typ {
	case "call":
		// 调用 xc 的 Call 方法，发起一个 RPC 调用。并将结果保存在 reply 变量中
		err = xc.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		// 调用 xc 的 Broadcast 方法，发起一个广播 RPC 调用
		err = xc.Broadcast(ctx, serviceMethod, args, &reply)
	}
	// 判断错误是否发生，如发生则记录错误日志，否则打印成功信息。
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

// call 函数，模拟 RPC 调用过程
func call(registry string) {
	// 创建一个新的发现实例，用于在注册中心发现服务
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	// 创建一个新的 XClient 实例，用于进行 RPC 调用
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	// 使用 defer 语句确保 xc 资源在函数结束时被正确释放
	defer func() { _ = xc.Close() }()
	// 设置一个等待组，用于等待所有的 RPC 调用结束
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		// 增加等待组的计数
		wg.Add(1)
		// 启动一个新的 goroutine，在函数结束时减少等待组的计数
		go func(i int) {
			defer wg.Done()
			// 调用服务端的 Foo.Sum 方法
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	// 等待所有的 RPC 调用结束
	wg.Wait()
}

// broadcast 函数，模拟广播 RPC 调用过程
func broadcast(registry string) {
	// 创建一个新的发现实例，用于在注册中心发现服务
	d := xclient.NewGeeRegistryDiscovery(registry, 0)
	// 创建一个新的 XClient 实例，用于进行 RPC 调用
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	// 使用 defer 语句确保 xc 资源在函数结束时被正确释放
	defer func() { _ = xc.Close() }()
	// 设置一个等待组，用于等待所有的 RPC 调用结束
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		// 增加等待组的计数
		wg.Add(1)
		// 启动一个新的 goroutine，在函数结束时减少等待组的计数
		go func(i int) {
			defer wg.Done()
			// 调用服务端的 Foo.Sum 方法
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// 创建一个上下文，设置了超时时间为 2 秒
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			// 在上下文的超时时间内，尝试调用服务端的 Foo.Sleep 方法
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	// 等待所有的 RPC 调用结束
	wg.Wait()
}

// 主函数，服务的入口点
func main() {
	// 设置日志记录的格式，不显示日期、时间等额外信息
	log.SetFlags(0)
	// 设置注册中心的地址。此例中，它是在本地运行的 HTTP 服务器的地址
	registryAddr := "http://localhost:9999/_geerpc_/registry"
	// 声明一个 WaitGroup 变量，用于等待子协程执行结束
	var wg sync.WaitGroup
	// 增加 WaitGroup 计数，表示有一个后台任务（启动注册中心）要执行
	wg.Add(1)
	// 启动注册中心，在新的 goroutine 中进行，以防止阻塞
	go startRegistry(&wg)
	// 等待注册中心启动完成
	wg.Wait()

	// 等待 1 秒，让注册中心有足够的时间准备好服务发现
	time.Sleep(time.Second)
	// 增加 WaitGroup 计数，表示有两个后台任务（启动服务器 x2）要执行
	wg.Add(2)
	// 启动两个服务器实例，每个实例在新的 goroutine 中运行，以实现高并发处理
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	// 等待所有服务器实例都已启动并运行
	wg.Wait()

	// 再等待 1 秒，确保服务器准备就绪可以接收请求
	time.Sleep(time.Second)
	// 调用 call 函数，它将使用发现机制创建一个 RPC 客户端实例，并模拟对 Foo.Sum 方法的并发调用
	call(registryAddr)
	// 调用 broadcast 函数，它将创建一个 RPC 客户端实例，并模拟对 Foo.Sum 和 Foo.Sleep 方法的广播调用，同时测试调用超时情况
	broadcast(registryAddr)
}
