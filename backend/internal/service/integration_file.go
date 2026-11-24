package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces the file boundaries of IntegrationService: CSV/XLSX
// parsing (parseFile/validateHeaders), the ordered raw-row JSON that keeps
// header order like Java's LinkedHashMap, and the CSV/XLSX export rendering
// (renderExport/safeCell). No macros or formulas are ever executed.

// rawRecord is one imported file row. keys preserves the header order so the
// stored raw JSON and the derived digests stay faithful to the Java backend.
type rawRecord struct {
	keys     []string
	vals     map[string]string
	formula  bool
	sheetRef string
}

// json renders the row as an insertion-ordered JSON object (Jackson does not
// HTML-escape, hence store.JSONString).
func (rr rawRecord) json() (string, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range rr.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kj, err := store.JSONString(k)
		if err != nil {
			return "", err
		}
		vj, err := store.JSONString(rr.vals[k])
		if err != nil {
			return "", err
		}
		b.WriteString(kj)
		b.WriteByte(':')
		b.WriteString(vj)
	}
	b.WriteByte('}')
	return b.String(), nil
}

// skipJSONValue consumes a whole value whose first token was already read.
func skipJSONValue(dec *json.Decoder, tok json.Token) error {
	if d, ok := tok.(json.Delim); ok && (d == '{' || d == '[') {
		depth := 1
		for depth > 0 {
			t, err := dec.Token()
			if err != nil {
				return fmt.Errorf("import row json: %w", err)
			}
			if dd, ok := t.(json.Delim); ok {
				switch dd {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
			}
		}
	}
	return nil
}

// rawJSONFromEnvelope extracts the verbatim "raw" object text from a decrypted
// import row envelope, preserving the original column order.
func rawJSONFromEnvelope(plain string) (string, error) {
	dec := json.NewDecoder(strings.NewReader(plain))
	tok, err := dec.Token()
	if err != nil {
		return "", fmt.Errorf("import row json: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return "", fmt.Errorf("import row json: not an object")
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("import row json: %w", err)
		}
		key, _ := kt.(string)
		start := dec.InputOffset()
		vt, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("import row json: %w", err)
		}
		if key == "raw" {
			if err := skipJSONValue(dec, vt); err != nil {
				return "", err
			}
			end := dec.InputOffset()
			if start > end || end > int64(len(plain)) {
				return "", fmt.Errorf("import row json: bad raw span")
			}
			return strings.TrimLeft(plain[start:end], ": \t\r\n"), nil
		}
		if err := skipJSONValue(dec, vt); err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("import row json: raw missing")
}

// rawKeys lists the top-level keys of a raw row object in stored order.
func rawKeys(rawText string) ([]string, error) {
	dec := json.NewDecoder(strings.NewReader(rawText))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("raw json: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return []string{}, nil
	}
	keys := []string{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("raw json: %w", err)
		}
		k, _ := kt.(string)
		keys = append(keys, k)
		var skip any
		if err := dec.Decode(&skip); err != nil {
			return nil, fmt.Errorf("raw json: %w", err)
		}
	}
	return keys, nil
}

// validateHeaders enforces unique non-blank headers, at most 200 columns.
// Header length is measured in UTF-16 code units like Java String.length().
func validateHeaders(headers []string) error {
	seen := map[string]bool{}
	ok := len(headers) > 0 && len(headers) <= 200
	for _, h := range headers {
		if strings.TrimSpace(h) == "" || len(utf16.Encode([]rune(h))) > 200 || seen[h] {
			ok = false
		}
		seen[h] = true
	}
	return errcode.Require(ok, 422, "INVALID_FILE", "表头须唯一非空，最多200列")
}

