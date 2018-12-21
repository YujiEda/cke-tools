package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/cke-tools/etcd-backup"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"gopkg.in/yaml.v2"
)

var flgConfig = flag.String("config", "", "path to configuration file")

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()

	if *flgConfig == "" {
		log.ErrorExit(errors.New("usage: etcd-backup -config=<CONFIGFILE>"))
	}

	f, err := os.Open(*flgConfig)
	if err != nil {
		log.ErrorExit(err)
	}
	cfg := etcd_backup.NewConfig()
	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		log.ErrorExit(err)
	}

	server := etcd_backup.NewServer(cfg)
	s := &well.HTTPServer{
		Server: &http.Server{
			Addr:    cfg.Listen,
			Handler: server,
		},
		ShutdownTimeout: 3 * time.Minute,
	}

	err = s.ListenAndServe()
	if err != nil {
		log.ErrorExit(err)
	}

	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		log.ErrorExit(err)
	}
}
