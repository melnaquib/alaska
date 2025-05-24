package walrus

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/namihq/walrus-go"

	"github.com/seaweedfs/seaweedfs/weed/glog"
	"github.com/seaweedfs/seaweedfs/weed/pb/filer_pb"
	"github.com/seaweedfs/seaweedfs/weed/pb/remote_pb"
	"github.com/seaweedfs/seaweedfs/weed/remote_storage"
	"github.com/seaweedfs/seaweedfs/weed/util"
)

func init() {
	remote_storage.RemoteStorageClientMakers["walrus"] = new(walrusRemoteStorageMaker)
}

type walrusRemoteStorageMaker struct{}

func (s walrusRemoteStorageMaker) HasBucket() bool {
	return true
}

func (s walrusRemoteStorageMaker) Make(conf *remote_pb.RemoteConf) (client remote_storage.RemoteStorageClient, err error) {
	client = &walrusRemoteStorageClient{
		conf:   conf,
		client: walrus.NewClient(),
	}
}

type walrusRemoteStorageClient struct {
	conf      *remote_pb.RemoteConf
	client    *walrus.Client
	projectID string
}

var _ = remote_storage.RemoteStorageClient(&walrusRemoteStorageClient{})

func (walrus *walrusRemoteStorageClient) Traverse(loc *remote_pb.RemoteStorageLocation, visitFn remote_storage.VisitFunc) (err error) {
	return
}

func (walrus *walrusRemoteStorageClient) ReadFile(loc *remote_pb.RemoteStorageLocation, offset int64, size int64) (data []byte, err error) {
	// walrus read BLOB_ID --out FILE_PATH

	key := loc.Path[1:]
	rangeReader, readErr := walrus.client.Bucket(loc.Bucket).Object(key).NewRangeReader(context.Background(), offset, size)
	if readErr != nil {
		return nil, readErr
	}
	data, err = io.ReadAll(rangeReader)

	if err != nil {
		return nil, fmt.Errorf("failed to download file %s%s: %v", loc.Bucket, loc.Path, err)
	}

	return
}

func (walrus *walrusRemoteStorageClient) WriteDirectory(loc *remote_pb.RemoteStorageLocation, entry *filer_pb.Entry) (err error) {
	return nil
}

func (walrus *walrusRemoteStorageClient) RemoveDirectory(loc *remote_pb.RemoteStorageLocation) (err error) {
	return nil
}

func (walrus *walrusRemoteStorageClient) WriteFile(loc *remote_pb.RemoteStorageLocation, entry *filer_pb.Entry, reader io.Reader) (remoteEntry *filer_pb.RemoteEntry, err error) {

	key := loc.Path[1:]

	metadata := toMetadata(entry.Extended)
	wc := walrus.client.Bucket(loc.Bucket).Object(key).NewWriter(context.Background())
	wc.Metadata = metadata
	if _, err = io.Copy(wc, reader); err != nil {
		return nil, fmt.Errorf("upload to walrus %s/%s%s: %v", loc.Name, loc.Bucket, loc.Path, err)
	}
	if err = wc.Close(); err != nil {
		return nil, fmt.Errorf("close walrus %s/%s%s: %v", loc.Name, loc.Bucket, loc.Path, err)
	}

	// read back the remote entry
	return walrus.readFileRemoteEntry(loc)

}

func (walrus *walrusRemoteStorageClient) readFileRemoteEntry(loc *remote_pb.RemoteStorageLocation) (*filer_pb.RemoteEntry, error) {
	key := loc.Path[1:]
	attr, err := walrus.client.Bucket(loc.Bucket).Object(key).Attrs(context.Background())

	if err != nil {
		return nil, err
	}

	return &filer_pb.RemoteEntry{
		RemoteMtime: attr.Updated.Unix(),
		RemoteSize:  attr.Size,
		RemoteETag:  attr.Etag,
		StorageName: walrus.conf.Name,
	}, nil

}

func toMetadata(attributes map[string][]byte) map[string]string {
	metadata := make(map[string]string)
	for k, v := range attributes {
		if strings.HasPrefix(k, "X-") {
			continue
		}
		metadata[k] = string(v)
	}
	return metadata
}

func (walrus *walrusRemoteStorageClient) UpdateFileMetadata(loc *remote_pb.RemoteStorageLocation, oldEntry *filer_pb.Entry, newEntry *filer_pb.Entry) (err error) {
	if reflect.DeepEqual(oldEntry.Extended, newEntry.Extended) {
		return nil
	}
	metadata := toMetadata(newEntry.Extended)

	key := loc.Path[1:]

	if len(metadata) > 0 {
		_, err = walrus.client.Bucket(loc.Bucket).Object(key).Update(context.Background(), storage.ObjectAttrsToUpdate{
			Metadata: metadata,
		})
	} else {
		// no way to delete the metadata yet
	}

	return
}
func (walrus *walrusRemoteStorageClient) DeleteFile(loc *remote_pb.RemoteStorageLocation) (err error) {
	key := loc.Path[1:]
	if err = walrus.client.Bucket(loc.Bucket).Object(key).Delete(context.Background()); err != nil {
		return fmt.Errorf("walrus delete %s%s: %v", loc.Bucket, key, err)
	}
	return
}

func (walrus *walrusRemoteStorageClient) ListBuckets() (buckets []*remote_storage.Bucket, err error) {
	// walrus list-blobs --include-expired
	if walrus.projectID == "" {
		return nil, fmt.Errorf("walrus project id or GOOGLE_CLOUD_PROJECT env variable not set")
	}
	iter := walrus.client.Buckets(context.Background(), walrus.projectID)
	for {
		b, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return buckets, err
		}
		buckets = append(buckets, &remote_storage.Bucket{
			Name:      b.Name,
			CreatedAt: b.Created,
		})
	}
	return
}

func (walrus *walrusRemoteStorageClient) CreateBucket(name string) (err error) {
	if walrus.projectID == "" {
		return fmt.Errorf("walrus project id or GOOGLE_CLOUD_PROJECT env variable not set")
	}
	err = walrus.client.Bucket(name).Create(context.Background(), walrus.projectID, &storage.BucketAttrs{})
	if err != nil {
		return fmt.Errorf("create bucket %s: %v", name, err)
	}
	return
}

func (walrus *walrusRemoteStorageClient) DeleteBucket(name string) (err error) {
	err = walrus.client.Bucket(name).Delete(context.Background())
	if err != nil {
		return fmt.Errorf("delete bucket %s: %v", name, err)
	}
	return
}
