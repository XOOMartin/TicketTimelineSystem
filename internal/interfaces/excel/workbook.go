// Package excel converts check-ins and reports to and from .xlsx files. Like
// JSON in the rest package, it is a presentation format, so it lives with the
// interfaces and the domain never sees it.
package excel

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// sheet writes one table: an optional title line, a header row, then rows.
type sheet struct {
	f       *excelize.File
	name    string
	row     int
	header  int // style ids
	title   int
	hours   int
	columns int
}

// newTab adds a sheet, or renames the default first sheet for the first
// one, so a workbook doesn't start with an empty "Sheet1". Every workbook
// here adds its sheets with it.
func newTab(f *excelize.File, name string) error {
	if f.SheetCount == 1 && f.GetSheetName(0) == "Sheet1" {
		return f.SetSheetName("Sheet1", name)
	}
	_, err := f.NewSheet(name)
	return err
}

// ref is a cell's name, e.g. ref(2, 5) is "B5".
func ref(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// colName is a column's letters, e.g. colName(28) is "AB".
func colName(col int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return name
}

// addSheet creates a sheet for one table, with its header, title and hours
// styles.
func addSheet(f *excelize.File, name string) (*sheet, error) {
	if err := newTab(f, name); err != nil {
		return nil, err
	}

	header, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, err
	}
	title, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	if err != nil {
		return nil, err
	}
	hours, err := f.NewStyle(&excelize.Style{NumFmt: 2}) // 0.00
	if err != nil {
		return nil, err
	}
	return &sheet{f: f, name: name, row: 1, header: header, title: title, hours: hours}, nil
}

func (s *sheet) cell(col, row int) string { return ref(col, row) }

// writeTitle adds a bold line and a blank line above the table.
func (s *sheet) writeTitle(text string) error {
	if err := s.f.SetCellValue(s.name, s.cell(1, s.row), text); err != nil {
		return err
	}
	if err := s.f.SetCellStyle(s.name, s.cell(1, s.row), s.cell(1, s.row), s.title); err != nil {
		return err
	}
	s.row += 2
	return nil
}

// writeHeader writes the column names and freezes the panes below them.
// widths are in characters, one per column.
func (s *sheet) writeHeader(names []string, widths []float64) error {
	s.columns = len(names)
	if err := s.f.SetSheetRow(s.name, s.cell(1, s.row), &names); err != nil {
		return err
	}
	if err := s.f.SetCellStyle(s.name, s.cell(1, s.row), s.cell(len(names), s.row), s.header); err != nil {
		return err
	}
	for i, w := range widths {
		col := colName(i + 1)
		if err := s.f.SetColWidth(s.name, col, col, w); err != nil {
			return err
		}
	}
	err := s.f.SetPanes(s.name, &excelize.Panes{
		Freeze:      true,
		YSplit:      s.row,
		TopLeftCell: s.cell(1, s.row+1),
		ActivePane:  "bottomLeft",
	})
	if err != nil {
		return err
	}
	s.row++
	return nil
}

func (s *sheet) writeRow(values []any) error {
	err := s.f.SetSheetRow(s.name, s.cell(1, s.row), &values)
	s.row++
	return err
}

// formatHours applies the 0.00 format to the given columns of the data rows.
func (s *sheet) formatHours(firstDataRow int, cols ...int) error {
	if s.row <= firstDataRow {
		return nil
	}
	for _, c := range cols {
		if err := s.f.SetCellStyle(s.name, s.cell(c, firstDataRow), s.cell(c, s.row-1), s.hours); err != nil {
			return err
		}
	}
	return nil
}

// addFilter turns on Excel's filter buttons for the table.
func (s *sheet) addFilter(headerRow int) error {
	last := s.row - 1
	if last < headerRow {
		last = headerRow
	}
	return s.f.AutoFilter(s.name, fmt.Sprintf("%s:%s", s.cell(1, headerRow), s.cell(s.columns, last)), nil)
}

func save(f *excelize.File, w io.Writer) error {
	f.SetActiveSheet(0)
	if err := f.Write(w); err != nil {
		return err
	}
	return f.Close()
}
