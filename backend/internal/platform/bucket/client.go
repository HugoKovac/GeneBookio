package bucket

import (
	"hkorpo/book/pkg/errorwrapper"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Init(cfg *ConfigBucket) (*minio.Client, error) {
	minioClient, err := minio.New(cfg.ENPOINT, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ACCESS_KEY, cfg.SECRET_KEY, ""),
		Secure: cfg.USE_SSL,
	})
	return minioClient, errorwrapper.Wrap(err)
}
