package debugutils

import (
	"bytes"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var _ = Describe("storage client tests", func() {

	var (
		storageObjects []*StorageObject
	)
	BeforeEach(func() {
		storageObjects = []*StorageObject{
			{
				Resource: bytes.NewBufferString("first"),
				Name:     "first",
			},
			{
				Resource: bytes.NewBufferString("second"),
				Name:     "second",
			},
			{
				Resource: bytes.NewBufferString("third"),
				Name:     "third",
			},
		}
	})

	Context("file client", func() {
		var (
			client *FileStorageClient
			fs     afero.Fs
			tmpd   string
		)

		BeforeEach(func() {
			var err error
			fs = afero.NewOsFs()
			client = NewFileStorageClient(fs)
			tmpd, err = afero.TempDir(fs, "", "")
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			fs.RemoveAll(tmpd)
		})

		It("can store a single file", func() {
			Expect(client.Save(tmpd, storageObjects[0])).NotTo(HaveOccurred())
			fileByt, err := afero.ReadFile(fs, filepath.Join(tmpd, storageObjects[0].Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(fileByt).To(Equal([]byte(storageObjects[0].Name)))
		})

		It("can store multiple files", func() {
			Expect(client.Save(tmpd, storageObjects...)).NotTo(HaveOccurred())
			for _, v := range storageObjects {
				fileByt, err := afero.ReadFile(fs, filepath.Join(tmpd, v.Name))
				Expect(err).NotTo(HaveOccurred())
				Expect(fileByt).To(Equal([]byte(v.Name)))
			}
		})

		It("can store no files", func() {
			Expect(client.Save(tmpd)).NotTo(HaveOccurred())
		})

	})

})
