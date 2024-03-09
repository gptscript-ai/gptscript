package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/mholt/archiver/v4"
)

func Extract(ctx context.Context, downloadURL, digest, targetDir string) error {
	if err := os.RemoveAll(targetDir); err != nil {
		return nil
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", targetDir, err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// NOTE: Because I'm validating the hash at the same time as extracting this isn't actually secure.
	// Security is still assumed the source is trusted.  Which is bad and should be changed.
	digester := sha256.New()
	input := io.TeeReader(resp.Body, digester)

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return err
	}

	format, input, err := archiver.Identify(filepath.Base(parsedURL.Path), input)
	if err != nil {
		return err
	}

	ex, ok := format.(archiver.Extractor)
	if !ok {
		return fmt.Errorf("failed to detect proper archive for extraction from %s got: %v", downloadURL, ex)
	}

	err = ex.Extract(ctx, input, nil, func(_ context.Context, f archiver.File) error {
		target := filepath.Join(targetDir, f.NameInArchive)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if f.IsDir() {
			return os.MkdirAll(target, f.Mode())
		} else if f.LinkTarget != "" {
			return os.Symlink(f.LinkTarget, target)
		}
		targetFile, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("create %s: %w", target, err)
		}
		arc, err := f.Open()
		if err != nil {
			return err
		}
		if _, err := io.Copy(targetFile, arc); err != nil {
			return err
		}
		if err := arc.Close(); err != nil {
			return err
		}
		if err := targetFile.Close(); err != nil {
			return err
		}
		if err := os.Chmod(target, f.Mode()); err != nil {
			return err
		}
		if err := os.Chtimes(target, time.Time{}, f.ModTime()); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	resultDigest := digester.Sum(nil)
	resultDigestString := hex.EncodeToString(resultDigest[:])

	if resultDigestString != digest {
		return fmt.Errorf("downloaded %s and expected digest %s but got %s", downloadURL, digest, resultDigestString)
	}

	return nil
}
