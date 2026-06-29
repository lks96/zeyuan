package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const exportImageCacheTTL = 7 * 24 * time.Hour

type exportImageCacheMetadata struct {
	URL       string    `json:"url"`
	MediaType string    `json:"mediaType"`
	Extension string    `json:"extension"`
	CachedAt  time.Time `json:"cachedAt"`
}

func downloadCachedExportImage(ctx context.Context, imageURL string, timeout time.Duration) ([]byte, string, string, error) {
	if content, mediaType, extension, ok := readExportImageCache(imageURL); ok {
		return content, mediaType, extension, nil
	}

	content, mediaType, extension, err := downloadExportImageWithTimeout(ctx, imageURL, timeout)
	if err != nil {
		return nil, "", "", err
	}

	_ = writeExportImageCache(imageURL, content, mediaType, extension)
	return content, mediaType, extension, nil
}

func readExportImageCache(imageURL string) ([]byte, string, string, bool) {
	cacheDir, err := exportImageCacheDir()
	if err != nil {
		return nil, "", "", false
	}

	key := exportImageCacheKey(imageURL)
	metadataPath := filepath.Join(cacheDir, key+".json")
	metadataContent, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, "", "", false
	}

	var metadata exportImageCacheMetadata
	if err := json.Unmarshal(metadataContent, &metadata); err != nil {
		_ = os.Remove(metadataPath)
		return nil, "", "", false
	}

	if metadata.URL != imageURL || metadata.Extension == "" || time.Since(metadata.CachedAt) > exportImageCacheTTL {
		removeExportImageCacheFiles(cacheDir, key, metadata.Extension)
		return nil, "", "", false
	}

	imagePath := filepath.Join(cacheDir, key+"."+metadata.Extension)
	content, err := os.ReadFile(imagePath)
	if err != nil || len(content) == 0 {
		removeExportImageCacheFiles(cacheDir, key, metadata.Extension)
		return nil, "", "", false
	}

	return content, metadata.MediaType, metadata.Extension, true
}

func writeExportImageCache(imageURL string, content []byte, mediaType string, extension string) error {
	if imageURL == "" || len(content) == 0 || extension == "" {
		return nil
	}

	cacheDir, err := exportImageCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	key := exportImageCacheKey(imageURL)
	imagePath := filepath.Join(cacheDir, key+"."+extension)
	imageTempPath := imagePath + ".tmp"
	if err := os.WriteFile(imageTempPath, content, 0644); err != nil {
		return err
	}
	_ = os.Remove(imagePath)
	if err := os.Rename(imageTempPath, imagePath); err != nil {
		_ = os.Remove(imageTempPath)
		return err
	}

	metadata := exportImageCacheMetadata{
		URL:       imageURL,
		MediaType: mediaType,
		Extension: extension,
		CachedAt:  time.Now(),
	}
	metadataContent, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	metadataPath := filepath.Join(cacheDir, key+".json")
	metadataTempPath := metadataPath + ".tmp"
	if err := os.WriteFile(metadataTempPath, metadataContent, 0644); err != nil {
		return err
	}
	_ = os.Remove(metadataPath)
	if err := os.Rename(metadataTempPath, metadataPath); err != nil {
		_ = os.Remove(metadataTempPath)
		return err
	}

	return nil
}

func exportImageCacheDir() (string, error) {
	candidates := []string{
		filepath.Join("backend", ".cache", "export-images"),
		filepath.Join(".cache", "export-images"),
		filepath.Join("..", "backend", ".cache", "export-images"),
	}

	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		parent := filepath.Dir(filepath.Dir(absolute))
		if _, err := os.Stat(parent); err == nil {
			return absolute, nil
		}
	}

	return "", fmt.Errorf("cannot resolve export image cache directory")
}

func exportImageCacheKey(imageURL string) string {
	sum := sha256.Sum256([]byte(imageURL))
	return hex.EncodeToString(sum[:])
}

func removeExportImageCacheFiles(cacheDir string, key string, extension string) {
	_ = os.Remove(filepath.Join(cacheDir, key+".json"))
	if extension != "" {
		_ = os.Remove(filepath.Join(cacheDir, key+"."+extension))
	}
}
