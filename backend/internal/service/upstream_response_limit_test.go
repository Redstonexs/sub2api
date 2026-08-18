package service

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestResolveUpstreamResponseReadLimit(t *testing.T) {
	t.Run("use default when config missing", func(t *testing.T) {
		require.Equal(t, defaultUpstreamResponseReadMaxBytes, resolveUpstreamResponseReadLimit(nil))
	})

	t.Run("use configured value", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Gateway.UpstreamResponseReadMaxBytes = 1234
		require.Equal(t, int64(1234), resolveUpstreamResponseReadLimit(cfg))
	})
}

func TestReadUpstreamResponseBodyLimited(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		body, err := readUpstreamResponseBodyLimited(bytes.NewReader([]byte("ok")), 2)
		require.NoError(t, err)
		require.Equal(t, []byte("ok"), body)
	})

	t.Run("exceeds limit", func(t *testing.T) {
		body, err := readUpstreamResponseBodyLimited(bytes.NewReader([]byte("toolong")), 3)
		require.Nil(t, body)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrUpstreamResponseBodyTooLarge))
	})
}

func TestReadUpstreamResponseBodyLimitedCapsDecompressedBodies(t *testing.T) {
	payload := bytes.Repeat([]byte("sub2api"), 128)
	tests := []struct {
		name   string
		reader func(*testing.T, []byte) io.ReadCloser
	}{
		{name: "gzip", reader: gzipResponseLimitReader},
		{name: "brotli", reader: brotliResponseLimitReader},
		{name: "deflate", reader: deflateResponseLimitReader},
		{name: "zstd", reader: zstdResponseLimitReader},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := tc.reader(t, payload)
			t.Cleanup(func() { _ = reader.Close() })

			body, err := readUpstreamResponseBodyLimited(reader, 128)
			require.Nil(t, body)
			require.ErrorIs(t, err, ErrUpstreamResponseBodyTooLarge)
		})
	}
}

func gzipResponseLimitReader(t *testing.T, payload []byte) io.ReadCloser {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	reader, err := gzip.NewReader(&compressed)
	require.NoError(t, err)
	return reader
}

func brotliResponseLimitReader(t *testing.T, payload []byte) io.ReadCloser {
	t.Helper()
	var compressed bytes.Buffer
	writer := brotli.NewWriter(&compressed)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return io.NopCloser(brotli.NewReader(&compressed))
}

func deflateResponseLimitReader(t *testing.T, payload []byte) io.ReadCloser {
	t.Helper()
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	require.NoError(t, err)
	_, err = writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return flate.NewReader(&compressed)
}

func zstdResponseLimitReader(t *testing.T, payload []byte) io.ReadCloser {
	t.Helper()
	var compressed bytes.Buffer
	writer, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	_, err = writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	reader, err := zstd.NewReader(&compressed)
	require.NoError(t, err)
	return reader.IOReadCloser()
}

func TestReadUpstreamResponseBody(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		body, err := ReadUpstreamResponseBody(bytes.NewReader([]byte("ok")), nil, nil, nil)
		require.NoError(t, err)
		require.Equal(t, []byte("ok"), body)
	})

	t.Run("exceeds limit calls onTooLarge", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Gateway.UpstreamResponseReadMaxBytes = 3

		called := false
		onTooLarge := func(_ *gin.Context) { called = true }

		body, err := ReadUpstreamResponseBody(bytes.NewReader([]byte("toolong")), cfg, nil, onTooLarge)
		require.Nil(t, body)
		require.True(t, errors.Is(err, ErrUpstreamResponseBodyTooLarge))
		require.True(t, called)
	})

	t.Run("nil onTooLarge does not panic", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Gateway.UpstreamResponseReadMaxBytes = 3

		body, err := ReadUpstreamResponseBody(bytes.NewReader([]byte("toolong")), cfg, nil, nil)
		require.Nil(t, body)
		require.True(t, errors.Is(err, ErrUpstreamResponseBodyTooLarge))
	})

	t.Run("io error does not call onTooLarge", func(t *testing.T) {
		called := false
		onTooLarge := func(_ *gin.Context) { called = true }

		body, err := ReadUpstreamResponseBody(iotest.ErrReader(errors.New("disk failure")), nil, nil, onTooLarge)
		require.Nil(t, body)
		require.Error(t, err)
		require.False(t, errors.Is(err, ErrUpstreamResponseBodyTooLarge))
		require.False(t, called)
	})
}

func TestReadBufferedUpstreamResponseDelegatesToBoundedRead(t *testing.T) {
	// Verify that AccountTestService.readBufferedUpstreamResponse correctly
	// delegates to readUpstreamResponseBodyLimited, providing bounded-read
	// protection for Grok video status polling.
	svc := &AccountTestService{}

	t.Run("nil reader returns error", func(t *testing.T) {
		body, err := svc.readBufferedUpstreamResponse(nil)
		require.Nil(t, body)
		require.Error(t, err)
	})

	t.Run("small body within limit", func(t *testing.T) {
		body, err := svc.readBufferedUpstreamResponse(bytes.NewReader([]byte(`{"status":"done"}`)))
		require.NoError(t, err)
		require.Contains(t, string(body), `"done"`)
	})

	t.Run("empty body returns empty", func(t *testing.T) {
		body, err := svc.readBufferedUpstreamResponse(bytes.NewReader(nil))
		require.NoError(t, err)
		require.Empty(t, body)
	})

	t.Run("read error propagates", func(t *testing.T) {
		body, err := svc.readBufferedUpstreamResponse(iotest.ErrReader(errors.New("connection reset")))
		require.Nil(t, body)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection reset")
	})
}