// parseImportFile parses CSV/XLSX bytes into ordered rows (parseFile).
func parseImportFile(data []byte, name string) ([]rawRecord, error) {
	var rows []rawRecord
	var err error
	switch {
	case strings.HasSuffix(name, ".csv"):
		rows, err = parseCSVRecords(data)
	case strings.HasSuffix(name, ".xlsx"):
		rows, err = parseXLSXFile(data)
	default:
		return nil, errcode.New(422, "INVALID_FILE", "仅接受CSV或XLSX文件")
	}
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(len(rows) > 0, 422, "INVALID_FILE", "名单不能为空"); err != nil {
		return nil, err
	}
	return rows, nil
}

// parseCSVRecords reproduces CSVFormat.DEFAULT with auto header, no duplicate
// header names and consistent record widths.
func parseCSVRecords(data []byte) ([]rawRecord, error) {
	bom := string([]byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(data) || strings.Contains(string(data), bom) {
		return nil, errcode.New(422, "INVALID_ENCODING", "CSV须使用UTF-8编码")
	}
	text := strings.TrimPrefix(string(data), bom)
	reader := csv.NewReader(strings.NewReader(text))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err == io.EOF {
		header = []string{}
	} else if err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	} else {
		// commons-csv rejects duplicate headers while parsing, before the
		// header validation message applies.
		seen := map[string]bool{}
		for _, h := range header {
			if seen[h] {
				return nil, fmt.Errorf("csv duplicate header %q", h)
			}
			seen[h] = true
		}
	}
	if err := validateHeaders(header); err != nil {
		return nil, err
	}
	rows := []rawRecord{}
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv record: %w", err)
		}
		if len(rec) != len(header) {
			return nil, errcode.New(422, "INVALID_FILE", "CSV列数不一致")
		}
		vals := make(map[string]string, len(header))
		for i, h := range header {
			vals[h] = rec[i]
		}
		rows = append(rows, rawRecord{keys: header, vals: vals})
		if len(rows) > 10000 {
			return nil, errcode.New(422, "INVALID_FILE", "最多10000行")
		}
	}
	return rows, nil
}

// parseXLSXFile runs the zip safety pre-scan then reads the first non-empty
// sheet exactly like the POI path (macro/size limits, header row discovery,
// formula rows flagged instead of executed).
func parseXLSXFile(data []byte) ([]rawRecord, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("xlsx zip: %w", err)
	}
	remaining := int64(50 * 1024 * 1024)
	for _, f := range zr.File {
		if strings.Contains(strings.ToLower(f.Name), "vbaproject") {
			return nil, errcode.New(422, "INVALID_FILE", "不接受宏文件")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("xlsx entry: %w", err)
		}
		n, err := io.Copy(io.Discard, io.LimitReader(rc, remaining+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("xlsx entry: %w", err)
		}
		remaining -= n
		if remaining < 0 {
			return nil, errcode.New(422, "INVALID_FILE", "解压文件过大")
		}
	}
	sheets, styles, shared, err := readXLSX(zr)
	if err != nil {
		return nil, err
	}
	return parseXLSXRecords(sheets, styles, shared)
}

type xlsxCell struct {
	col   int
	kind  string // n, s, str, b, e, inlineStr, f (formula)
	text  string
	style int
}

type xlsxSheet struct {
	name   string
	rows   map[int][]xlsxCell
	maxRow int // 0-based index of the last present row, -1 when empty
}

func (sh xlsxSheet) cellAt(row, col int) *xlsxCell {
	for i := range sh.rows[row] {
		if sh.rows[row][i].col == col {
			return &sh.rows[row][i]
		}
	}
	return nil
}

type xlsxStyleInfo struct {
	numFmts map[int]string
	cellXfs []int
}

