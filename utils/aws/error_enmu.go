package aws

type AwsError string

func (e AwsError) Error() string {
	return string(e)
}

func (e AwsError) Is(error AwsError) bool {
	if e == error {
		return true
	}
	return false
}

var AwsErrorEnum = struct {
	BucketNotExist AwsError
	ObjectNotExist AwsError
	PutObjetFail   AwsError
}{
	BucketNotExist: AwsError("bucket_not_exist"),
	ObjectNotExist: AwsError("object_not_exist"),
	PutObjetFail:   AwsError("put_object_fail"),
}
