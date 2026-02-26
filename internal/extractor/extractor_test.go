package extractor_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"

	"github.com/dsaleh/david-dotfiles/internal/extractor"
)

func TestExtract_tarGz(t *testing.T) {
	// Build a .tar.gz with a single file "mybin"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte("#!/bin/sh\necho hello")
	tw.WriteHeader(&tar.Header{Name: "mybin", Mode: 0755, Size: int64(len(content))})
	tw.Write(content)
	tw.Close()
	gz.Close()

	src, _ := os.CreateTemp("", "test-*.tar.gz")
	src.Write(buf.Bytes())
	src.Close()
	defer os.Remove(src.Name())

	dst, _ := os.MkdirTemp("", "extract-dst-*")
	defer os.RemoveAll(dst)

	if err := extractor.Extract(src.Name(), dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "mybin")); err != nil {
		t.Errorf("mybin not found in dst: %v", err)
	}
}

func TestExtract_zip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("mybin")
	f.Write([]byte("binary"))
	zw.Close()

	src, _ := os.CreateTemp("", "test-*.zip")
	src.Write(buf.Bytes())
	src.Close()
	defer os.Remove(src.Name())

	dst, _ := os.MkdirTemp("", "extract-dst-*")
	defer os.RemoveAll(dst)

	if err := extractor.Extract(src.Name(), dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "mybin")); err != nil {
		t.Errorf("mybin not found in dst: %v", err)
	}
}

func TestExtract_txz(t *testing.T) {
	// Build a .txz (xz-compressed tar) with a single file "mybin"
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatalf("create xz writer: %v", err)
	}
	tw := tar.NewWriter(xw)
	content := []byte("#!/bin/sh\necho hello")
	tw.WriteHeader(&tar.Header{Name: "mybin", Mode: 0755, Size: int64(len(content))})
	tw.Write(content)
	tw.Close()
	xw.Close()

	src, _ := os.CreateTemp("", "test-*.txz")
	src.Write(buf.Bytes())
	src.Close()
	defer os.Remove(src.Name())

	dst, _ := os.MkdirTemp("", "extract-dst-*")
	defer os.RemoveAll(dst)

	if err := extractor.Extract(src.Name(), dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "mybin")); err != nil {
		t.Errorf("mybin not found in dst: %v", err)
	}
}

func TestExtract_rawBinary(t *testing.T) {
	src, _ := os.CreateTemp("", "mybinary-1.2.3-linux-amd64")
	src.Write([]byte("ELF binary content"))
	src.Close()
	defer os.Remove(src.Name())

	dst, _ := os.MkdirTemp("", "extract-dst-*")
	defer os.RemoveAll(dst)

	if err := extractor.Extract(src.Name(), dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, _ := os.ReadDir(dst)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in dst, got %d", len(entries))
	}
	info, _ := entries[0].Info()
	if info.Mode()&0111 == 0 {
		t.Error("raw binary should be executable")
	}
}
