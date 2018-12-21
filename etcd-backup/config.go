package etcd_backup

import "github.com/cybozu-go/etcdutil"

const (
	defaultBackupDir = "/etcd-backup"
	defaultListen    = "0.0.0.0:8080"
	defaultRotate    = 14
)

func NewConfig() *Config {
	return &Config{
		BackupDir: defaultBackupDir,
		Listen:    defaultListen,
		Rotate:    defaultRotate,
		Etcd:      etcdutil.NewConfig(""),
	}
}

type Config struct {
	BackupDir string           `yaml:"backup-dir,omitempty"`
	Listen    string           `yaml:"listen,omitempty"`
	Rotate    int              `yaml:"rotate,omitempty"`
	Etcd      *etcdutil.Config `yaml:"etcd"`
}
