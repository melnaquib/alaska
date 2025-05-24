package walrus_backend

import (
	"bytes"
	// "context"
	"encoding/base64"
	"fmt"
	"github.com/namihq/walrus-go"
	"github.com/seaweedfs/seaweedfs/weed/util"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/google/uuid"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/volume_server_pb"
	"github.com/seaweedfs/seaweedfs/weed/storage/backend"
)

func init() {
	glog.Infof("Init Walrus Backend")
	backend.BackendStorageFactories["walrus"] = &WalrusBackendFactory{}
}

type WalrusBackendFactory struct {
}

func (factory *WalrusBackendFactory) StorageType() backend.StorageType {
	return "walrus"
}

func (factory *WalrusBackendFactory) BuildStorage(configuration backend.StringProperties, configPrefix string, id string) (backend.BackendStorage, error) {
	return newWalrusBackendStorage(configuration, configPrefix, id)
}

type WalrusBackendStorage struct {
	id              string
	remoteName      string
	keyTemplate     *template.Template
	keyTemplateText string
	client          *walrus_go.Client

	//TODO
	aggregatorURLs []string
	publisherURLs  []string
	// TODO
	encryptionKey   []byte // base64 encoded key
	encryptionSuite string // AES256GCM (default) or AES256CBC
}

func newWalrusBackendStorage(configuration backend.StringProperties, configPrefix string, id string) (s *WalrusBackendStorage, err error) {
	s = &WalrusBackendStorage{}
	s.id = id
	s.remoteName = configuration.GetString(configPrefix + "remote_name")

	// s.aggregatorURLs = configuration.GetString(configPrefix + "aggregator_urls").Split(",")
	// s.publisherURLs = configuration.GetString(configPrefix + "publisher_urls").Split(",")

	s.encryptionKey, err = base64.StdEncoding.DecodeString(configuration.GetString(configPrefix + "encryption_key"))
	if err != nil {
		return
	}
	s.encryptionSuite = configuration.GetString(configPrefix + "encryption_suite")

	s.client = walrus_go.NewClient()
	if err != nil {
		return
	}

	// TODO; encryption

	return
}

func (s *WalrusBackendStorage) ToProperties() map[string]string {
	m := make(map[string]string)
	m["remote_name"] = s.remoteName
	if len(s.keyTemplateText) > 0 {
		m["key_template"] = s.keyTemplateText
	}
	return m
}

func keyFromBlobId(id string) (key string) {
	key = id
	return
}

func blobIdFromkey(key string) (id string) {
	id = key
	return
}

func formatKey(key string, storage WalrusBackendStorage) (fKey string, err error) {
	var b bytes.Buffer
	if len(storage.keyTemplateText) == 0 {
		fKey = key
	} else {
		err = storage.keyTemplate.Execute(&b, key)
		if err == nil {
			fKey = b.String()
		}
	}
	return
}

func (s *WalrusBackendStorage) NewStorageFile(key string, tierInfo *volume_server_pb.VolumeInfo) backend.BackendStorageFile {
	f := &WalrusBackendStorageFile{
		backendStorage: s,
		key:            key,
		tierInfo:       tierInfo,
	}

	return f
}

func (s *WalrusBackendStorage) CopyFile(f *os.File, fn func(progressed int64, percentage float32) error) (key string, size int64, err error) {
	randomUuid, err := uuid.NewRandom()
	if err != nil {
		return key, 0, err
	}
	key = randomUuid.String()

	key, err = formatKey(key, *s)
	if err != nil {
		return key, 0, err
	}

	glog.V(1).Infof("copy dat file of %s to remote walrus.%s as %s", f.Name(), s.id, key)

	util.Retry("upload via Walrus", func() error {
		size, err = uploadViaWalrus(s.client, f.Name(), key, fn)
		return err
	})

	return
}

func uploadViaWalrus(client *walrus_go.Client, filename string, key string, fn func(progressed int64, percentage float32) error) (fileSize int64, err error) {
	fileBlobID, err := client.StoreFile(filename, &walrus_go.StoreOptions{Epochs: 5})
	if err != nil {
		log.Fatalf("Error storing file: %v", err)
	}
	fmt.Printf("Stored file blob ID: %s\n", fileBlobID)

	fileInfo, err := os.Stat(filename)
	if err != nil {
		log.Fatalf("Error reading file info: %v", err)
		return
	}
	fileSize = fileInfo.Size()
	progressed := fileSize
	fn(progressed, 100)
	return fileSize, nil
}

func (s *WalrusBackendStorage) DownloadFile(filename string, key string, fn func(progressed int64, percentage float32) error) (size int64, err error) {
	glog.V(1).Infof("download dat file of %s from remote walrus.%s as %s", filename, s.id, key)

	util.Retry("download via Walrus", func() error {
		size, err = downloadViaWalrus(s.client, filename, key, fn)
		return err
	})

	return
}

func downloadViaWalrus(client *walrus_go.Client, filename string, key string, fn func(progressed int64, percentage float32) error) (fileSize int64, err error) {
	// ctx := context.TODO()
	err = client.ReadToFile(key, filename, nil)
	if err != nil {
		log.Fatalf("Error Reading To file: %v", err)
		return
	}
	fileInfo, err := os.Stat(filename)
	if err != nil {
		log.Fatalf("Error reading file info: %v", err)
		return
	}
	written := fileInfo.Size()
	progressed := written
	fn(progressed, 100)
	return fileSize, nil
}

func (s *WalrusBackendStorage) DeleteFile(key string) (err error) {
	glog.V(1).Infof("delete dat file %s from remote", key)

	util.Retry("delete via Walrus", func() error {
		err = deleteViaWalrus(s.client, key)
		return err
	})
	return
}

func deleteViaWalrus(client *walrus_go.Client, key string) (err error) {
	// TODO: sdk does not have
	return nil
}

type WalrusBackendStorageFile struct {
	backendStorage *WalrusBackendStorage
	key            string
	tierInfo       *volume_server_pb.VolumeInfo
}

func (walrusBackendStorageFile WalrusBackendStorageFile) ReadAt(p []byte, off int64) (n int, err error) {
	panic("not implemented")
}

func (walrusBackendStorageFile WalrusBackendStorageFile) WriteAt(p []byte, off int64) (n int, err error) {
	panic("not implemented")
}

func (walrusBackendStorageFile WalrusBackendStorageFile) Truncate(off int64) error {
	panic("not implemented")
}

func (walrusBackendStorageFile WalrusBackendStorageFile) Close() error {
	return nil
}

func (walrusBackendStorageFile WalrusBackendStorageFile) GetStat() (datSize int64, modTime time.Time, err error) {
	metadata, err := walrusBackendStorageFile.backendStorage.client.Head(walrusBackendStorageFile.key)
	if err != nil {
		return
	}
	datSize = metadata.ContentLength
	modTime, err = time.Parse(time.RFC3339, metadata.LastModified)
	return
}

func (walrusBackendStorageFile WalrusBackendStorageFile) Name() string {
	return walrusBackendStorageFile.key
}

func (walrusBackendStorageFile WalrusBackendStorageFile) Sync() error {
	return nil
}
