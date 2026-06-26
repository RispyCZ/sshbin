package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	v4signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
)

// S3Storage stores files in an S3-compatible bucket.
// Set AWS_ENDPOINT_URL to use Cloudflare R2 or other S3-compatible services.
type S3Storage struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3 creates an S3Storage with a pre-configured client. Intended for tests.
func NewS3(client *s3.Client, bucket, prefix string) *S3Storage {
	return &S3Storage{client: client, bucket: bucket, prefix: prefix}
}

func newS3Storage(ctx context.Context, u *url.URL) (*S3Storage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	_, customEndpoint := os.LookupEnv("AWS_ENDPOINT_URL")
	return &S3Storage{
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			// Custom endpoints (S3Mock, R2) require path-style addressing;
			// virtual-hosted style only works when DNS resolves the bucket subdomain.
			o.UsePathStyle = customEndpoint
		}),
		bucket: u.Host,
		prefix: strings.TrimPrefix(u.Path, "/"),
	}, nil
}

func (s *S3Storage) objectKey(id, name string) string {
	if s.prefix == "" {
		return id + "/" + name
	}
	return s.prefix + "/" + id + "/" + name
}

// unsignedPayload swaps the dynamic TLS-aware payload signing middleware with
// UnsignedPayload so PutObject works over plain HTTP (e.g. local S3Mock) where
// the SDK would otherwise attempt to SHA256-hash the unseekable pipe body.
// UnsignedPayload is accepted by S3 and S3-compatible stores on both HTTP and HTTPS.
func unsignedPayload(stack *middleware.Stack) error {
	_, err := stack.Finalize.Swap("ComputePayloadHash", &v4signer.UnsignedPayload{})
	return err
}

func (s *S3Storage) Create(ctx context.Context, id, name string) (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	key := s.objectKey(id, name)
	errCh := make(chan error, 1)
	go func() {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   pr,
		}, func(o *s3.Options) {
			o.APIOptions = append(o.APIOptions, unsignedPayload)
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		})
		pr.CloseWithError(err)
		errCh <- err
	}()
	return &s3WriteCloser{pw: pw, errCh: errCh}, nil
}

type s3WriteCloser struct {
	pw    *io.PipeWriter
	errCh chan error
}

func (w *s3WriteCloser) Write(p []byte) (int, error) { return w.pw.Write(p) }

func (w *s3WriteCloser) Close() error {
	w.pw.Close()
	return <-w.errCh
}

func (s *S3Storage) Open(ctx context.Context, id, name string) (io.ReadSeekCloser, error) {
	key := s.objectKey(id, name)
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *s3types.NotFound
		if errors.As(err, &notFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s3ReadSeekCloser{
		ctx:    ctx,
		client: s.client,
		bucket: s.bucket,
		key:    key,
		size:   aws.ToInt64(head.ContentLength),
	}, nil
}

type s3ReadSeekCloser struct {
	ctx    context.Context
	client *s3.Client
	bucket string
	key    string
	size   int64
	offset int64
	rc     io.ReadCloser
}

func (r *s3ReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.offset + offset
	case io.SeekEnd:
		abs = r.size + offset
	default:
		return 0, fmt.Errorf("seek: invalid whence %d", whence)
	}
	if abs < 0 {
		return 0, fmt.Errorf("seek: negative position %d", abs)
	}
	if abs != r.offset {
		if r.rc != nil {
			r.rc.Close()
			r.rc = nil
		}
		r.offset = abs
	}
	return abs, nil
}

func (r *s3ReadSeekCloser) Read(p []byte) (int, error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}
	if r.rc == nil {
		resp, err := r.client.GetObject(r.ctx, &s3.GetObjectInput{
			Bucket: aws.String(r.bucket),
			Key:    aws.String(r.key),
			Range:  aws.String(fmt.Sprintf("bytes=%d-", r.offset)),
		})
		if err != nil {
			return 0, err
		}
		r.rc = resp.Body
	}
	n, err := r.rc.Read(p)
	r.offset += int64(n)
	return n, err
}

func (r *s3ReadSeekCloser) Close() error {
	if r.rc != nil {
		return r.rc.Close()
	}
	return nil
}
