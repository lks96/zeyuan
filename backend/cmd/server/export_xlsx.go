package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"temu-tools/backend/internal/models"
)

const (
	maxExportImageBytes = 8 * 1024 * 1024

	// WPS cellimages.xml 使用 EMU 尺寸，1px ≈ 9525 EMU。
	// 这里不追求和参考表完全一致，只给一个稳定、非 0、打印较稳的默认尺寸。
	wpsEMUPerPixel           = 9525
	wpsDefaultImageWidthPX   = 800
	wpsDefaultImageHeightPX  = 800
	deliveryHeaderRowHeight  = 42
	deliveryDataRowHeight    = 133
	deliveryImageColumnWidth = 17
)

type exportImage struct {
	URL       string
	RelID     string
	Filename  string
	Content   []byte
	MediaType string

	// WPS DISPIMG 使用的图片 ID，例如：
	// =DISPIMG("ID_85689C5742F8408F9D1A718E4E3D801D",1)
	DispID string
}

type exportImageAnchor struct {
	RowIndex int
	Image    *exportImage
}

func buildDeliveryExtractWorkbook(ctx context.Context, batch models.DeliveryExtractBatch) ([]byte, error) {
	images, anchors, _ := collectDeliveryExportImages(ctx, batch.Rows)

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	hasCellImages := len(images) > 0

	files := map[string]string{
		"_rels/.rels":                rootRelationshipsXML(),
		"docProps/app.xml":           appPropertiesXML(),
		"docProps/core.xml":          corePropertiesXML(),
		"xl/workbook.xml":            workbookXML(),
		"xl/_rels/workbook.xml.rels": workbookRelationshipsXML(hasCellImages),
		"xl/styles.xml":              stylesXML(),
		"xl/worksheets/sheet1.xml":   deliveryWorksheetXML(batch, anchors),
		"[Content_Types].xml":        contentTypesXML(images),
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

func collectDeliveryExportImages(ctx context.Context, rows []models.DeliveryExtractRow) ([]*exportImage, []exportImageAnchor, []string) {
	imageByURL := make(map[string]*exportImage)
	images := make([]*exportImage, 0)
	anchors := make([]exportImageAnchor, 0)
	imageErrors := make([]string, 0)

	for index, row := range rows {
		excelRow := index + 3

		imageURL := strings.TrimSpace(row.ProductSkcPicture)
		if imageURL == "" {
			// 没有图片地址不算错误
			continue
		}

		normalizedURL, err := normalizeImageURL(imageURL)
		if err != nil {
			imageErrors = append(imageErrors, fmt.Sprintf("第%d行图片地址不合法：%s，错误：%v", excelRow, imageURL, err))
			continue
		}

		image := imageByURL[normalizedURL]
		if image == nil {
			content, mediaType, extension, err := downloadExportImage(ctx, normalizedURL)
			if err != nil {
				imageErrors = append(imageErrors, fmt.Sprintf("第%d行图片下载失败：%s，错误：%v", excelRow, normalizedURL, err))
				continue
			}

			if len(content) == 0 {
				imageErrors = append(imageErrors, fmt.Sprintf("第%d行图片内容为空：%s", excelRow, normalizedURL))
				continue
			}

			imageIndex := len(images) + 1

			image = &exportImage{
				URL:       normalizedURL,
				RelID:     fmt.Sprintf("rId%d", imageIndex),
				Filename:  fmt.Sprintf("image%d.%s", imageIndex, extension),
				Content:   content,
				MediaType: mediaType,
				DispID:    newWPSDispImageID(imageIndex),
			}

			imageByURL[normalizedURL] = image
			images = append(images, image)
		}

		anchors = append(anchors, exportImageAnchor{
			RowIndex: excelRow,
			Image:    image,
		})
	}

	return images, anchors, imageErrors
}

func normalizeImageURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("empty url")
	}

	// 有些数据可能是 //xxx.com/a.jpg
	if strings.HasPrefix(rawURL, "//") {
		rawURL = "https:" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme == "" {
		return "", fmt.Errorf("missing url scheme, need http or https")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported url scheme: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing host")
	}

	return rawURL, nil
}

func newWPSDispImageID(index int) string {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err == nil {
		return "ID_" + strings.ToUpper(hex.EncodeToString(randomBytes))
	}

	// 极端情况下 crypto/rand 失败，使用时间戳兜底
	return fmt.Sprintf("ID_%032X", time.Now().UnixNano()+int64(index))
}

func downloadExportImage(ctx context.Context, imageURL string) ([]byte, string, string, error) {
	return downloadExportImageWithTimeout(ctx, imageURL, 15*time.Second)
}

func downloadExportImageWithTimeout(ctx context.Context, imageURL string, timeout time.Duration) ([]byte, string, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", "", err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36")
	request.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	request.Header.Set("Connection", "close")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, "", "", err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", "", fmt.Errorf("image request failed: %s", response.Status)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, maxExportImageBytes+1))
	if err != nil {
		return nil, "", "", err
	}

	if len(content) > maxExportImageBytes {
		return nil, "", "", fmt.Errorf("image too large, max size is %d bytes", maxExportImageBytes)
	}

	if len(content) == 0 {
		return nil, "", "", fmt.Errorf("empty image body")
	}

	mediaType := response.Header.Get("Content-Type")
	if semicolon := strings.Index(mediaType, ";"); semicolon >= 0 {
		mediaType = mediaType[:semicolon]
	}
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))

	if mediaType == "" || mediaType == "application/octet-stream" {
		mediaType = http.DetectContentType(content)
	}

	switch mediaType {
	case "image/png":
		return content, mediaType, "png", nil

	case "image/jpeg", "image/jpg":
		return content, "image/jpeg", "jpeg", nil

	case "image/webp":
		// 注意：WPS 的 DISPIMG 不一定稳定支持 webp。
		// 如果你的图片源是 webp，建议后端转成 jpeg/png。
		return nil, "", "", fmt.Errorf("unsupported image type: image/webp, need convert to jpeg/png first")

	case "image/avif":
		return nil, "", "", fmt.Errorf("unsupported image type: image/avif, need convert to jpeg/png first")

	default:
		detected := http.DetectContentType(content)
		return nil, "", "", fmt.Errorf("unsupported image type: header=%s, detected=%s", mediaType, detected)
	}
}

