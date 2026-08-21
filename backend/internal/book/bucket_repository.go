package book

import (
	"bytes"
	"context"
	pbucket "hkorpo/book/internal/platform/bucket"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"iter"
	"strings"

	"github.com/minio/minio-go/v7"
)

type BucketRepo interface {
	UploadString(ctx context.Context, bucketName primitive.Bucket, path, content string, ctype pbucket.ContentType) error
	GetBucketFileAsString(ctx context.Context, bucket primitive.Bucket, path string) (string, error)
	GetBucketFileAsBytes(ctx context.Context, bucket primitive.Bucket, path string) ([]byte, error)
	// GetBucketObjectAsReader streams an object, optionally restricted to a
	// byte range (rangeStart < 0 means "no range: the whole object"). It
	// returns the reader, the size of what the reader will yield, the
	// object's total size (needed for a Content-Range header even when
	// ranged), and the content type.
	GetBucketObjectAsReader(ctx context.Context, bucket primitive.Bucket, path string, rangeStart, rangeEnd int64) (reader io.Reader, size int64, totalSize int64, contentType string, err error)
	GetFilesIteratorOfDir(ctx context.Context, bucket primitive.Bucket, path string) iter.Seq[minio.ObjectInfo]
	UploadReader(ctx context.Context, bucketName primitive.Bucket, path string, content io.ReadCloser, len int64, ctype pbucket.ContentType) error
}

type BucketRepoImpl struct {
	client *minio.Client
}

func NewBucketRepoImpl(client *minio.Client) *BucketRepoImpl {
	return &BucketRepoImpl{
		client: client,
	}
}

func (bc *BucketRepoImpl) GetBucketFileAsString(ctx context.Context, bucket primitive.Bucket, path string) (string, error) {
	obj, err := bc.client.GetObject(ctx, string(bucket), path, minio.GetObjectOptions{})
	if err != nil {
		return "", errorwrapper.Wrap(err)
	}
	defer obj.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, obj)
	if err != nil {
		return "", errorwrapper.Wrap(err)
	}

	return buf.String(), nil
}

func (bc *BucketRepoImpl) GetBucketFileAsBytes(ctx context.Context, bucket primitive.Bucket, path string) ([]byte, error) {
	obj, err := bc.client.GetObject(ctx, string(bucket), path, minio.GetObjectOptions{})
	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	defer obj.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, obj); err != nil {
		return nil, errorwrapper.Wrap(err)
	}
	return buf.Bytes(), nil
}

func (bc *BucketRepoImpl) GetBucketObjectAsReader(ctx context.Context, bucket primitive.Bucket, path string, rangeStart, rangeEnd int64) (io.Reader, int64, int64, string, error) {
	stat, err := bc.client.StatObject(ctx, string(bucket), path, minio.StatObjectOptions{})
	if err != nil {
		return nil, 0, 0, "", errorwrapper.Wrap(err)
	}

	opts := minio.GetObjectOptions{}
	size := stat.Size
	if rangeStart >= 0 {
		end := rangeEnd
		if end <= 0 || end >= stat.Size {
			end = stat.Size - 1
		}
		if err := opts.SetRange(rangeStart, end); err != nil {
			return nil, 0, 0, "", errorwrapper.Wrap(err)
		}
		size = end - rangeStart + 1
	}

	obj, err := bc.client.GetObject(ctx, string(bucket), path, opts)
	if err != nil {
		return nil, 0, 0, "", errorwrapper.Wrap(err)
	}

	return obj, size, stat.Size, stat.ContentType, nil
}

func (bc *BucketRepoImpl) GetFilesIteratorOfDir(ctx context.Context, bucket primitive.Bucket, path string) iter.Seq[minio.ObjectInfo] {
	return bc.client.ListObjectsIter(ctx, string(bucket), minio.ListObjectsOptions{
		Prefix: path,
	})

}

func (bc *BucketRepoImpl) UploadString(ctx context.Context, bucketName primitive.Bucket, path, content string, ctype pbucket.ContentType) error {
	_, err := bc.client.PutObject(
		ctx,
		string(bucketName),
		path,
		strings.NewReader(content),
		int64(len(content)),
		minio.PutObjectOptions{ContentType: string(ctype)},
	)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	return nil
}

func (bc *BucketRepoImpl) UploadReader(ctx context.Context, bucketName primitive.Bucket, path string, content io.ReadCloser, len int64, ctype pbucket.ContentType) error {
	_, err := bc.client.PutObject(
		ctx,
		string(bucketName),
		path,
		content,
		len,
		minio.PutObjectOptions{ContentType: string(ctype)},
	)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	return nil
}
