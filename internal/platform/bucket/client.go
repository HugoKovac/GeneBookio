package bucket

import (
	"bytes"
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"io"
	"iter"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type BucketClient struct {
	client *minio.Client
}

func Init(ctx context.Context, cfg *ConfigBucket) (*BucketClient, error) {
	minioClient, err := minio.New(cfg.ENPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ACCESS_KEY, cfg.SECRET_KEY, ""),
		Secure: cfg.USE_SSL,
	})
	if err != nil {
		return nil, err
	}

	return &BucketClient{
		client: minioClient,
	}, nil
}

func (bc *BucketClient) GetBucketFileAsString(ctx context.Context, bucket primitive.Bucket, path string) (string, error) {
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

func (bc *BucketClient) GetBucketFileAsBytes(ctx context.Context, bucket primitive.Bucket, path string) ([]byte, error) {
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

func (bc *BucketClient) GetFilesIteratorOfDir(ctx context.Context, bucket primitive.Bucket, path string) iter.Seq[minio.ObjectInfo] {
	return bc.client.ListObjectsIter(ctx, string(bucket), minio.ListObjectsOptions{
		Prefix: path,
	})

}

func (bc *BucketClient) UploadStringAsTextFile(ctx context.Context, bucketName primitive.Bucket, path, content string) error {
	_, err := bc.client.PutObject(
		ctx,
		string(bucketName),
		path,
		strings.NewReader(content),
		int64(len(content)),
		minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"},
	)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	return nil
}