func deliveryWorksheetXML(batch models.DeliveryExtractBatch, anchors []exportImageAnchor) string {
	lastRow := len(batch.Rows) + 2
	if lastRow < 2 {
		lastRow = 2
	}

	imageByRow := make(map[int]*exportImage)
	for _, anchor := range anchors {
		imageByRow[anchor.RowIndex] = anchor.Image
	}

	mergeRanges := deliveryMergeRanges(batch.Rows)

	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	builder.WriteString(fmt.Sprintf(`<dimension ref="A1:G%d"/>`, lastRow))
	builder.WriteString(`<sheetViews><sheetView workbookViewId="0"/></sheetViews>`)

	// 参考 WPS 表格打印效果：图片列稍宽，数据行稍高。
	builder.WriteString(`<sheetFormatPr defaultColWidth="9" defaultRowHeight="14.4"/>`)
	builder.WriteString(`<cols>`)
	builder.WriteString(fmt.Sprintf(`<col min="1" max="1" width="%d" customWidth="1"/>`, deliveryImageColumnWidth))
	builder.WriteString(`<col min="2" max="2" width="6" customWidth="1"/>`)
	builder.WriteString(`<col min="3" max="3" width="22" customWidth="1"/>`)
	builder.WriteString(`<col min="4" max="4" width="8" customWidth="1"/>`)
	builder.WriteString(`<col min="5" max="5" width="14" customWidth="1"/>`)
	builder.WriteString(`<col min="6" max="6" width="15" customWidth="1"/>`)
	builder.WriteString(`<col min="7" max="7" width="12" customWidth="1"/>`)
	builder.WriteString(`</cols>`)

	builder.WriteString(`<sheetData>`)

	builder.WriteString(fmt.Sprintf(`<row r="1" ht="%d" customHeight="1">`, deliveryHeaderRowHeight))
	headers := []string{"产品图片", "产品根数", "产品配置", "套数", "收货仓库", "SKC码", "欧代"}
	for index, header := range headers {
		builder.WriteString(inlineStringCell(cellRef(index+1, 1), 1, header))
	}
	builder.WriteString(`</row>`)

	builder.WriteString(`<row r="2" ht="18" customHeight="1">`)
	builder.WriteString(inlineStringCell("A2", 2, deliveryExportTitle(batch.Rows)))
	builder.WriteString(`</row>`)

	mergedRows := make(map[int]bool)
	for _, mergeRange := range mergeRanges {
		for row := mergeRange[0] + 1; row <= mergeRange[1]; row++ {
			mergedRows[row] = true
		}
	}

	for index, row := range batch.Rows {
		excelRow := index + 3
		builder.WriteString(fmt.Sprintf(`<row r="%d" ht="%d" customHeight="1">`, excelRow, deliveryDataRowHeight))

		image := imageByRow[excelRow]
		if image != nil {
			builder.WriteString(dispImageCell(cellRef(1, excelRow), 3, image.DispID))
		} else {
			builder.WriteString(inlineStringCell(cellRef(1, excelRow), 3, ""))
		}

		builder.WriteString(inlineStringCell(cellRef(2, excelRow), 3, deliveryProductPieces(row)))
		builder.WriteString(inlineStringCell(cellRef(3, excelRow), 4, row.ProductConfig))
		builder.WriteString(numberCell(cellRef(4, excelRow), 3, row.SkcNum))

		if !mergedRows[excelRow] {
			builder.WriteString(inlineStringCell(cellRef(5, excelRow), 3, row.ReceiverName))
		}

		builder.WriteString(inlineStringCell(cellRef(6, excelRow), 3, row.SKC))
		builder.WriteString(inlineStringCell(cellRef(7, excelRow), 3, row.EuRepresentative))
		builder.WriteString(`</row>`)
	}

	builder.WriteString(`</sheetData>`)

	mergeCount := 1 + len(mergeRanges)
	builder.WriteString(fmt.Sprintf(`<mergeCells count="%d">`, mergeCount))
	builder.WriteString(`<mergeCell ref="A2:G2"/>`)
	for _, mergeRange := range mergeRanges {
		builder.WriteString(fmt.Sprintf(`<mergeCell ref="E%d:E%d"/>`, mergeRange[0], mergeRange[1]))
	}
	builder.WriteString(`</mergeCells>`)

	builder.WriteString(`</worksheet>`)
	return builder.String()
}