func readXLSXPart(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("xlsx part %s: %w", name, err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("xlsx part %s missing", name)
}

func readXLSX(zr *zip.Reader) ([]xlsxSheet, xlsxStyleInfo, []string, error) {
	styles := xlsxStyleInfo{numFmts: map[int]string{}}
	if data, err := readXLSXPart(zr, "xl/styles.xml"); err == nil {
		var doc struct {
			NumFmts []struct {
				ID   int    `xml:"numFmtId,attr"`
				Code string `xml:"formatCode,attr"`
			} `xml:"numFmts>numFmt"`
			CellXfs []struct {
				NumFmtID int `xml:"numFmtId,attr"`
			} `xml:"cellXfs>xf"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, styles, nil, fmt.Errorf("xlsx styles: %w", err)
		}
		for _, nf := range doc.NumFmts {
			styles.numFmts[nf.ID] = nf.Code
		}
		for _, xf := range doc.CellXfs {
			styles.cellXfs = append(styles.cellXfs, xf.NumFmtID)
		}
	}
	shared := []string{}
	if data, err := readXLSXPart(zr, "xl/sharedStrings.xml"); err == nil {
		shared, err = readXLSXSharedStrings(data)
		if err != nil {
			return nil, styles, nil, err
		}
	}
	wbData, err := readXLSXPart(zr, "xl/workbook.xml")
	if err != nil {
		return nil, styles, nil, err
	}
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := xml.Unmarshal(wbData, &wb); err != nil {
		return nil, styles, nil, fmt.Errorf("xlsx workbook: %w", err)
	}
	targets := map[string]string{}
	if relData, err := readXLSXPart(zr, "xl/_rels/workbook.xml.rels"); err == nil {
		var rels struct {
			Relationships []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
			} `xml:"Relationship"`
		}
		if err := xml.Unmarshal(relData, &rels); err != nil {
			return nil, styles, nil, fmt.Errorf("xlsx rels: %w", err)
		}
		for _, rel := range rels.Relationships {
			targets[rel.ID] = rel.Target
		}
	}
	sheets := []xlsxSheet{}
	for _, s := range wb.Sheets {
		target := targets[s.RID]
		if target == "" {
			return nil, styles, nil, fmt.Errorf("xlsx sheet %q: relationship missing", s.Name)
		}
		part := path.Clean("xl/" + strings.TrimPrefix(target, "/"))
		if strings.HasPrefix(target, "/") {
			part = path.Clean(strings.TrimPrefix(target, "/"))
		}
		data, err := readXLSXPart(zr, part)
		if err != nil {
			return nil, styles, nil, err
		}
		sheet, err := readXLSXSheet(data, s.Name)
		if err != nil {
			return nil, styles, nil, err
		}
		sheets = append(sheets, sheet)
	}
	return sheets, styles, shared, nil
}

func readXLSXSharedStrings(data []byte) ([]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	out := []string{}
	var cur strings.Builder
	inSI, inT := false, false
	phonetic := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xlsx shared strings: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				inT = true
			case "rPh":
				phonetic++
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "si":
				out = append(out, cur.String())
				inSI = false
			case "t":
				inT = false
			case "rPh":
				phonetic--
			}
		case xml.CharData:
			if inSI && inT && phonetic == 0 {
				cur.Write(t)
			}
		}
	}
	return out, nil
}

func readXLSXSheet(data []byte, name string) (xlsxSheet, error) {
	var doc struct {
		Rows []struct {
			R     int `xml:"r,attr"`
			Cells []struct {
				R string  `xml:"r,attr"`
				T string  `xml:"t,attr"`
				S int     `xml:"s,attr"`
				F *string `xml:"f"`
				V string  `xml:"v"`
				IS        *struct {
					T string `xml:"t"`
					R []struct {
						T string `xml:"t"`
					} `xml:"r"`
				} `xml:"is"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return xlsxSheet{}, fmt.Errorf("xlsx sheet %q: %w", name, err)
	}
	sheet := xlsxSheet{name: name, rows: map[int][]xlsxCell{}, maxRow: -1}
	prevRow := 0
	for _, row := range doc.Rows {
		idx := row.R - 1
		if row.R == 0 {
			idx = prevRow
		}
		prevRow = idx + 1
		if idx > sheet.maxRow {
			sheet.maxRow = idx
		}
		cells := []xlsxCell{}
		prevCol := -1
		for _, cell := range row.Cells {
			col := prevCol + 1
			if cell.R != "" {
				col = xlsxColumnIndex(cell.R)
			}
			prevCol = col
			kind := cell.T
			if kind == "" {
				kind = "n"
			}
			text := cell.V
			if cell.F != nil {
				kind = "f"
			} else if kind == "inlineStr" && cell.IS != nil {
				text = cell.IS.T
				for _, run := range cell.IS.R {
					text += run.T
				}
			}
			cells = append(cells, xlsxCell{col: col, kind: kind, text: text, style: cell.S})
		}
		sheet.rows[idx] = cells
	}
	return sheet, nil
}

// xlsxColumnIndex converts an A1-style reference to its 0-based column.
func xlsxColumnIndex(ref string) int {
	col := 0
	for _, ch := range ref {
		if ch >= 'A' && ch <= 'Z' {
			col = col*26 + int(ch-'A') + 1
		} else if ch >= 'a' && ch <= 'z' {
			col = col*26 + int(ch-'a') + 1
		} else {
			break
		}
	}
	return col - 1
}

func parseXLSXRecords(sheets []xlsxSheet, styles xlsxStyleInfo, shared []string) ([]rawRecord, error) {
	sheetIndex, headerIndex := -1, -1
	for si, sheet := range sheets {
		for idx := 0; idx <= sheet.maxRow; idx++ {
			nonempty := false
			for _, cell := range sheet.rows[idx] {
				if cell.kind == "f" || cell.kind == "e" {
					nonempty = true
					break
				}
				if strings.TrimSpace(formatXLSXCell(cell, styles, shared)) != "" {
					nonempty = true
					break
				}
			}
			if nonempty {
				sheetIndex, headerIndex = si, idx
				break
			}
		}
		if sheetIndex >= 0 {
			break
		}
	}
	if err := errcode.Require(sheetIndex >= 0, 422, "INVALID_FILE", "缺少非空工作表"); err != nil {
		return nil, err
	}
	sheet := sheets[sheetIndex]
	if err := errcode.Require(sheet.maxRow-headerIndex <= 10000, 422, "INVALID_FILE", "最多10000行"); err != nil {
		return nil, err
	}
	headerCells := sheet.rows[headerIndex]
	lastCellNum := 0
	for _, cell := range headerCells {
		if cell.col+1 > lastCellNum {
			lastCellNum = cell.col + 1
		}
	}
	headers := make([]string, lastCellNum)
	for i := 0; i < lastCellNum; i++ {
		cell := sheet.cellAt(headerIndex, i)
		if cell != nil && cell.kind == "f" {
			return nil, errcode.New(422, "INVALID_FILE", "不接受公式")
		}
		if cell != nil {
			headers[i] = formatXLSXCell(*cell, styles, shared)
		}
	}
	if err := validateHeaders(headers); err != nil {
		return nil, err
	}
	rows := []rawRecord{}
	for idx := headerIndex + 1; idx <= sheet.maxRow; idx++ {
		if _, present := sheet.rows[idx]; !present {
			continue
		}
		record := rawRecord{keys: headers, vals: make(map[string]string, len(headers)),
			sheetRef: "sheet:" + strconv.Itoa(sheetIndex) + ":" + sheet.name}
		for j, h := range headers {
			cell := sheet.cellAt(idx, j)
			switch {
			case cell == nil:
				record.vals[h] = ""
			case cell.kind == "f":
				record.formula = true
				record.vals[h] = ""
			default:
				record.vals[h] = formatXLSXCell(*cell, styles, shared)
			}
		}
		rows = append(rows, record)
	}
	return rows, nil
}

// formatXLSXCell approximates POI DataFormatter(Locale.ROOT): strings as-is,
// booleans TRUE/FALSE, numbers in Excel General format, date-styled numbers
// rendered as plain dates/times.
func formatXLSXCell(cell xlsxCell, styles xlsxStyleInfo, shared []string) string {
	switch cell.kind {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(cell.text))
		if err != nil || idx < 0 || idx >= len(shared) {
			return ""
		}
		return shared[idx]
	case "inlineStr", "str", "e":
		return cell.text
	case "b":
		if cell.text == "1" {
			return "TRUE"
		}
		return "FALSE"
	case "f":
		return ""
	default:
		if strings.TrimSpace(cell.text) == "" {
			return ""
		}
		f, err := strconv.ParseFloat(cell.text, 64)
		if err != nil {
			return cell.text
		}
		if code, isDate := dateStyleFormat(cell.style, styles); isDate {
			return renderExcelDate(f, code)
		}
		return excelGeneral(f)
	}
}

// dateStyleFormat reports whether the cell style uses a date/time format.
func dateStyleFormat(style int, styles xlsxStyleInfo) (string, bool) {
	id := 0
	if style >= 0 && style < len(styles.cellXfs) {
		id = styles.cellXfs[style]
	}
	switch {
	case id >= 14 && id <= 22, id >= 27 && id <= 36, id >= 45 && id <= 47, id >= 50 && id <= 58:
		return builtinDateFormat(id), true
	}
	code, ok := styles.numFmts[id]
	if !ok {
		return "", false
	}
	return code, isDateFormatCode(code)
}

func builtinDateFormat(id int) string {
	switch id {
	case 18, 19, 20, 21, 45, 46, 47:
		return "h:mm:ss"
	case 22:
		return "m/d/yy h:mm"
	default:
		return "m/d/yy"
	}
}

var dateTokenRe = regexp.MustCompile(`[ymdhs]`)

// isDateFormatCode strips quoted literals and elapsed-time brackets, then
// looks for date/time tokens (POI DateUtil.isADateFormat approximation).
func isDateFormatCode(code string) bool {
	var b strings.Builder
	inQuote := false
	bracket := 0
	for _, ch := range code {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case inQuote:
		case ch == '[':
			bracket++
		case ch == ']':
			if bracket > 0 {
				bracket--
			}
		case bracket > 0:
		default:
			b.WriteRune(ch)
		}
	}
	cleaned := strings.ToLower(b.String())
	if strings.Contains(cleaned, "general") {
		return false
	}
	return dateTokenRe.MatchString(cleaned)
}

