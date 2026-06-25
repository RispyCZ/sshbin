package sftpd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

type uploadOnlyHandler struct {
	storage storage.Storage
	repo    sharing.Repository
	baseURL string
	stderr  io.Writer
}

// -- FilePut --

type uploadFile struct {
	id       string
	name     string
	wc       io.WriteCloser
	handler  *uploadOnlyHandler
	writeErr error
}

func (f *uploadFile) WriteAt(p []byte, off int64) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	// sftp sends sequential chunks; WriteAt with offset is used but we rely on sequential delivery
	n, err := f.wc.Write(p)
	if err != nil {
		f.writeErr = err
	}
	return n, err
}

func (f *uploadFile) Close() error {
	if err := f.wc.Close(); err != nil {
		return err
	}
	ctx := context.Background()
	s := sharing.Sharing{
		ID:        f.id,
		FileID:    f.id,
		FileName:  f.name,
		CreatedAt: time.Now(),
	}
	if err := f.handler.repo.Create(ctx, s); err != nil {
		return err
	}
	url := sharing.SetupURL(f.handler.baseURL, f.id)
	fmt.Fprint(f.handler.stderr, setupBanner(f.name, url))
	return nil
}

func (h *uploadOnlyHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	id := uuid.New().String()
	name := filepath.Base(r.Filepath)
	wc, err := h.storage.Create(context.Background(), id, name)
	if err != nil {
		return nil, err
	}
	return &uploadFile{id: id, name: name, wc: wc, handler: h}, nil
}

// -- FileGet --

func (h *uploadOnlyHandler) Fileread(*sftp.Request) (io.ReaderAt, error) {
	return nil, syscall.EPERM
}

// -- FileCmd --

func (h *uploadOnlyHandler) Filecmd(r *sftp.Request) error {
	// scp issues fsetstat after upload to copy the source mtime/permissions.
	// Accept it as a no-op; we don't preserve metadata. Deny real mutations.
	if r.Method == "Setstat" {
		return nil
	}
	return syscall.EPERM
}

// -- FileList --

type syntheticFileInfo struct {
	name  string
	isDir bool
}

func (f syntheticFileInfo) Name() string { return f.name }
func (f syntheticFileInfo) Size() int64  { return 0 }
func (f syntheticFileInfo) Mode() os.FileMode {
	if f.isDir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
func (f syntheticFileInfo) ModTime() time.Time { return time.Time{} }
func (f syntheticFileInfo) IsDir() bool        { return f.isDir }
func (f syntheticFileInfo) Sys() interface{}   { return nil }

// isDir reports whether a stat target is the virtual root. pkg/sftp cleans
// paths (stripping trailing slashes) before they reach us, so the only
// distinguishable directory is the root the client lands in. Everything else is
// treated as a regular file (a potential upload target). This is enough for the
// `scp file host:` flow, where the client stats the root before writing.
func isDir(path string) bool {
	return path == "" || path == "." || path == "/"
}

func (h *uploadOnlyHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "Lstat", "Stat":
		fi := syntheticFileInfo{name: filepath.Base(r.Filepath), isDir: isDir(r.Filepath)}
		return singleStat{fi}, nil
	default:
		return nil, syscall.EPERM
	}
}

type singleStat struct{ fi os.FileInfo }

func (s singleStat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	if offset > 0 {
		return 0, io.EOF
	}
	ls[0] = s.fi
	return 1, io.EOF
}

func Handlers(st storage.Storage, repo sharing.Repository, baseURL string, stderr io.Writer) sftp.Handlers {
	h := &uploadOnlyHandler{storage: st, repo: repo, baseURL: baseURL, stderr: stderr}
	return sftp.Handlers{
		FileGet:  h,
		FilePut:  h,
		FileCmd:  h,
		FileList: h,
	}
}