func deliveryMergeRanges(rows []models.DeliveryExtractRow) [][2]int {
	ranges := make([][2]int, 0)

	start := 0
	for start < len(rows) {
		key := strings.TrimSpace(rows[start].ExpressBatchSn)
		if key == "" {
			start++
			continue
		}

		end := start
		for end+1 < len(rows) && strings.TrimSpace(rows[end+1].ExpressBatchSn) == key {
			end++
		}

		if end > start {
			ranges = append(ranges, [2]int{start + 3, end + 3})
		}

		start = end + 1
	}

	return ranges
}

func deliveryProductPieces(row models.DeliveryExtractRow) string {
	if row.ProductPieces > 0 {
		return fmt.Sprintf("%dP", row.ProductPieces)
	}
	return ""
}

func deliveryExportTitle(rows []models.DeliveryExtractRow) string {
	if len(rows) == 0 {
		return "发货 JSON 提取"
	}

	shopName := strings.TrimSpace(rows[0].ShopName)
	if shopName == "" {
		shopName = strings.TrimSpace(rows[0].SupplierID)
	}

	euRepresentative := strings.TrimSpace(rows[0].EuRepresentative)
	if euRepresentative == "" {
		return shopName
	}

	return shopName + " - " + euRepresentative
}

func dispImageCell(ref string, style int, imageID string) string {
	formula := fmt.Sprintf(`_xlfn.DISPIMG("%s",1)`, imageID)
	value := fmt.Sprintf(`=DISPIMG("%s",1)`, imageID)

	return fmt.Sprintf(
		`<c r="%s" s="%d" t="str"><f>%s</f><v>%s</v></c>`,
		ref,
		style,
		escapeXML(formula),
		escapeXML(value),
	)
}

