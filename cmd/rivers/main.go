package main

import (
	"errors"
	"flag"
	"net"
	"strings"
	"time"

	"github.com/cybozu-go/cke-tools/rivers"
	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
)

var (
	flgListen          = flag.String("listen", "", "Listen address and port (address:port)")
	flgUpstreams       = flag.String("upstreams", "", "Comma-separated upstream servers (addr1:port1,addr2:port2")
	flgShutdownTimeout = flag.String("shutdown-timeout", "", "Timeout for server shutting-down gracefully (disabled if specified \"0\")")
)

func serve(lns []net.Listener, upstreams []string, cfg rivers.Config) {
	for _, ln := range lns {
		s := rivers.NewServer(upstreams, cfg)
		s.Serve(ln)

	}
	err := cmd.Wait()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}

func listen() ([]net.Listener, error) {
	if len(*flgListen) == 0 {
		return nil, errors.New("--listen is blank")
	}
	ln, err := net.Listen("tcp", *flgListen)
	if err != nil {
		return nil, err
	}
	return []net.Listener{ln}, nil
}

func run() error {
	if len(*flgUpstreams) == 0 {
		return errors.New("--upstreams is blank")
	}
	upstreams := strings.Split(*flgUpstreams, ",")

	var shutdownTimeout time.Duration
	if len(*flgShutdownTimeout) > 0 {
		var err error
		shutdownTimeout, err = time.ParseDuration(*flgShutdownTimeout)
		if err != nil {
			return err
		}
	}

	cfg := rivers.Config{
		ShutdownTimeout: shutdownTimeout,
	}
	g := &cmd.Graceful{
		Listen: listen,
		Serve: func(lsn []net.Listener) {
			serve(lsn, upstreams, cfg)
		},
		ExitTimeout: 30 * time.Second,
	}
	g.Run()
	return cmd.Wait()
}

func main() {
	flag.Parse()
	cmd.LogConfig{}.Apply()

	err := run()
	if err != nil && !cmd.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
