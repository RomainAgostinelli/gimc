package gimc

import (
	"errors"
	"os"
)

// FileDatasource is the structure used to represent a file as datasource for the cache.
type FileDatasource struct {
	name   string
	opened bool
	file   *os.File
}

// NewFileDatasource creates a new file datasource compatible with the cache datasource.
// Does not open the file, you must call the Open method.
func NewFileDatasource(filename string) *FileDatasource {
	return &FileDatasource{
		name:   filename,
		opened: false,
	}
}

func (fd *FileDatasource) ReadAt(p []byte, off int64) (n int, err error) {
	if fd.opened {
		return fd.file.ReadAt(p, off)
	} else {
		return 0, errors.New("file not opened, please call 'Open' method first")
	}
}

func (fd *FileDatasource) WriteAt(p []byte, off int64) (n int, err error) {
	if fd.opened {
		return fd.file.WriteAt(p, off)
	} else {
		return 0, errors.New("file not opened please call 'Open' method first")
	}
}

func (fd *FileDatasource) Open() error {
	file, err := os.OpenFile(fd.name, os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	fd.file = file
	fd.opened = true
	return nil
}

func (fd *FileDatasource) Close() error {
	err := fd.file.Close()
	if err != nil {
		fd.opened = false
	}
	return err
}