// renderExcelDate converts an Excel 1900-system serial (with the Lotus leap
// bug) and renders a plain date or date-time string.
func renderExcelDate(serial float64, code string) string {
	days := int64(math.Floor(serial))
	frac := serial - float64(days)
	base := time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
	if days >= 61 {
		base = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	}
	ms := int64(math.Round(frac * 86400000))
	t := base.AddDate(0, 0, int(days)).Add(time.Duration(ms) * time.Millisecond)
	lower := strings.ToLower(code)
	hasTime := strings.ContainsAny(lower, "hs") || strings.Contains(lower, "[m]")
	switch {
	case strings.Contains(lower, "h") && !strings.ContainsAny(lower, "yd"):
		return t.Format("15:04:05")
	case hasTime:
		return t.Format("2006-01-02 15:04:05")
	default:
		return t.Format("2006-01-02")
	}
}

// excelGeneral reproduces Excel's General number format (POI
// ExcelGeneralNumberFormat): integers below 1e11, scientific notation at the
// extremes, otherwise up to 11 significant digits.
func excelGeneral(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	abs := math.Abs(f)
	if abs < 1e11 && f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	if abs >= 1e11 || (abs < 1e-10 && f != 0) {
		s := strconv.FormatFloat(f, 'e', 10, 64)
		mant, exp, _ := strings.Cut(s, "e")
		if strings.Contains(mant, ".") {
			mant = strings.TrimRight(mant, "0")
			mant = strings.TrimSuffix(mant, ".")
		}
		e, _ := strconv.Atoi(exp)
		return mant + "E" + strconv.Itoa(e)
	}
	wholeDigits := 1
	if abs >= 1 {
		wholeDigits = int(math.Floor(math.Log10(abs))) + 1
	}
	frac := 11 - wholeDigits
	if frac < 0 {
		frac = 0
	}
	if frac > 10 {
		frac = 10
	}
	s := strconv.FormatFloat(f, 'f', frac, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return s
}

// formulaPrefixRe matches cells that spreadsheet apps would execute
// (safeCell: leading Java \s whitespace then = + - @).
var formulaPrefixRe = regexp.MustCompile(`(?s)^[ \t\n\x0b\f\r]*[=+\-@].*`)

// safeCell neutralizes formula injection in exported cells.
func safeCell(value any) string {
	var x string
	switch t := value.(type) {
	case nil:
		x = ""
	case time.Time:
		x = javaTimeCell(t)
	default:
		x = store.StrOrEmpty(value)
	}
	if formulaPrefixRe.MatchString(x) {
		return "'" + x
	}
	return x
}

// javaTimeCell renders OffsetDateTime.toString() (shortest ISO local time,
// fractional seconds in 3/6/9-digit groups, Z for UTC) as used by safeCell.
func javaTimeCell(t time.Time) string {
	t = t.UTC()
	s := t.Format("2006-01-02T15:04")
	if t.Second() != 0 || t.Nanosecond() != 0 {
		s += fmt.Sprintf(":%02d", t.Second())
		if t.Nanosecond() != 0 {
			frac := fmt.Sprintf("%09d", t.Nanosecond())
			n := len(frac)
			for n > 0 && frac[n-1] == '0' {
				n--
			}
			if n%3 != 0 {
				n += 3 - n%3
			}
			s += "." + frac[:n]
		}
	}
	return s + "Z"
}

// csvQuote applies CSVFormat.DEFAULT minimal quoting.
func csvQuote(s string) string {
	if strings.ContainsAny(s, ",\"\r\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// renderCSVExport writes BOM + CRLF records like CSVPrinter(CSVFormat.DEFAULT).
func renderCSVExport(columns []string, rows []map[string]any) []byte {
	var b strings.Builder
	b.WriteString(string([]byte{0xEF, 0xBB, 0xBF}))
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = csvQuote(col)
	}
	b.WriteString(strings.Join(quoted, ","))
	b.WriteString("\r\n")
	for _, row := range rows {
		for i, col := range columns {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(csvQuote(safeCell(row[col])))
		}
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

// renderXLSXExport writes a minimal string-only workbook (sheet "UniGate"),
// mirroring the POI XSSFWorkbook output shape.
func renderXLSXExport(columns []string, rows [][]string) ([]byte, error) {
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	writeRow := func(idx int, cells []string) {
		sheet.WriteString(`<row r="` + strconv.Itoa(idx) + `">`)
		for i, cell := range cells {
			ref := excelColumn(i+1) + strconv.Itoa(idx)
			sheet.WriteString(`<c r="` + ref + `" t="inlineStr"><is><t xml:space="preserve">` + xmlEscape(cell) + `</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	writeRow(1, columns)
	for i, row := range rows {
		writeRow(i+2, row)
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := []struct{ name, content string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			`</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="UniGate" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`</Relationships>`},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>` +
			`<fills count="1"><fill><patternFill patternType="none"/></fill></fills>` +
			`<borders count="1"><border/></borders>` +
			`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
			`<cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>` +
			`</styleSheet>`},
		{"xl/worksheets/sheet1.xml", sheet.String()},
	}
	for _, part := range parts {
		w, err := zw.Create(part.name)
		if err != nil {
			return nil, fmt.Errorf("xlsx write: %w", err)
		}
		if _, err := io.WriteString(w, part.content); err != nil {
			return nil, fmt.Errorf("xlsx write: %w", err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("xlsx close: %w", err)
	}
	return buf.Bytes(), nil
}
