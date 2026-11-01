// Copyright (c) 2026 Gembit Soultan Shirazi <gembit.soultan@gmail.com>

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ResourceLoader fetches resources referenced by a document: stylesheets,
// images and fonts. ref is already resolved against the document base.
type ResourceLoader interface {
	Load(ctx context.Context, ref string) ([]byte, error)
}

// LoaderFunc adapts a function into a ResourceLoader.
type LoaderFunc func(ctx context.Context, ref string) ([]byte, error)

// Load implements ResourceLoader.
func (f LoaderFunc) Load(ctx context.Context, ref string) ([]byte, error) { return f(ctx, ref) }

// ResolveURL resolves ref against base. Both may be URLs or file paths.
func ResolveURL(base, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "data:") {
		return ref
	}
	if u, err := url.Parse(ref); err == nil && u.Scheme != "" && len(u.Scheme) > 1 {
		return ref
	}
	if base == "" {
		return ref
	}
	if bu, err := url.Parse(base); err == nil && (bu.Scheme == "http" || bu.Scheme == "https" || bu.Scheme == "file") {
		if r, err := url.Parse(ref); err == nil {
			return bu.ResolveReference(r).String()
		}
	}
	if strings.HasPrefix(ref, "/") || filepath.IsAbs(ref) {
		return ref
	}
	dir := base
	if !strings.HasSuffix(base, "/") && !strings.HasSuffix(base, string(filepath.Separator)) {
		if st, err := os.Stat(base); err != nil || !st.IsDir() {
			dir = filepath.Dir(base)
		}
	}
	return filepath.Join(dir, filepath.FromSlash(ref))
}

// DefaultLoader returns the loader described by cfg: data URLs always, local
// files when AllowFile (or from cfg.FS), http(s) only when AllowHTTP.
func DefaultLoader(cfg *Config) ResourceLoader {
	if cfg.Loader != nil {
		return cfg.Loader
	}
	return LoaderFunc(func(ctx context.Context, ref string) ([]byte, error) {
		return defaultLoad(ctx, cfg, ref)
	})
}

func defaultLoad(ctx context.Context, cfg *Config, ref string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limit := cfg.MaxResourceSize
	if limit <= 0 {
		limit = 50 << 20
	}
	if strings.HasPrefix(ref, "data:") {
		return decodeDataURL(ref)
	}
	u, err := url.Parse(ref)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		if !cfg.AllowHTTP {
			return nil, fmt.Errorf("%w: %s (enable WithHTTP)", ErrResourceDenied, ref)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("proteus: GET %s: %s", ref, resp.Status)
		}
		return readLimited(resp.Body, limit, ref)
	}
	if err == nil && u.Scheme == "file" {
		ref = u.Path
	} else if err == nil && u.Scheme != "" && len(u.Scheme) > 1 {
		return nil, fmt.Errorf("%w: scheme %q", ErrResourceDenied, u.Scheme)
	}
	if cfg.FS != nil {
		f, err := cfg.FS.Open(path.Clean(strings.TrimPrefix(filepath.ToSlash(ref), "/")))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return readLimited(f, limit, ref)
	}
	if !cfg.AllowFile {
		return nil, fmt.Errorf("%w: %s (local files disabled)", ErrResourceDenied, ref)
	}
	f, err := os.Open(ref)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLimited(f, limit, ref)
}

func readLimited(r io.Reader, limit int64, ref string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s", ErrResourceTooLarge, ref)
	}
	return data, nil
}

func decodeDataURL(ref string) ([]byte, error) {
	comma := strings.IndexByte(ref, ',')
	if comma < 0 {
		return nil, fmt.Errorf("proteus: malformed data URL")
	}
	meta, payload := ref[5:comma], ref[comma+1:]
	if strings.HasSuffix(meta, ";base64") {
		return base64.StdEncoding.DecodeString(strings.Join(strings.Fields(payload), ""))
	}
	s, err := url.PathUnescape(payload)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}
