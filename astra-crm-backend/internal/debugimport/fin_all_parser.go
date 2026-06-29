package debugimport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const finAllSheetName = "Fin_ALL"

var (
	ErrInvalidWorkbook = errors.New("invalid workbook")
	ErrFinAllNotFound  = errors.New("Fin_ALL sheet not found")
)

type finAllRow struct {
	SourceRow int64
	Operator  string
	Phone     string
	Card      string
	Holder    string
	Circles   []finAllCircle
}

type finAllCircle struct {
	Number                int
	Date                  time.Time
	InboundTurnoverMinor  int64
	ClosingBalanceMinor   int64
	OutboundTurnoverMinor int64
	Blocked               bool
	Status                string
}

type parseWarning struct {
	Row     int64  `json:"row"`
	Circle  int    `json:"circle,omitempty"`
	Message string `json:"message"`
}

func parseFinAllWorkbook(data []byte, loc *time.Location) ([]finAllRow, []parseWarning, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidWorkbook, err)
	}

	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}

	sharedStrings, err := readSharedStrings(files)
	if err != nil {
		return nil, nil, err
	}
	sheets, err := readWorkbookSheets(files)
	if err != nil {
		return nil, nil, err
	}
	rels, err := readWorkbookRelationships(files)
	if err != nil {
		return nil, nil, err
	}

	var sheetPath string
	for _, sheet := range sheets {
		if sheet.Name != finAllSheetName {
			continue
		}
		target, ok := rels[sheet.RelID]
		if !ok {
			return nil, nil, fmt.Errorf("%w: missing relationship for %s", ErrInvalidWorkbook, finAllSheetName)
		}
		sheetPath = normalizeXLTarget(target)
		break
	}
	if sheetPath == "" {
		return nil, nil, ErrFinAllNotFound
	}

	matrix, err := readWorksheet(files, sheetPath, sharedStrings)
	if err != nil {
		return nil, nil, err
	}

	return parseFinAllRows(matrix, loc)
}

type workbookSheet struct {
	Name  string
	RelID string
}

func readWorkbookSheets(files map[string]*zip.File) ([]workbookSheet, error) {
	file := files["xl/workbook.xml"]
	if file == nil {
		return nil, fmt.Errorf("%w: xl/workbook.xml not found", ErrInvalidWorkbook)
	}
	body, err := readZipFile(file)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	var sheets []workbookSheet
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: workbook xml: %v", ErrInvalidWorkbook, err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sheet" {
			continue
		}
		var sheet workbookSheet
		for _, attr := range start.Attr {
			switch {
			case attr.Name.Local == "name":
				sheet.Name = attr.Value
			case attr.Name.Local == "id":
				sheet.RelID = attr.Value
			}
		}
		if sheet.Name != "" && sheet.RelID != "" {
			sheets = append(sheets, sheet)
		}
	}
	return sheets, nil
}

func readWorkbookRelationships(files map[string]*zip.File) (map[string]string, error) {
	file := files["xl/_rels/workbook.xml.rels"]
	if file == nil {
		return nil, fmt.Errorf("%w: xl/_rels/workbook.xml.rels not found", ErrInvalidWorkbook)
	}
	body, err := readZipFile(file)
	if err != nil {
		return nil, err
	}

	type relationship struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	type relationships struct {
		Items []relationship `xml:"Relationship"`
	}
	var parsed relationships
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: workbook relationships: %v", ErrInvalidWorkbook, err)
	}
	result := map[string]string{}
	for _, rel := range parsed.Items {
		result[rel.ID] = rel.Target
	}
	return result, nil
}

func readSharedStrings(files map[string]*zip.File) ([]string, error) {
	file := files["xl/sharedStrings.xml"]
	if file == nil {
		return nil, nil
	}
	body, err := readZipFile(file)
	if err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(bytes.NewReader(body))
	var values []string
	var current strings.Builder
	inSI := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: shared strings: %v", ErrInvalidWorkbook, err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "si" {
				inSI = true
				current.Reset()
			}
		case xml.EndElement:
			if value.Name.Local == "si" && inSI {
				values = append(values, current.String())
				inSI = false
			}
		case xml.CharData:
			if inSI {
				current.Write([]byte(value))
			}
		}
	}
	return values, nil
}

type worksheetCell struct {
	Ref  string `xml:"r,attr"`
	Type string `xml:"t,attr"`
	V    string `xml:"v"`
	IS   struct {
		T string `xml:"t"`
	} `xml:"is"`
}

type worksheetRow struct {
	Index int64           `xml:"r,attr"`
	Cells []worksheetCell `xml:"c"`
}

type worksheetData struct {
	Rows []worksheetRow `xml:"sheetData>row"`
}

func readWorksheet(files map[string]*zip.File, sheetPath string, sharedStrings []string) (map[int64]map[int]string, error) {
	file := files[sheetPath]
	if file == nil {
		return nil, fmt.Errorf("%w: %s not found", ErrInvalidWorkbook, sheetPath)
	}
	body, err := readZipFile(file)
	if err != nil {
		return nil, err
	}
	var parsed worksheetData
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: worksheet xml: %v", ErrInvalidWorkbook, err)
	}

	matrix := make(map[int64]map[int]string, len(parsed.Rows))
	for _, row := range parsed.Rows {
		values := map[int]string{}
		for _, cell := range row.Cells {
			col := columnIndex(cell.Ref)
			if col <= 0 {
				continue
			}
			value := decodeCellValue(cell, sharedStrings)
			if value != "" {
				values[col] = value
			}
		}
		matrix[row.Index] = values
	}
	return matrix, nil
}

