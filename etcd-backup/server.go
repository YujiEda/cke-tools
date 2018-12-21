package etcd_backup

import (
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

}
