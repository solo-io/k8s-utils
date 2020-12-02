package debugutils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type StorageObject struct {
	Resource io.Reader
	Name     string
}

type StorageClient interface {
	Save(location string, resources ...*StorageObject) error
}

type FileStorageClient struct {
	fs afero.Fs
}

func NewFileStorageClient(fs afero.Fs) *FileStorageClient {
	return &FileStorageClient{fs: fs}
}

func DefaultFileStorageClient() *FileStorageClient {
	return &FileStorageClient{fs: afero.NewOsFs()}
}

func (fsc *FileStorageClient) Save(location string, resources ...*StorageObject) error {
	for _, resource := range resources {
		fileName := filepath.Join(location, resource.Name)
		file, err := fsc.fs.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			return err
		}
		_, err = io.Copy(file, resource.Resource)
		if err != nil {
			return err
		}
		file.Close()
	}
	return nil
}
