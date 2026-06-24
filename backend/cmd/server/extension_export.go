package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"temu-tools/backend/internal/models"
)

const extensionArchiveFilename = "temu-seller-sync-extension.zip"

func (app appServer) handleExportExtensionArchive(w http.ResponseWriter, r *http.Request, _ models.User) {
	apiBase := extensionAPIBaseFromRequest(r)
	data, err := buildExtensionZip(apiBase)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build extension archive")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, extensionArchiveFilename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func buildExtensionZip(apiBase string) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)

	sourceDir := filepath.Join(projectRoot(), "chrome-extension")
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == ".DS_Store" || entry.Name() == "Thumbs.db" {
			continue
		}
		target := entry.Name()
		if target == "config.js" {
			if err := addZipFile(archive, target, []byte(extensionConfigJS(apiBase))); err != nil {
				return nil, err
			}
			continue
		}
		content, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := addZipFile(archive, target, content); err != nil {
			return nil, err
		}
	}

	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func extensionAPIBaseFromRequest(r *http.Request) string {
	if apiBase := normalizeExtensionAPIBase(r.URL.Query().Get("apiBase")); apiBase != "" {
		return apiBase
	}
	if origin := normalizeExtensionAPIBase(r.Header.Get("Origin")); origin != "" {
		return strings.TrimRight(origin, "/") + "/api"
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		if parsed, err := url.Parse(referer); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.Scheme + "://" + parsed.Host + "/api"
		}
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto == "http" || forwardedProto == "https" {
		scheme = forwardedProto
	}
	if host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); host != "" {
		return scheme + "://" + host + "/api"
	}
	return scheme + "://" + r.Host + "/api"
}

func normalizeExtensionAPIBase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return strings.TrimRight(value, "/")
}

func extensionConfigJS(apiBase string) string {
	return fmt.Sprintf("globalThis.TEMU_TOOLS_EXTENSION_CONFIG = {\n  apiBase: %q,\n}\n", apiBase)
}
