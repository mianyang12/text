// 文件首先声明了 xclient 包，随后导入了 errors、math、math/rand、sync 和 time 包。
package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// 定义常量，用于表示选择服务器的不同模式， 如随机选择或轮询选择。
type SelectMode int

const (
	RandomSelect = iota
	RoundRobinSelect
)

type Discovery interface {
	// 刷新发现，意味着从远程注册表更新服务器列表。
	Refresh() error
	// 更新发现的服务器列表。
	Update(servers []string) error
	// 根据用户提供的模式（随机或轮询）获取单个服务器地址。
	Get(mode SelectMode) (string, error)
	// 获取所有已知的服务器地址列表。
	GetAll() ([]string, error)
}

// 确保 MultiServersDiscovery 类型实现了 Discovery 接口。
var _ Discovery = (*MultiServersDiscovery)(nil)

// 定义了 MultiServersDiscovery 结构体，它实现了 Discovery 接口。该结构体用于在没有注册中心的情况下，手动管理多个服务器地址。
type MultiServersDiscovery struct {
	// 用于生成随机数。
	r *rand.Rand
	// 读写锁，用于保护内部服务器列表的并发访问。
	mu sync.RWMutex
	// 存储服务器地址的字符串切片。
	servers []string
	// 用于轮询算法的索引。
	index int
}

// 刷新对于MultiServersDiscovery没有意义，所以忽略该操作，可以根据实际应用需求进行优化。
func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

// Update 方法动态更新 discovery 中的服务器列表，确保了在并发环境下对服务器列表的安全操作。
func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

// Get 方法根据指定的模式返回一个服务器地址，它适用于随机选择和轮询选择。对于无效的模式，会返回错误。
func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n] // 确保服务器列表已更新，因此对 n 取模以确保安全
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

// GetAll 方法返回 discovery 中的所有服务器地址，使用读写锁的读锁以确保在获取服务器列表时没有写操作。
func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// 返回一个 d.servers 的副本，以避免外部修改原始列表
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

// NewMultiServerDiscovery 函数为 discovery 创建一个实例，初始化时会设置随机数生成器，并随机设置轮询算法的索引。
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}
