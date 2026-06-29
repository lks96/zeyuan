package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"temu-tools/backend/internal/models"
	"temu-tools/backend/internal/store"
)

func (app appServer) handleExportToolPackage(w http.ResponseWriter, r *http.Request, _ models.User) {
	toolID := strings.TrimSpace(r.PathValue("id"))
	if toolID == "" {
		writeError(w, http.StatusBadRequest, "invalid tool package id")
		return
	}

	pkg, err := app.store.GetToolPackage(r.Context(), toolID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "tool package not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tool package")
		return
	}

	data, err := buildToolPackageZip(pkg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to export tool package")
		return
	}

	filename := safeToolPackageFilename(pkg.ID, pkg.Version)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func buildToolPackageZip(pkg models.ToolPackage) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)

	manifest, err := exportedToolManifest(pkg)
	if err != nil {
		return nil, err
	}
	if err := addZipFile(archive, "manifest.json", manifest); err != nil {
		return nil, err
	}

	if err := addZipFile(archive, "README.md", []byte(exportedToolReadme(pkg))); err != nil {
		return nil, err
	}

	if err := addZipFile(archive, "api/routes.json", exportedToolRoutes(pkg)); err != nil {
		return nil, err
	}

	for _, item := range toolPackageExportFiles(pkg.ID) {
		if err := addLocalFileToZip(archive, item.Source, item.Target); err != nil {
			return nil, err
		}
	}

	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type toolPackageExportFile struct {
	Source string
	Target string
}

func toolPackageExportFiles(toolID string) []toolPackageExportFile {
	common := []toolPackageExportFile{
		{Source: "frontend/src/views/ToolsView.vue", Target: "frontend/src/views/ToolsView.vue"},
		{Source: "frontend/src/services/api.ts", Target: "frontend/src/services/api.ts"},
		{Source: "backend/cmd/server/main.go", Target: "backend/cmd/server/main.go"},
		{Source: "backend/internal/store/store.go", Target: "backend/internal/store/store.go"},
		{Source: "backend/internal/models/models.go", Target: "backend/internal/models/models.go"},
	}

	switch toolID {
	case "delivery-json-extract":
		return append([]toolPackageExportFile{
			{Source: "tools/builtin/delivery-json-extract/manifest.json", Target: "source/manifest.json"},
			{Source: "backend/migrations/004_delivery_extract_tool.sql", Target: "migrations/004_delivery_extract_tool.sql"},
			{Source: "backend/migrations/005_delivery_extract_supplier.sql", Target: "migrations/005_delivery_extract_supplier.sql"},
			{Source: "backend/migrations/006_delivery_extract_shop_link.sql", Target: "migrations/006_delivery_extract_shop_link.sql"},
			{Source: "backend/migrations/013_delivery_extract_express_batch.sql", Target: "migrations/013_delivery_extract_express_batch.sql"},
			{Source: "backend/cmd/server/export_xlsx.go", Target: "backend/cmd/server/export_xlsx.go"},
		}, common...)
	case "product-research":
		return append([]toolPackageExportFile{
			{Source: "tools/builtin/product-research/manifest.json", Target: "source/manifest.json"},
			{Source: "backend/migrations/012_product_collection.sql", Target: "migrations/012_product_collection.sql"},
			{Source: "backend/migrations/015_product_collection_supplier_shop_link.sql", Target: "migrations/015_product_collection_supplier_shop_link.sql"},
			{Source: "backend/cmd/server/export_xlsx.go", Target: "backend/cmd/server/export_xlsx.go"},
			{Source: "backend/cmd/server/export_image_cache.go", Target: "backend/cmd/server/export_image_cache.go"},
			{Source: "backend/cmd/server/product_export_xlsx.go", Target: "backend/cmd/server/product_export_xlsx.go"},
		}, common...)
	default:
		return append([]toolPackageExportFile{
			{Source: filepath.ToSlash(filepath.Join("tools", "builtin", toolID, "manifest.json")), Target: "source/manifest.json"},
		}, common...)
	}
}

