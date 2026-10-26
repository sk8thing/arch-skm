package sign

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func decompress(ext string, data []byte) ([]byte, error) {
	switch ext {
	case ".ko":
		return data, nil
	case ".zst":
		return decompressZstd(data)
	case ".xz":
		return decompressXZ(data)
	case ".gz":
		return decompressGzip(data)
	default:
		return nil, fmt.Errorf("unsupported module extension %q", ext)
	}
}

func compress(ext string, data []byte) ([]byte, error) {
	switch ext {
	case ".zst":
		return compressZstd(data)
	case ".xz":
		return compressXZ(data)
	case ".gz":
		return compressGzip(data)
	default:
		return nil, fmt.Errorf("unsupported compressed module extension %q", ext)
	}
}

func isCompressed(ext string) bool {
	return ext == ".zst" || ext == ".xz" || ext == ".gz"
}

func decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("create zstd decoder: %w", err)
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("decompress zstd module: %w", err)
	}

	return decompressed, nil
}

func compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("create zstd encoder: %w", err)
	}
	defer encoder.Close()

	return encoder.EncodeAll(data, nil), nil
}

func decompressXZ(data []byte) ([]byte, error) {
	reader, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create xz reader: %w", err)
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompress xz module: %w", err)
	}

	return decompressed, nil
}

func compressXZ(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	writer, err := xz.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("create xz writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("compress xz module: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close xz writer: %w", err)
	}

	return buf.Bytes(), nil
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompress gzip module: %w", err)
	}

	return decompressed, nil
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("compress gzip module: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}
