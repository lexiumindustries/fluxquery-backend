package exporter

import (
	"io"
	"strings"

	"github.com/go-pdf/fpdf"
)

// PDFEncoder implements RowEncoder for PDF generation.
// It creates a simple grid layout for exported data.
// WARNING: PDF generation is memory intensive and slower than CSV/JSON.
type PDFEncoder struct {
	pdf *fpdf.Fpdf
	w   io.Writer
	err error
}

// NewPDFEncoder creates a new PDF encoder.
func NewPDFEncoder(w io.Writer) *PDFEncoder {
	pdf := fpdf.New("L", "mm", "A4", "") // Landscape, mm, A4
	pdf.SetFont("Arial", "", 10)
	pdf.AddPage()
	return &PDFEncoder{
		pdf: pdf,
		w:   w,
	}
}

// WriteHeader writes the table headers.
func (e *PDFEncoder) WriteHeader(columns []string) error {
	if e.err != nil {
		return e.err
	}

	e.pdf.SetFont("Arial", "B", 10)
	// Simple assumption: distribute width equally
	// A4 Landscape width is ~297mm. Left/Right margins default to 10mm each.
	// Usable width ~277mm.
	pageWidth, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	usableWidth := pageWidth - left - right

	colWidth := usableWidth / float64(len(columns))

	for _, col := range columns {
		e.pdf.CellFormat(colWidth, 7, col, "1", 0, "C", false, 0, "")
	}
	e.pdf.Ln(-1)
	e.pdf.SetFont("Arial", "", 10) // Reset font
	return nil
}

// WriteRow writes a single row of data.
func (e *PDFEncoder) WriteRow(values []interface{}) error {
	if e.err != nil {
		return e.err
	}

	pageWidth, _ := e.pdf.GetPageSize()
	left, _, right, _ := e.pdf.GetMargins()
	usableWidth := pageWidth - left - right
	colWidth := usableWidth / float64(len(values))

	// Determine max height for the row (handling multiline text)
	// For simplicity in this v1, we just enforce single line or simple truncation/wrapping
	// Standard cell height 7mm
	rowHeight := 7.0

	for _, v := range values {
		str := toString(v)                 // Re-use our toString helper from csv_encoder (need to expose it or duplicate it)
		str = strings.TrimPrefix(str, "'") // Remove CSV injection protection quote for visual smoothness in PDF

		// Basic cleanup of unsupported chars if necessary
		// fpdf handles utf8 if we use the unicode translator, but standard core fonts are lat1.
		// For now, simple ASCII.

		e.pdf.CellFormat(colWidth, rowHeight, str, "1", 0, "L", false, 0, "")
	}
	e.pdf.Ln(-1)
	return nil
}

// Flush writes the PDF to the underlying writer.
func (e *PDFEncoder) Flush() error {
	if e.err != nil {
		return e.err
	}
	return e.pdf.Output(e.w)
}

// Error returns any stored error.
func (e *PDFEncoder) Error() error {
	return e.err
}

// Close flushes and satisfies io.Closer.
func (e *PDFEncoder) Close() error {
	return e.Flush()
}
