package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"temu-tools/backend/internal/models"
)

const (
	productExportImageTimeout     = 12 * time.Second
	productExportImageConcurrency = 16
)

var productConfigPiecePattern = regexp.MustCompile(`(?i)(\d+)\s*p`)

type productExportImageAnchor struct {
	RowIndex int
	Image    *exportImage
}

func buildProductCollectionWorkbook(ctx context.Context, products []models.ProductCollectionProduct) ([]byte, error) {
	images, anchors := collectProductCollectionExportImages(ctx, products)

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	hasCellImages := len(images) > 0

	files := map[string]string{
		"_rels/.rels":                rootRelationshipsXML(),
		"docProps/app.xml":           appPropertiesXML(),
		"docProps/core.xml":          productCollectionCorePropertiesXML(),
		"xl/workbook.xml":            productCollectionWorkbookXML(),
		"xl/_rels/workbook.xml.rels": workbookRelationshipsXML(hasCellImages),
		"xl/styles.xml":              stylesXML(),
		"xl/worksheets/sheet1.xml":   productCollectionWorksheetXML(products, anchors),
		"[Content_Types].xml":        productCollectionContentTypesXML(images),
	}

	if hasCellImages {
		files["xl/cellimages.xml"] = cellImagesXML(images)
		files["xl/_rels/cellimages.xml.rels"] = cellImagesRelationshipsXML(images)
	}

	for name, content := range files {
		if err := writeZipText(zipWriter, name, content); err != nil {
			return nil, err
		}
	}

	for _, image := range images {
		if err := writeZipBytes(zipWriter, "xl/media/"+image.Filename, image.Content); err != nil {
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func collectProductCollectionExportImages(ctx context.Context, products []models.ProductCollectionProduct) ([]*exportImage, []productExportImageAnchor) {
	type imageTask struct {
		RowIndex int
		URL      string
	}

	rowTasks := make([]imageTask, 0)
	uniqueURLs := make([]string, 0)
	seenURLs := make(map[string]bool)
	for index, product := range products {
		imageURL := strings.TrimSpace(product.MainImageURL)
		if imageURL == "" {
			continue
		}

		normalizedURL, err := normalizeImageURL(imageURL)
		if err != nil {
			continue
		}
		rowTasks = append(rowTasks, imageTask{
			RowIndex: index + 2,
			URL:      normalizedURL,
		})
		if !seenURLs[normalizedURL] {
			seenURLs[normalizedURL] = true
			uniqueURLs = append(uniqueURLs, normalizedURL)
		}
	}

	imageByURL := make(map[string]*exportImage)
	var imageMutex sync.Mutex
	var waitGroup sync.WaitGroup
	semaphore := make(chan struct{}, productExportImageConcurrency)

	for _, imageURL := range uniqueURLs {
		imageURL := imageURL
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			content, mediaType, extension, err := downloadCachedExportImage(ctx, imageURL, productExportImageTimeout)
			if err != nil {
				return
			}

			imageMutex.Lock()
			defer imageMutex.Unlock()
			imageIndex := len(imageByURL) + 1
			imageByURL[imageURL] = &exportImage{
				URL:       imageURL,
				RelID:     fmt.Sprintf("rId%d", imageIndex),
				Filename:  fmt.Sprintf("product%d.%s", imageIndex, extension),
				Content:   content,
				MediaType: mediaType,
				DispID:    newWPSDispImageID(imageIndex),
			}
		}()
	}
	waitGroup.Wait()

	images := make([]*exportImage, 0, len(imageByURL))
	anchors := make([]productExportImageAnchor, 0, len(rowTasks))
	appendedImages := make(map[string]bool)
	for _, task := range rowTasks {
		image := imageByURL[task.URL]
		if image == nil {
			continue
		}
		if !appendedImages[task.URL] {
			appendedImages[task.URL] = true
			images = append(images, image)
		}
		anchors = append(anchors, productExportImageAnchor{RowIndex: task.RowIndex, Image: image})
	}

	return images, anchors
}

func productCollectionWorksheetXML(products []models.ProductCollectionProduct, anchors []productExportImageAnchor) string {
	lastRow := len(products) + 1
	if lastRow < 1 {
		lastRow = 1
	}

	imageByRow := make(map[int]*exportImage)
	for _, anchor := range anchors {
		imageByRow[anchor.RowIndex] = anchor.Image
	}

	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	builder.WriteString(fmt.Sprintf(`<dimension ref="A1:H%d"/>`, lastRow))
	builder.WriteString(`<sheetViews><sheetView workbookViewId="0"/></sheetViews>`)
	builder.WriteString(`<sheetFormatPr defaultRowHeight="18"/>`)
	builder.WriteString(`<cols>`)
	builder.WriteString(`<col min="1" max="1" width="16" customWidth="1"/>`)
	builder.WriteString(`<col min="2" max="2" width="18" customWidth="1"/>`)
	builder.WriteString(`<col min="3" max="3" width="9" customWidth="1"/>`)
	builder.WriteString(`<col min="4" max="4" width="34" customWidth="1"/>`)
	builder.WriteString(`<col min="5" max="5" width="12" customWidth="1"/>`)
	builder.WriteString(`<col min="6" max="6" width="12" customWidth="1"/>`)
	builder.WriteString(`<col min="7" max="7" width="16" customWidth="1"/>`)
	builder.WriteString(`<col min="8" max="8" width="20" customWidth="1"/>`)
	builder.WriteString(`</cols>`)
	builder.WriteString(`<sheetData>`)

	headers := []string{"主图", "SKC", "根数", "配置", "价格", "成本", "状态", "创建时间"}
	builder.WriteString(`<row r="1" ht="36" customHeight="1">`)
	for index, header := range headers {
		builder.WriteString(inlineStringCell(cellRef(index+1, 1), 1, header))
	}
	builder.WriteString(`</row>`)

	for index, product := range products {
		excelRow := index + 2
		builder.WriteString(fmt.Sprintf(`<row r="%d" ht="86" customHeight="1">`, excelRow))
		if image := imageByRow[excelRow]; image != nil {
			builder.WriteString(dispImageCell(cellRef(1, excelRow), 3, image.DispID))
		} else {
			builder.WriteString(inlineStringCell(cellRef(1, excelRow), 3, product.MainImageURL))
		}
		builder.WriteString(inlineStringCell(cellRef(2, excelRow), 3, product.ProductSkcID))
		builder.WriteString(inlineStringCell(cellRef(3, excelRow), 3, productPiecesText(product.NumberOfPiecesNew, product.ProductConfig)))
		builder.WriteString(inlineStringCell(cellRef(4, excelRow), 4, product.ProductConfig))
		builder.WriteString(inlineStringCell(cellRef(5, excelRow), 3, centsText(product.SupplierPriceCent)))
		builder.WriteString(inlineStringCell(cellRef(6, excelRow), 3, centsText(product.CostPriceCent)))
		builder.WriteString(inlineStringCell(cellRef(7, excelRow), 3, productCollectionStatusLabel(product.SkcTopStatus)))
		builder.WriteString(inlineStringCell(cellRef(8, excelRow), 3, productCreatedAtText(product.ProductCreatedAt)))
		builder.WriteString(`</row>`)
	}

	builder.WriteString(`</sheetData>`)
	builder.WriteString(`</worksheet>`)
	return builder.String()
}

func productCollectionWorkbookXML() string {
	return xmlHeader() +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets>` +
		`<sheet name="商品采集" sheetId="1" r:id="rId1"/>` +
		`</sheets>` +
		`<calcPr calcId="0" fullCalcOnLoad="1" forceFullCalc="1"/>` +
		`</workbook>`
}

func productCollectionContentTypesXML(images []*exportImage) string {
	defaults := map[string]string{
		"rels": "application/vnd.openxmlformats-package.relationships+xml",
		"xml":  "application/xml",
	}
	for _, image := range images {
		extension := path.Ext(image.Filename)
		if strings.HasPrefix(extension, ".") {
			extension = extension[1:]
		}
		defaults[extension] = image.MediaType
	}

	overrides := map[string]string{
		"/docProps/app.xml":         "application/vnd.openxmlformats-officedocument.extended-properties+xml",
		"/docProps/core.xml":        "application/vnd.openxmlformats-package.core-properties+xml",
		"/xl/workbook.xml":          "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml",
		"/xl/worksheets/sheet1.xml": "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml",
		"/xl/styles.xml":            "application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml",
	}
	if len(images) > 0 {
		overrides["/xl/cellimages.xml"] = "application/vnd.wps-officedocument.cellimage+xml"
	}

	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)

	defaultExtensions := make([]string, 0, len(defaults))
	for extension := range defaults {
		defaultExtensions = append(defaultExtensions, extension)
	}
	sort.Strings(defaultExtensions)
	for _, extension := range defaultExtensions {
		builder.WriteString(fmt.Sprintf(
			`<Default Extension="%s" ContentType="%s"/>`,
			escapeXML(extension),
			escapeXML(defaults[extension]),
		))
	}

	overrideParts := make([]string, 0, len(overrides))
	for part := range overrides {
		overrideParts = append(overrideParts, part)
	}
	sort.Strings(overrideParts)
	for _, part := range overrideParts {
		builder.WriteString(fmt.Sprintf(
			`<Override PartName="%s" ContentType="%s"/>`,
			escapeXML(part),
			escapeXML(overrides[part]),
		))
	}

	builder.WriteString(`</Types>`)
	return builder.String()
}

func productCollectionCorePropertiesXML() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return xmlHeader() +
		`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
		`xmlns:dcterms="http://purl.org/dc/terms/" ` +
		`xmlns:dcmitype="http://purl.org/dc/dcmitype/" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<dc:title>商品采集导出</dc:title>` +
		`<dc:creator>Temu Tools</dc:creator>` +
		`<cp:lastModifiedBy>Temu Tools</cp:lastModifiedBy>` +
		`<dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created>` +
		`<dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified>` +
		`</cp:coreProperties>`
}

func productPiecesText(pieces int, config string) string {
	if pieces > 0 {
		return fmt.Sprintf("%dP", pieces)
	}

	calculatedPieces := productConfigPieces(config)
	if calculatedPieces > 0 {
		return fmt.Sprintf("%dP", calculatedPieces)
	}

	return "0P"
}

func productConfigPieces(config string) int {
	total := 0
	matches := productConfigPiecePattern.FindAllStringSubmatchIndex(config, -1)
	for _, match := range matches {
		if len(match) < 4 || productConfigPieceMatchContinuesWord(config, match[1]) {
			continue
		}
		var pieces int
		if _, err := fmt.Sscanf(config[match[2]:match[3]], "%d", &pieces); err == nil {
			total += pieces
		}
	}
	return total
}

func productConfigPieceMatchContinuesWord(config string, endIndex int) bool {
	if endIndex >= len(config) {
		return false
	}
	next := config[endIndex]
	return (next >= '0' && next <= '9') || (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z')
}

func centsText(cents int) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func productCreatedAtText(createdAt *time.Time) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.Local().Format("2006-01-02 15:04:05")
}

func productCollectionStatusLabel(status int) string {
	switch status {
	case 0:
		return "未发布到站点"
	case 100:
		return "在售中"
	case 200:
		return "已下架/已终止"
	case 300:
		return "已删除"
	default:
		return fmt.Sprintf("未知状态%d", status)
	}
}
