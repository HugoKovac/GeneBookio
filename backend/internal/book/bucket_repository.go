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

func (bc *BucketRepoImpl) GetBucketObjectAsReader(ctx context.Context, bucket primitive.Bucket, path string) (io.Reader, int64, string, error) {
	obj, err := bc.client.GetObject(ctx, string(bucket), path, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, "", errorwrapper.Wrap(err)
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, "", err
	}
	return obj, stat.Size, stat.ContentType, nil
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