func exportedToolReadme(pkg models.ToolPackage) string {
	return fmt.Sprintf(`# %s

工具 ID：%s
版本：%s
分类：%s
入口类型：%s

此压缩包由 Temu Tools 工具中心导出，当前是源码级工具包，包含工具清单、接口声明、相关迁移和前后端源码参考。

## 目录说明

- manifest.json：标准工具包清单。
- source/manifest.json：仓库内置工具原始清单。
- api/routes.json：工具使用的后端接口。
- migrations/：工具相关数据库迁移。
- frontend/：当前主系统里承载工具界面的前端源码。
- backend/：当前主系统里承载工具接口和数据访问的后端源码。

## 说明

当前两个工具仍是主系统内置面板，尚未完全拆成独立微前端产物。所以导出的 frontend/ 和 backend/ 是源码参考，不是可直接单独运行的完整服务。后续做安装器时，可以继续把这些文件拆分成独立前端构建产物、安装 SQL 和工具服务。
`, pkg.Name, pkg.ID, pkg.Version, pkg.Category, pkg.EntryType)
}

func exportedToolRoutes(pkg models.ToolPackage) []byte {
	routes := map[string][]string{
		"delivery-json-extract": {
			"GET /api/tools/delivery-extractions",
			"GET /api/tools/delivery-extractions/latest",
			"GET /api/tools/delivery-extractions/latest/export",
			"POST /api/tools/delivery-extractions/import-json",
		},
		"product-research": {
			"GET /api/tools/product-collection/products",
			"GET /api/tools/product-collection/products/export",
			"POST /api/tools/product-collection/import-json",
			"POST /api/tools/product-collection/products/batch-maintenance",
			"PUT /api/tools/product-collection/products/{id}",
		},
	}

	payload := map[string]any{
		"toolId": pkg.ID,
		"routes": routes[pkg.ID],
	}
	if payload["routes"] == nil {
		payload["routes"] = []string{}
	}

	data, _ := json.MarshalIndent(payload, "", "  ")
	return data
}

func exportedToolManifest(pkg models.ToolPackage) ([]byte, error) {
	manifest := map[string]any{
		"toolId":        pkg.ID,
		"toolName":      pkg.Name,
		"version":       pkg.Version,
		"toolDesc":      pkg.Description,
		"toolIcon":      pkg.Icon,
		"toolCategory":  pkg.Category,
		"toolStatus":    pkg.Status,
		"packageType":   pkg.PackageType,
		"entryType":     pkg.EntryType,
		"entryPath":     pkg.EntryPath,
		"panelKey":      pkg.PanelKey,
		"isRecommended": pkg.Recommended,
		"isRemovable":   pkg.Removable,
		"sortOrder":     pkg.SortOrder,
		"permissions":   pkg.Permissions,
	}

	if strings.TrimSpace(pkg.ManifestJSON) != "" && strings.TrimSpace(pkg.ManifestJSON) != "{}" {
		var stored map[string]any
		if err := json.Unmarshal([]byte(pkg.ManifestJSON), &stored); err == nil {
			for key, value := range stored {
				manifest[key] = value
			}
		}
	}

	return json.MarshalIndent(manifest, "", "  ")
}

func addZipFile(archive *zip.Writer, name string, content []byte) error {
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}

func addLocalFileToZip(archive *zip.Writer, source string, target string) error {
	content, err := os.ReadFile(filepath.Join(projectRoot(), filepath.FromSlash(source)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return addZipFile(archive, target, content)
}

func projectRoot() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return "."
	}

	current := workingDir
	for {
		if pathExists(filepath.Join(current, "backend")) && pathExists(filepath.Join(current, "frontend")) {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return workingDir
		}
		current = parent
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func safeToolPackageFilename(toolID string, version string) string {
	cleaner := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	id := cleaner.ReplaceAllString(toolID, "-")
	if id == "" {
		id = "tool"
	}
	version = cleaner.ReplaceAllString(version, "-")
	if version == "" {
		version = "1.0.0"
	}
	return fmt.Sprintf("%s-%s.tool.zip", id, version)
}
