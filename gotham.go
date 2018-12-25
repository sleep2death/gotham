package gotham

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/sleep2death/evio"
)

var log = logrus.New()

// Config of the server
type Config struct {
	Network   string // tcp tcp4 tcp6 udp udp4 udp6 unix, default is tcp
	Address   string // the address of the server, default is localhost
	Port      int    // the port of the server, default is 8101
	ReusePort bool   // the SO_REUSEPORT socket option, not available on windows: [details] https://stackoverflow.com/questions/14388706/socket-options-so-reuseaddr-and-so-reuseport-how-do-they-differ-do-they-mean-t
	Stdlib    bool   // use the standard net package or not, default is false(using syscall to call system network functions

	NumLoops    int              // multithread if the value is larger than 1, default is based on your cpu cores
	LoadBalance evio.LoadBalance // loadbalance strategy, not work if the NumLoops is less than 1
}

func (config *Config) getAddr() (addr string) {
	scheme := config.Network
	if config.Stdlib {
		scheme += "-net"
	}
	addr = fmt.Sprintf("%s://%s:%d?reusePort=%t", scheme, config.Address, config.Port, config.ReusePort)
	return
}

// Default get the default server
func Default() *Config {
	log.Level = logrus.DebugLevel

	config := &Config{
		Network:   "tcp",
		Address:   "localhost",
		Port:      8202,
		ReusePort: true,
		Stdlib:    false,

		NumLoops:    runtime.NumCPU(),
		LoadBalance: evio.Random,
	}

	return config
}

// Serve start to serve
func Serve(config *Config) (err error) {

	events := initEvents(config.NumLoops)
	address := config.getAddr()

	err = evio.Serve(events, address)
	return
}
