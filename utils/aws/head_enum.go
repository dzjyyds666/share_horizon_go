package aws

type OssHeadType string

func (o OssHeadType) ToString() string {
	return string(o)
}

var OssHeaders = struct {
	StorageId   OssHeadType
	DirectoryId OssHeadType
	BucketId    OssHeadType

	UploadId  OssHeadType
	UploadKey OssHeadType
}{
	StorageId:   "X-Oss-StorageId",
	DirectoryId: "X-Oss-DirectoryId",
	BucketId:    "X-Oss-BucketId",
	UploadId:    "X-Oss-UploadId",
	UploadKey:   "X-Oss-UploadKey",
}
