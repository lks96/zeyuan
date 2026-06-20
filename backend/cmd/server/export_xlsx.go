package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"temu-tools/backend/internal/models"
)

type exportImage struct {
	URL       string
	RelID     string
	Filename  string
	Content   []byte
	MediaType string
}

type exportImageAnchor struct {
	RowIndex int
	Image    *exportImage
}

func buildDeliveryExtractWorkbook(ctx context.Context, batch models.DeliveryExtractBatch) ([]byte, error) {
	images, anchors := collectDeliveryExportImages(ctx, batch.Rows)

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	files := map[string]string{
		"_rels/.rels":                         rootRelationshipsXML(),
		"docProps/app.xml":                    appPropertiesXML(),
		"docProps/core.xml":                   corePropertiesXML(),
		"xl/workbook.xml":                     workbookXML(),
		"xl/_rels/workbook.xml.rels":          workbookRelationshipsXML(),
		"xl/styles.xml":                       stylesXML(),
		"xl/worksheets/sheet1.xml":            deliveryWorksheetXML(batch, len(anchors) > 0),
		"xl/worksheets/_rels/sheet1.xml.rels": sheetRelationshipsXML(len(anchors) > 0),
		"xl/drawings/drawing1.xml":            drawingXML(anchors),
		"xl/drawings/_rels/drawing1.xml.rels": drawingRelationshipsXML(images),
		"[Content_Types].xml":                 contentTypesXML(images),
	}

	for name, content := range files {
		if strings.Contains(name, "drawing1") && len(anchors) == 0 {
			continue
		}
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

func collectDeliveryExportImages(ctx context.Context, rows []models.DeliveryExtractRow) ([]*exportImage, []exportImageAnchor) {
	imageByURL := make(map[string]*exportImage)
	images := make([]*exportImage, 0)
	anchors := make([]exportImageAnchor, 0)

	for index, row := range rows {
		imageURL := strings.TrimSpace(row.ProductSkcPicture)
		if imageURL == "" {
			continue
		}

		image := imageByURL[imageURL]
		if image == nil {
			content, mediaType, extension, err := downloadExportImage(ctx, imageURL)
			if err != nil {
				continue
			}
			image = &exportImage{
				URL:       imageURL,
				RelID:     fmt.Sprintf("rId%d", len(images)+1),
				Filename:  fmt.Sprintf("image%d.%s", len(images)+1, extension),
				Content:   content,
				MediaType: mediaType,
			}
			imageByURL[imageURL] = image
			images = append(images, image)
		}

		anchors = append(anchors, exportImageAnchor{
			RowIndex: index + 3,
			Image:    image,
		})
	}

	return images, anchors
}

func downloadExportImage(ctx context.Context, imageURL string) ([]byte, string, string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", "", fmt.Errorf("image request failed: %s", response.Status)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024))
	if err != nil {
		return nil, "", "", err
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
	default:
		return nil, "", "", fmt.Errorf("unsupported image type: %s", mediaType)
	}
}

func deliveryWorksheetXML(batch models.DeliveryExtractBatch, hasDrawing bool) string {
	lastRow := len(batch.Rows) + 2
	if lastRow < 2 {
		lastRow = 2
	}

	mergeRanges := deliveryMergeRanges(batch.Rows)
	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	builder.WriteString(fmt.Sprintf(`<dimension ref="A1:G%d"/>`, lastRow))
	builder.WriteString(`<sheetViews><sheetView workbookViewId="0"/></sheetViews>`)
	builder.WriteString(`<sheetFormatPr defaultRowHeight="18"/>`)
	builder.WriteString(`<cols>`)
	builder.WriteString(`<col min="1" max="1" width="22" customWidth="1"/>`)
	builder.WriteString(`<col min="2" max="2" width="9" customWidth="1"/>`)
	builder.WriteString(`<col min="3" max="3" width="28" customWidth="1"/>`)
	builder.WriteString(`<col min="4" max="4" width="10" customWidth="1"/>`)
	builder.WriteString(`<col min="5" max="5" width="18" customWidth="1"/>`)
	builder.WriteString(`<col min="6" max="6" width="18" customWidth="1"/>`)
	builder.WriteString(`<col min="7" max="7" width="14" customWidth="1"/>`)
	builder.WriteString(`</cols><sheetData>`)

	builder.WriteString(`<row r="1" ht="36" customHeight="1">`)
	headers := []string{"产品图片", "产品根数", "产品配置", "套数", "收货仓库", "SKC码", "欧代"}
	for index, header := range headers {
		builder.WriteString(inlineStringCell(cellRef(index+1, 1), 1, header))
	}
	builder.WriteString(`</row>`)

	builder.WriteString(`<row r="2" ht="18" customHeight="1">`)
	builder.WriteString(inlineStringCell("A2", 2, deliveryExportTitle(batch.Rows)))
	builder.WriteString(`</row>`)

	mergeStartByRow := make(map[int]int)
	mergedRows := make(map[int]bool)
	for _, mergeRange := range mergeRanges {
		mergeStartByRow[mergeRange[0]] = mergeRange[1]
		for row := mergeRange[0] + 1; row <= mergeRange[1]; row++ {
			mergedRows[row] = true
		}
	}

	for index, row := range batch.Rows {
		excelRow := index + 3
		builder.WriteString(fmt.Sprintf(`<row r="%d" ht="118" customHeight="1">`, excelRow))
		builder.WriteString(inlineStringCell(cellRef(1, excelRow), 3, ""))
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

	merges := append([][2]int{{2, 2}}, mergeRanges...)
	if len(merges) > 0 {
		builder.WriteString(fmt.Sprintf(`<mergeCells count="%d">`, len(merges)))
		builder.WriteString(`<mergeCell ref="A2:G2"/>`)
		for _, mergeRange := range mergeRanges {
			builder.WriteString(fmt.Sprintf(`<mergeCell ref="E%d:E%d"/>`, mergeRange[0], mergeRange[1]))
		}
		builder.WriteString(`</mergeCells>`)
	}

	if hasDrawing {
		builder.WriteString(`<drawing r:id="rId1"/>`)
	}
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

func drawingXML(anchors []exportImageAnchor) string {
	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<xdr:wsDr xmlns:xdr="http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	for index, anchor := range anchors {
		row := anchor.RowIndex - 1
		builder.WriteString(`<xdr:twoCellAnchor editAs="oneCell">`)
		builder.WriteString(fmt.Sprintf(`<xdr:from><xdr:col>0</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:from>`, row))
		builder.WriteString(fmt.Sprintf(`<xdr:to><xdr:col>1</xdr:col><xdr:colOff>0</xdr:colOff><xdr:row>%d</xdr:row><xdr:rowOff>0</xdr:rowOff></xdr:to>`, row+1))
		builder.WriteString(`<xdr:pic>`)
		builder.WriteString(fmt.Sprintf(`<xdr:nvPicPr><xdr:cNvPr id="%d" name="Product Image %d"/><xdr:cNvPicPr/></xdr:nvPicPr>`, index+2, index+1))
		builder.WriteString(fmt.Sprintf(`<xdr:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></xdr:blipFill>`, anchor.Image.RelID))
		builder.WriteString(`<xdr:spPr><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></xdr:spPr>`)
		builder.WriteString(`</xdr:pic><xdr:clientData/></xdr:twoCellAnchor>`)
	}
	builder.WriteString(`</xdr:wsDr>`)
	return builder.String()
}

func inlineStringCell(ref string, style int, text string) string {
	return fmt.Sprintf(`<c r="%s" s="%d" t="inlineStr"><is><t>%s</t></is></c>`, ref, style, escapeXML(text))
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
	return xmlHeader() + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
}

func workbookXML() string {
	return xmlHeader() + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="发货提取" sheetId="1" r:id="rId1"/></sheets></workbook>`
}

func workbookRelationshipsXML() string {
	return xmlHeader() + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`
}

func sheetRelationshipsXML(hasDrawing bool) string {
	if !hasDrawing {
		return xmlHeader() + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	}
	return xmlHeader() + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing1.xml"/></Relationships>`
}

func drawingRelationshipsXML(images []*exportImage) string {
	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for _, image := range images {
		builder.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, image.RelID, image.Filename))
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

	var builder strings.Builder
	builder.WriteString(xmlHeader())
	builder.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	for extension, mediaType := range defaults {
		builder.WriteString(fmt.Sprintf(`<Default Extension="%s" ContentType="%s"/>`, extension, mediaType))
	}
	overrides := map[string]string{
		"/docProps/app.xml":         "application/vnd.openxmlformats-officedocument.extended-properties+xml",
		"/docProps/core.xml":        "application/vnd.openxmlformats-package.core-properties+xml",
		"/xl/workbook.xml":          "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml",
		"/xl/worksheets/sheet1.xml": "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml",
		"/xl/styles.xml":            "application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml",
	}
	if len(images) > 0 {
		overrides["/xl/drawings/drawing1.xml"] = "application/vnd.openxmlformats-officedocument.drawing+xml"
	}
	for part, contentType := range overrides {
		builder.WriteString(fmt.Sprintf(`<Override PartName="%s" ContentType="%s"/>`, part, contentType))
	}
	builder.WriteString(`</Types>`)
	return builder.String()
}

func appPropertiesXML() string {
	return xmlHeader() + `<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>Temu Tools</Application></Properties>`
}

func corePropertiesXML() string {
	now := time.Now().UTC().Format(time.RFC3339)
	return xmlHeader() + `<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>发货 JSON 提取导出</dc:title><dc:creator>Temu Tools</dc:creator><cp:lastModifiedBy>Temu Tools</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">` + now + `</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">` + now + `</dcterms:modified></cp:coreProperties>`
}

func stylesXML() string {
	return xmlHeader() + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="2"><font><sz val="11"/><name val="Microsoft YaHei"/></font><font><b/><sz val="11"/><name val="Microsoft YaHei"/></font></fonts><fills count="4"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF4472C4"/><bgColor indexed="64"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFD9E2F3"/><bgColor indexed="64"/></patternFill></fill></fills><borders count="2"><border><left/><right/><top/><bottom/><diagonal/></border><border><left style="thin"><color rgb="FF000000"/></left><right style="thin"><color rgb="FF000000"/></right><top style="thin"><color rgb="FF000000"/></top><bottom style="thin"><color rgb="FF000000"/></bottom><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="5"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/><xf numFmtId="0" fontId="1" fillId="2" borderId="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf><xf numFmtId="0" fontId="1" fillId="3" borderId="1" applyFont="1" applyFill="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center"/></xf><xf numFmtId="0" fontId="0" fillId="0" borderId="1" applyBorder="1" applyAlignment="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf><xf numFmtId="0" fontId="0" fillId="0" borderId="1" applyBorder="1" applyAlignment="1"><alignment horizontal="left" vertical="center" wrapText="1"/></xf></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`
}
