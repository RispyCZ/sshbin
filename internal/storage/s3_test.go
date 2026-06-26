package storage_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/rispycz/sshbin/internal/storage"
)

func newTestS3(t *testing.T) *storage.S3Storage {
	t.Helper()
	faker := gofakes3.New(s3mem.New())
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.UsePathStyle = true
	})
	const bucket = "test-bucket"
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatal(err)
	}
	return storage.NewS3(client, bucket, "")
}

func TestS3Storage_RoundTrip(t *testing.T) {
	st := newTestS3(t)
	ctx := context.Background()

	const content = "hello, s3 world"
	w, err := st.Create(ctx, "id1", "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := st.Open(ctx, "id1", "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestS3Storage_OpenMissing(t *testing.T) {
	st := newTestS3(t)
	ctx := context.Background()

	_, err := st.Open(ctx, "missing", "nope.txt")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestS3Storage_Seek(t *testing.T) {
	st := newTestS3(t)
	ctx := context.Background()

	const content = "0123456789"
	w, err := st.Create(ctx, "id2", "numbers.txt")
	if err != nil {
		t.Fatal(err)
	}
	io.WriteString(w, content)
	w.Close()

	r, err := st.Open(ctx, "id2", "numbers.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(content)) {
		t.Errorf("size: got %d, want %d", size, len(content))
	}

	if _, err := r.Seek(5, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "56789" {
		t.Errorf("got %q, want %q", buf, "56789")
	}
}

func TestS3Storage_WithPrefix(t *testing.T) {
	t.Helper()
	faker := gofakes3.New(s3mem.New())
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.UsePathStyle = true
	})
	const bucket = "prefix-bucket"
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatal(err)
	}
	st := storage.NewS3(client, bucket, "myapp/uploads")

	w, err := st.Create(ctx, "abc", "data.bin")
	if err != nil {
		t.Fatal(err)
	}
	io.WriteString(w, "prefixed")
	w.Close()

	r, err := st.Open(ctx, "abc", "data.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if string(got) != "prefixed" {
		t.Errorf("got %q, want %q", got, "prefixed")
	}
}