func inlineStringCell(ref string, style int, text string) string {
	return fmt.Sprintf(
		`<c r="%s" s="%d" t="inlineStr"><is><t>%s</t></is></c>`,
		ref,
		style,
		escapeXML(text),
	)
}

func numberCell(ref string, style int, value int) string {
	return fmt.Sprintf(`<c r="%s" s="%d"><v>%d</v></c>`, ref, style, value)
}

func cellRef(column int, row int) string {
	return string(rune('A'+column-1)) + fmt.Sprintf("%d", row)
}

func escapeXML(value string) string {
	var builder strings.Builder
	_ = xml.EscapeText(&builder, []byte(value))
	return builder.String()
}

func writeZipText(zipWriter *zip.Writer, name string, content string) error {
	return writeZipBytes(zipWriter, name, []byte(content))
}

func writeZipBytes(zipWriter *zip.Writer, name string, content []byte) error {
	writer, err := zipWriter.Create(name)
	if err != nil {
		return err
	}

	_, err = writer.Write(content)
	return err
}

func xmlHeader() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
}

func rootRelationshipsXML() string {
	return xmlHeader() +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>` +
		`</Relationships>`
}

func workbookXML() string {
	return xmlHeader() +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets>` +
		`<sheet name="发货提取" sheetId="1" r:id="rId1"/>` +
		`</sheets>` +
		`<calcPr calcId="0" fullCalcOnLoad="1" forceFullCalc="1"/>` +
		`</workbook>`
}

func workbookRelationshipsXML(hasCellImages bool) string {
	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	builder.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>`)
	builder.WriteString(`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)

	if hasCellImages {
		builder.WriteString(`<Relationship Id="rId3" Type="http://www.wps.cn/officeDocument/2020/cellImage" Target="cellimages.xml"/>`)
	}

	builder.WriteString(`</Relationships>`)
	return builder.String()
}

func cellImagesXML(images []*exportImage) string {
	var builder strings.Builder
	builder.WriteString(xmlHeader())

	builder.WriteString(`<etc:cellImages `)
	builder.WriteString(`xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" `)
	builder.WriteString(`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" `)
	builder.WriteString(`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" `)
	builder.WriteString(`xmlns:etc="http://www.wps.cn/officeDocument/2017/etCustomData">`)

	cx := wpsDefaultImageWidthPX * wpsEMUPerPixel
	cy := wpsDefaultImageHeightPX * wpsEMUPerPixel

	for index, image := range images {
		builder.WriteString(`<etc:cellImage>`)
		builder.WriteString(`<xdr:pic>`)

		builder.WriteString(`<xdr:nvPicPr>`)
		builder.WriteString(fmt.Sprintf(
			`<xdr:cNvPr id="%d" name="%s" descr="%s"/>`,
			index+1,
			escapeXML(image.DispID),
			escapeXML(image.DispID),
		))
		builder.WriteString(`<xdr:cNvPicPr><a:picLocks noChangeAspect="1"/></xdr:cNvPicPr>`)
		builder.WriteString(`</xdr:nvPicPr>`)

		builder.WriteString(`<xdr:blipFill>`)
		builder.WriteString(fmt.Sprintf(`<a:blip r:embed="%s"/>`, image.RelID))
		builder.WriteString(`<a:stretch><a:fillRect/></a:stretch>`)
		builder.WriteString(`</xdr:blipFill>`)

		builder.WriteString(`<xdr:spPr>`)
		builder.WriteString(`<a:xfrm>`)
		builder.WriteString(`<a:off x="0" y="0"/>`)
		builder.WriteString(fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, cx, cy))
		builder.WriteString(`</a:xfrm>`)
		builder.WriteString(`<a:prstGeom prst="rect"><a:avLst/></a:prstGeom>`)
		builder.WriteString(`<a:noFill/>`)
		builder.WriteString(`<a:ln w="9525"><a:noFill/></a:ln>`)
		builder.WriteString(`</xdr:spPr>`)

		builder.WriteString(`</xdr:pic>`)
		builder.WriteString(`</etc:cellImage>`)
	}

	builder.WriteString(`</etc:cellImages>`)
	return builder.String()
}

