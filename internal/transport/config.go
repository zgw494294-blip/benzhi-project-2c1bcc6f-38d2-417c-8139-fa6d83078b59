package transport

import "flag"

type Config struct {
	Addr      string
	DataDir   string
	SelfCheck bool
	HMACKey   string
}

func ParseConfig() Config {
	addr := flag.String("addr", "127.0.0.1:19081", "监听地址")
	dir := flag.String("data-dir", "./data", "数据目录")
	self := flag.Bool("selfcheck", false, "执行自检")
	flag.Parse()
	return Config{Addr: *addr, DataDir: *dir, SelfCheck: *self}
}
