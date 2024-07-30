package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// GeeRegistry 是一个简单的注册中心，提供以下功能
// 添加一个服务器并接收心跳以保持其存活
// 同步返回所有存活的服务器并删除死服务器
type GeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex // 保护以下内容
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

// New 使用超时设置创建一个注册表实例
func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultGeeRegister = New(defaultTimeout)

// putServer 将服务器添加到注册表中，或者更新其记录的开始时间
func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() // 如果已经存在，更新开始时间以保持存活
	}
}

// aliveServers 返回所有存活的服务器地址列表
func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// 简单起见，服务器地址放在请求头里
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		// 简单起见，服务器地址放在请求头里
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP 在指定的路径上注册一个HTTP处理器来处理GeeRegistry消息
func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

// Heartbeat 是一个辅助函数，用于服务器向注册中心注册或发送心跳
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// 确保在从注册表中删除之前有足够的时间发送心跳
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
