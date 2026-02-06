package exporter

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ExcelEncoder implements RowEncoder for Excel (.xlsx) files.
// It uses excelize.StreamWriter for efficient writing of large files.
type ExcelEncoder struct {
	f            *excelize.File
	sw           *excelize.StreamWriter
	w            io.Writer
	sheetName    string
	rowIdx       int
	err          error
	headerLength int
}

// NewExcelEncoder creates a new Excel encoder.
// It initializes a new workbook and specific stream writer for high performance.
func NewExcelEncoder(w io.Writer) *ExcelEncoder {
	f := excelize.NewFile()
	sheetName := "Sheet1"
	sw, err := f.NewStreamWriter(sheetName)
	if err != nil {
		return &ExcelEncoder{err: err}
	}

	return &ExcelEncoder{
		f:         f,
		sw:        sw,
		w:         w,
		sheetName: sheetName,
		rowIdx:    1,
	}
}

func (e *ExcelEncoder) WriteHeader(columns []string) error {
	if e.err != nil {
		return e.err
	}

	e.headerLength = len(columns)
	row := make([]interface{}, len(columns))
	for i, col := range columns {
		row[i] = col
	}

	cell, err := excelize.CoordinatesToCellName(1, e.rowIdx)
	if err != nil {
		e.err = err
		return err
	}

	if err := e.sw.SetRow(cell, row); err != nil {
		e.err = err
		return err
	}

	e.rowIdx++
	return nil
}

func (e *ExcelEncoder) WriteRow(values []interface{}) error {
	if e.err != nil {
		return e.err
	}

	// Excelize StreamWriter requires interface{} slice
	row := make([]interface{}, len(values))
	for i, v := range values {
		var s string
		switch val := v.(type) {
		case []byte:
			s = string(val)
		case string:
			s = val
		case nil:
			s = "NULL"
		default:
			// For other types, we can use fmt.Sprint or just pass through if excelize handles them
			// But for safety and consistency with Formula Injection mitigation, let's treat all as strings for now
			// or handle them specifically. Excelize handles numbers natively.
			row[i] = v
			continue
		}

		// Formula Injection Mitigation
		if len(s) > 0 {
			first := s[0]
			if first == '=' || first == '+' || first == '-' || first == '@' {
				s = "'" + s
			}
		}
		row[i] = s
	}

	cell, err := excelize.CoordinatesToCellName(1, e.rowIdx)
	if err != nil {
		e.err = err
		return err
	}

	if err := e.sw.SetRow(cell, row); err != nil {
		e.err = err
		return err
	}

	e.rowIdx++

	// Excel hard limit: 1,048,576 rows
	if e.rowIdx > 1048576 {
		e.err = fmt.Errorf("excel row limit exceeded (1,048,576 rows)")
		return e.err
	}

	return nil
}

func (e *ExcelEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}

	if err := e.sw.Flush(); err != nil {
		e.err = err
		return err
	}

	return e.f.Write(e.w)
}

func (e *ExcelEncoder) Error() error {
	return e.err
}

func (e *ExcelEncoder) Close() error {
	if e.f != nil {
		// We don't usually need to close the file if we are writing to a buffer/stream
		// but let's be safe.
		_ = e.f.Close()
	}
	return nil
}
