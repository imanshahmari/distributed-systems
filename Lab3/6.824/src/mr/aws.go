package mr

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	bucket string = "distributed-systems-lab3-uihwarv084q3br86d"
)

func UploadFile(filename string) error {

	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	var timeout time.Duration = 10 * time.Minute

	// Create a new session to connect to s3 using anonymous credentials, as the credential system on aws is not set up properly for this account
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	}))
	svc := s3.New(sess, &aws.Config{Credentials: credentials.AnonymousCredentials})

	// Add a timeout in the context
	ctx := context.Background()
	var cancelFn func()
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}
	defer cancelFn()

	_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   f,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			fmt.Fprintf(os.Stderr, "upload canceled due to timeout, %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "failed to upload object, %v\n", err)
		}
		os.Exit(1)
	}
	if printStuff {
		fmt.Printf("successfully uploaded file to %s/%s\n", bucket, filename)
	}
	return nil
}

func DownloadFile(filename string) ([]byte, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	}))
	svc := s3.New(sess, &aws.Config{Credentials: credentials.AnonymousCredentials})

	// Download the file from s3 bucket
	object, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			fmt.Fprintf(os.Stderr, "download canceled due to timeout, %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "failed to download object, %v\n", err)
		}
		os.Exit(1)
	}

	body, _ := io.ReadAll(object.Body)

	return body, nil
}
