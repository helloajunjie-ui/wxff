package dbkey

// Scanner 在一块内存上找 key。新的扫描策略（比如微信新版本的 heap pattern）
// 只要实现这个接口，注册到 main 即可，不用改主流程。
//
// 内存视图是连续的 []byte（对应进程的一块区域），Scanner 自己决定怎么找候选、
// 怎么用 Matcher 验证、怎么往 found 发 Hit。
type Scanner interface {
	Name() string
	// Scan 扫描一块内存。
	//   view:     当前读到的内存块
	//   dbs:      所有待解 db
	//   saltIdx:  salt→dbIndex 列表
	//   matcher:  候选 key 验证器
	//   found:    找到 Hit 就发这里
	//   remaining: 还没找到 key 的 salt 集合（map[saltString]struct{}），Scanner 自己更新
	Scan(view []byte, dbs []DB, saltIdx map[string][]int,
		matcher *Matcher, found chan<- Hit, remaining map[string]struct{})
}

// Registry 把所有 Scanner 按顺序跑。
type Registry struct {
	scanners []Scanner
}

func NewRegistry(ss ...Scanner) *Registry {
	return &Registry{scanners: ss}
}

func (r *Registry) Scanners() []Scanner { return r.scanners }