func decodeCellValue(cell worksheetCell, sharedStrings []string) string {
	switch cell.Type {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(cell.V))
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return ""
		}
		return strings.TrimSpace(sharedStrings[idx])
	case "inlineStr":
		return strings.TrimSpace(cell.IS.T)
	default:
		return strings.TrimSpace(cell.V)
	}
}

func parseFinAllRows(matrix map[int64]map[int]string, loc *time.Location) ([]finAllRow, []parseWarning, error) {
	if loc == nil {
		loc = time.UTC
	}

	rowNumbers := make([]int64, 0, len(matrix))
	for row := range matrix {
		rowNumbers = append(rowNumbers, row)
	}
	sort.Slice(rowNumbers, func(i, j int) bool { return rowNumbers[i] < rowNumbers[j] })

	var rows []finAllRow
	var warnings []parseWarning
	for _, rowNumber := range rowNumbers {
		if rowNumber == 1 {
			continue
		}
		values := matrix[rowNumber]
		operator := cleanText(values[1])
		phone := normalizeDigits(values[2])
		card := normalizeDigits(values[3])
		holder := cleanText(values[4])
		if operator == "" && phone == "" && card == "" {
			continue
		}
		if operator == "" || phone == "" || card == "" || holder == "" {
			warnings = append(warnings, parseWarning{Row: rowNumber, Message: "строка пропущена: нужны Сотруд, Телефон, НК и ФИО"})
			continue
		}

		item := finAllRow{
			SourceRow: rowNumber,
			Operator:  operator,
			Phone:     phone,
			Card:      card,
			Holder:    holder,
		}
		var previousBalance *int64
		for circle := 1; circle <= 6; circle++ {
			baseCol := 8 + (circle-1)*4
			dateRaw := cleanText(values[baseCol])
			inboundRaw := cleanText(values[baseCol+1])
			balanceRaw := cleanText(values[baseCol+2])
			status := cleanText(values[baseCol+3])
			if dateRaw == "" && inboundRaw == "" && balanceRaw == "" && status == "" {
				continue
			}

			circleDate, err := parseExcelDate(dateRaw, loc)
			if err != nil {
				warnings = append(warnings, parseWarning{Row: rowNumber, Circle: circle, Message: "круг пропущен: дата не распознана"})
				continue
			}
			inboundMinor, err := parseMoneyMinor(inboundRaw)
			if err != nil {
				warnings = append(warnings, parseWarning{Row: rowNumber, Circle: circle, Message: "круг пропущен: оборот не распознан"})
				continue
			}
			balanceMinor, err := parseMoneyMinor(balanceRaw)
			if err != nil {
				warnings = append(warnings, parseWarning{Row: rowNumber, Circle: circle, Message: "круг пропущен: остаток не распознан"})
				continue
			}

			outboundMinor := inboundMinor - balanceMinor
			if previousBalance != nil {
				outboundMinor = *previousBalance + inboundMinor - balanceMinor
			}
			if outboundMinor < 0 {
				warnings = append(warnings, parseWarning{Row: rowNumber, Circle: circle, Message: "расчет выплат получился отрицательным, сохранено 0"})
				outboundMinor = 0
			}

			blocked := strings.EqualFold(status, "Блок")
			item.Circles = append(item.Circles, finAllCircle{
				Number:                circle,
				Date:                  circleDate,
				InboundTurnoverMinor:  inboundMinor,
				ClosingBalanceMinor:   balanceMinor,
				OutboundTurnoverMinor: outboundMinor,
				Blocked:               blocked,
				Status:                status,
			})
			nextBalance := balanceMinor
			previousBalance = &nextBalance
		}
		if len(item.Circles) == 0 {
			warnings = append(warnings, parseWarning{Row: rowNumber, Message: "строка пропущена: нет кругов"})
			continue
		}
		rows = append(rows, item)
	}
	return rows, warnings, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func normalizeXLTarget(target string) string {
	target = strings.TrimPrefix(target, "/")
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean("xl/" + target)
}

var cellColumnRegexp = regexp.MustCompile(`^[A-Z]+`)

func columnIndex(ref string) int {
	letters := cellColumnRegexp.FindString(strings.ToUpper(ref))
	if letters == "" {
		return 0
	}
	col := 0
	for _, r := range letters {
		col = col*26 + int(r-'A'+1)
	}
	return col
}

func cleanText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeDigits(value string) string {
	value = cleanText(value)
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, ".Ee") {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			value = strconv.FormatInt(int64(parsed), 10)
		}
	}
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func parseExcelDate(value string, loc *time.Location) (time.Time, error) {
	value = cleanText(value)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}
	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		days := int(serial)
		fraction := serial - float64(days)
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, loc)
		return base.AddDate(0, 0, days).Add(time.Duration(fraction * float64(24*time.Hour))), nil
	}
	for _, layout := range []string{"02.01.2006", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, loc); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date %q", value)
}

func parseMoneyMinor(value string) (int64, error) {
	value = strings.ReplaceAll(cleanText(value), " ", "")
	value = strings.ReplaceAll(value, ",", ".")
	if value == "" {
		return 0, nil
	}
	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, fmt.Errorf("invalid money %q", value)
	}
	rat.Mul(rat, big.NewRat(100, 1))
	if rat.Sign() < 0 {
		return 0, fmt.Errorf("negative money %q", value)
	}
	num := new(big.Int).Set(rat.Num())
	den := new(big.Int).Set(rat.Denom())
	half := new(big.Int).Div(new(big.Int).Set(den), big.NewInt(2))
	num.Add(num, half)
	num.Quo(num, den)
	if !num.IsInt64() {
		return 0, fmt.Errorf("money too large %q", value)
	}
	return num.Int64(), nil
}
