package etcd_backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/etcd/pkg/fileutil"
	"github.com/cybozu-go/etcdutil"
)

const (
	backupSucceed = "backup successfully"
)

type Server struct {
	cfg *Config
}

func NewServer(cfg *Config) *Server {
	return &Server{cfg}
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !strings.HasPrefix(r.URL.Path, "/api/v1/backup") {
		renderError(ctx, w, APIErrNotFound)
		return
	}
	p := r.URL.Path[len("/api/v1/backup"):]
	switch r.Method {
	case http.MethodGet:
		if len(p) == 0 {
			s.handleBackupList(w, r)
			return
		} else if strings.HasPrefix(p, "/") && len(p) > 1 {
			s.handleBackupDownload(w, r, p[1:])
			return
		}
	case http.MethodPost:
		if len(p) == 0 {
			s.handleBackupSave(w, r)
			return
		}
	}
	renderError(ctx, w, APIErrNotFound)
}

func (s Server) handleBackupList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := ioutil.ReadDir(s.cfg.BackupDir)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	var files []string
	for _, i := range info {
		if i.IsDir() {
			continue
		}
		files = append(files, i.Name())
	}

	renderJSON(w, files, http.StatusOK)
}

func (s Server) handleBackupDownload(w http.ResponseWriter, r *http.Request, filename string) {
	ctx := r.Context()
	target := filepath.Join(s.cfg.BackupDir, filename)
	fi, err := os.Stat(target)
	if os.IsNotExist(err) {
		renderError(ctx, w, APIErrNotFound)
		return
	}

	f, err := os.Open(target)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	defer f.Close()
	header := w.Header()
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	header.Set("content-type", contentType)
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
}

func (s Server) handleBackupSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	filename := filepath.Join(s.cfg.BackupDir, snapshotName(now))
	cli, err := etcdutil.NewClient(s.cfg.Etcd)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	defer cli.Close()

	// Take snapshot to temp file
	partpath := filename + ".part"
	defer os.RemoveAll(partpath)

	fp, err := os.OpenFile(partpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileutil.PrivateFileMode)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	var rd io.ReadCloser
	rd, err = cli.Snapshot(ctx)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	_, err = io.Copy(fp, rd)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	err = fileutil.Fsync(fp)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	err = fp.Close()
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	// Rename temp file to expected file name
	err = os.Rename(partpath, filename)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	// Compress snapshot file
	f, err := os.Open(filename)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	gzipName := filename + ".gz"
	zf, err := os.Create(gzipName)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	zw := gzip.NewWriter(zf)

	_, err = io.Copy(zw, f)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}
	defer os.Remove(filename)

	err = zw.Close()
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	fi, err := os.Stat(gzipName)
	if err != nil {
		renderError(ctx, w, InternalServerError(err))
		return
	}

	renderJSON(w, map[string]interface{}{
		"message":  backupSucceed,
		"filename": fi.Name(),
		"filesize": fi.Size(),
	}, http.StatusOK)
}

func snapshotName(date time.Time) string {
	return fmt.Sprintf("snapshot-%s.db", date.Format("20060102_150405"))
}
