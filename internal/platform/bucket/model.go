package bucket

type ConfigBucket struct {
	ACCESS_KEY string `envconfig:"MINIO_ACCESS_KEY"`
	SECRET_KEY string `envconfig:"MINIO_SECRET_KEY"`
	ENPOINT    string `envconfig:"MINIO_ENPOINT"`
	USE_SSL    bool   `envconfig:"MINIO_USE_SSL"`
}
