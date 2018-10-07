package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/acomagu/hackmd-go"
	"github.com/gregjones/httpcache"
)

const sessionID = "xxxx"

var ctx = context.Background()
var hmd = hackmd.NewClient(sessionID, &http.Client{
	Transport: httpcache.NewMemoryCacheTransport(),
})

func main() {
	fuse.Debug = func(msg interface{}) {
		fmt.Fprintln(os.Stderr, msg)
	}

	c, err := fuse.Mount(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	hfs := NewFS()

	fs.Serve(c, hfs)
}

// FS implements the hello world file system.
type FS struct {
	dir *Dir
}

func NewFS() *FS {
	return &FS{
		dir: &Dir{
			bufFile: &BufFile{},
		},
	}
}

func (hfs *FS) Root() (fs.Node, error) {
	return hfs.dir, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct {
	bufFile *BufFile
}

func (*Dir) Attr(ctx context.Context, resp *fuse.Attr) error {
	resp.Mode = os.ModeDir | 0777
	return nil
}

var names map[string]string

func (dir *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := hmd.GetHistory(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, fuse.ENOENT
	}

	names = make(map[string]string)
	var dirents []fuse.Dirent
	for _, entry := range entries {
		var name string
		for i := 0; ; i++ {
			if i == 0 {
				name = entry.Text
			} else {
				name = fmt.Sprintf("%s(%d)", entry.Text, i)
			}

			if _, ok := names[name]; !ok {
				break
			}
		}

		names[name] = entry.ID
		dirents = append(dirents, fuse.Dirent{
			Name: name,
		})
	}

	return dirents, nil
}

func (dir *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	id, ok := names[name]
	if !ok {
		return nil, fuse.ENOENT
	}

	return &NoteFile{
		ID: id,
	}, nil
}

type NoteFile struct {
	ID  string
	buf []byte
}

func (file *NoteFile) Attr(ctx context.Context, resp *fuse.Attr) error {
	resp.Mode = 0666
	return nil
}

func (file *NoteFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	resp.Flags = fuse.OpenDirectIO
	return file, nil
}

func (file *NoteFile) ReadAll(ctx context.Context) ([]byte, error) {
	if err := file.readFromServer(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, fuse.ENOENT
	}

	return file.buf, nil
}

func (file *NoteFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if file.buf == nil {
		if err := file.readFromServer(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return fuse.ENOENT
		}
	}

	file.buf = append(file.buf[0:req.Offset], req.Data...)
	resp.Size = len(req.Data)

	return nil
}

func (file *NoteFile) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	file.writeToServer()
	return fuse.Errno(syscall.ENOSYS)
}

func (file *NoteFile) readFromServer() error {
	r, err := hmd.GetNoteBody(ctx, file.ID)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	file.buf = body

	return nil
}

func (file *NoteFile) writeToServer() error {
	fmt.Println("wrote")
	return nil
}

type BufFile struct {
	buf []byte
}

func (file *BufFile) Attr(ctx context.Context, resp *fuse.Attr) error {
	resp.Mode = 0666
	resp.Size = uint64(len(file.buf))
	return nil
}

func (file *BufFile) ReadAll(ctx context.Context) ([]byte, error) {
	return file.buf, nil
}

func (file *BufFile) Write(req *fuse.WriteRequest, resp *fuse.WriteResponse, ctx context.Context) error {
	file.buf = append(file.buf[0:req.Offset], req.Data...)
	resp.Size = len(req.Data)

	return nil
}
