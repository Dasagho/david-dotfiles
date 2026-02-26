package extractor

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// Extract dispatches to the correct extraction strategy based on the file extension.
// For unknown extensions, the file is treated as a raw binary and copied to dst.
func Extract(srcPath, dstDir string) error {
	name := filepath.Base(srcPath)
	switch {
	case strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz"):
		return extractTar(srcPath, dstDir, "gz")
	case strings.HasSuffix(name, ".tar.xz") || strings.HasSuffix(name, ".txz"):
		return extractTar(srcPath, dstDir, "xz")
	case strings.HasSuffix(name, ".tar.bz2"):
		return extractTar(srcPath, dstDir, "bz2")
	case strings.HasSuffix(name, ".zip"):
		return extractZip(srcPath, dstDir)
	default:
		return copyBinary(srcPath, dstDir)
	}
}

func extractTar(srcPath, dstDir, compression string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader
	switch compression {
	case "gz":
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("open gzip: %w", err)
		}
		defer gr.Close()
		r = gr
	case "bz2":
		r = bzip2.NewReader(f)
	case "xz":
		xr, err := xz.NewReader(f)
		if err != nil {
			return fmt.Errorf("open xz: %w", err)
		}
		r = xr
	}

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		// Sanitize path to prevent path traversal
		target := filepath.Join(dstDir, filepath.Clean("/" + hdr.Name)[1:])
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func extractZip(srcPath, dstDir string) error {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dstDir, filepath.Clean("/" + f.Name)[1:])
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func copyBinary(srcPath, dstDir string) error {
	name := filepath.Base(srcPath)
	dst := filepath.Join(dstDir, name)

	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
