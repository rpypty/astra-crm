package debugimport

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"
)

func TestParseFinAllWorkbookCalculatesCircles(t *testing.T) {
	data := testFinAllWorkbook(t)
	rows, warnings, err := parseFinAllWorkbook(data, time.UTC)
	if err != nil {
		t.Fatalf("parseFinAllWorkbook() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if rows[0].Operator != "Fin_Core_Op5" || rows[0].Phone != "79380406524" || rows[0].Card != "2204321200000000" {
		t.Fatalf("unexpected row identity: %+v", rows[0])
	}
	if len(rows[0].Circles) != 3 {
		t.Fatalf("circles len = %d, want 3", len(rows[0].Circles))
	}

	first := rows[0].Circles[0]
	if first.InboundTurnoverMinor != 100000 || first.ClosingBalanceMinor != 10000 || first.OutboundTurnoverMinor != 90000 {
		t.Fatalf("first circle = %+v", first)
	}

	second := rows[0].Circles[1]
	if second.OutboundTurnoverMinor != 45000 {
		t.Fatalf("second outbound = %d, want 45000", second.OutboundTurnoverMinor)
	}

	third := rows[0].Circles[2]
	if !third.Blocked || third.OutboundTurnoverMinor != 0 {
		t.Fatalf("third circle = %+v", third)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1", len(warnings))
	}
	if warnings[0].Row != 2 || warnings[0].Circle != 3 {
		t.Fatalf("warning = %+v", warnings[0])
	}
}

func testFinAllWorkbook(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	writeZipFile(t, writer, "[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
</Types>`)
	writeZipFile(t, writer, "xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="Fin_ALL" sheetId="1" r:id="rId1"/></sheets>
</workbook>`)
	writeZipFile(t, writer, "xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`)
	writeZipFile(t, writer, "xl/sharedStrings.xml", `<?xml version="1.0" encoding="UTF-8"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <si><t>Сотруд</t></si><si><t>Телефон</t></si><si><t>НК</t></si><si><t>ФИО</t></si>
  <si><t>Общийоборот</t></si><si><t>Остаток</t></si><si><t>СтатусБанка</t></si><si><t>Дата1круг</t></si>
  <si><t>Оборот</t></si><si><t>Дата2круг</t></si><si><t>Дата3круг</t></si><si><t>Fin_Core_Op5</t></si>
  <si><t>Иван И.</t></si><si><t>Все Ок</t></si><si><t>Блок</t></si>
</sst>`)
	writeZipFile(t, writer, "xl/worksheets/sheet1.xml", `<?xml version="1.0" encoding="UTF-8"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1">
      <c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>2</v></c><c r="D1" t="s"><v>3</v></c>
      <c r="E1" t="s"><v>4</v></c><c r="F1" t="s"><v>5</v></c><c r="G1" t="s"><v>6</v></c><c r="H1" t="s"><v>7</v></c>
      <c r="I1" t="s"><v>8</v></c><c r="J1" t="s"><v>5</v></c><c r="K1" t="s"><v>6</v></c><c r="L1" t="s"><v>9</v></c>
      <c r="P1" t="s"><v>10</v></c>
    </row>
    <row r="2">
      <c r="A2" t="s"><v>11</v></c><c r="B2"><v>79380406524</v></c><c r="C2"><v>2204321200000000</v></c><c r="D2" t="s"><v>12</v></c>
      <c r="H2"><v>46188</v></c><c r="I2"><v>1000</v></c><c r="J2"><v>100</v></c><c r="K2" t="s"><v>13</v></c>
      <c r="L2"><v>46189</v></c><c r="M2"><v>500</v></c><c r="N2"><v>150</v></c><c r="O2" t="s"><v>13</v></c>
      <c r="P2"><v>46190</v></c><c r="Q2"><v>0</v></c><c r="R2"><v>150.50</v></c><c r="S2" t="s"><v>14</v></c>
    </row>
  </sheetData>
</worksheet>`)
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func writeZipFile(t *testing.T, writer *zip.Writer, name string, body string) {
	t.Helper()
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := file.Write([]byte(body)); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