func cellImagesRelationshipsXML(images []*exportImage) string {
	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)

	for _, image := range images {
		builder.WriteString(fmt.Sprintf(
			`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/%s"/>`,
			image.RelID,
			image.Filename,
		))
	}

	builder.WriteString(`</Relationships>`)
	return builder.String()
}

func contentTypesXML(images []*exportImage) string {
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

func appPropertiesXML() string {
	return xmlHeader() +
		`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">` +
		`<Application>Temu Tools</Application>` +
		`</Properties>`
}

func corePropertiesXML() string {
	now := time.Now().UTC().Format(time.RFC3339)

	return xmlHeader() +
		`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" ` +
		`xmlns:dc="http://purl.org/dc/elements/1.1/" ` +
		`xmlns:dcterms="http://purl.org/dc/terms/" ` +
		`xmlns:dcmitype="http://purl.org/dc/dcmitype/" ` +
		`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<dc:title>Kunsong Grocery</dc:title>` +
		`<dc:creator>Temu Tools</dc:creator>` +
		`<cp:lastModifiedBy>Temu Tools</cp:lastModifiedBy>` +
		`<dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created>` +
		`<dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified>` +
		`</cp:coreProperties>`
}

func stylesXML() string {
	return xmlHeader() +
		`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
		`<fonts count="2">` +
		`<font><sz val="11"/><name val="Microsoft YaHei"/></font>` +
		`<font><b/><sz val="11"/><name val="Microsoft YaHei"/></font>` +
		`</fonts>` +

		`<fills count="4">` +
		`<fill><patternFill patternType="none"/></fill>` +
		`<fill><patternFill patternType="gray125"/></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FF4472C4"/><bgColor indexed="64"/></patternFill></fill>` +
		`<fill><patternFill patternType="solid"><fgColor rgb="FFD9E2F3"/><bgColor indexed="64"/></patternFill></fill>` +
		`</fills>` +

		`<borders count="2">` +
		`<border><left/><right/><top/><bottom/><diagonal/></border>` +
		`<border>` +
		`<left style="thin"><color rgb="FF000000"/></left>` +
		`<right style="thin"><color rgb="FF000000"/></right>` +
		`<top style="thin"><color rgb="FF000000"/></top>` +
		`<bottom style="thin"><color rgb="FF000000"/></bottom>` +
		`<diagonal/>` +
		`</border>` +
		`</borders>` +

		`<cellStyleXfs count="1">` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0"/>` +
		`</cellStyleXfs>` +

		`<cellXfs count="5">` +
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="0"/>` +

		// 样式 1：表头
		`<xf numFmtId="0" fontId="1" fillId="2" borderId="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1">` +
		`<alignment horizontal="center" vertical="center" wrapText="1"/>` +
		`</xf>` +

		// 样式 2：标题行
		`<xf numFmtId="0" fontId="1" fillId="3" borderId="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1">` +
		`<alignment horizontal="center" vertical="center"/>` +
		`</xf>` +

		// 样式 3：居中内容，图片单元格也用这个
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="1" applyBorder="1" applyAlignment="1">` +
		`<alignment horizontal="center" vertical="center" wrapText="1"/>` +
		`</xf>` +

		// 样式 4：左对齐内容
		`<xf numFmtId="0" fontId="0" fillId="0" borderId="1" applyBorder="1" applyAlignment="1">` +
		`<alignment horizontal="left" vertical="center" wrapText="1"/>` +
		`</xf>` +
		`</cellXfs>` +

		`<cellStyles count="1">` +
		`<cellStyle name="Normal" xfId="0" builtinId="0"/>` +
		`</cellStyles>` +
		`</styleSheet>`
}
