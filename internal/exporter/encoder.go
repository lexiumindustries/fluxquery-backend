package exporter

import "io"

// RowEncoder defines a common interface for different export formats (CSV, JSON, Excel).
// It allows the exporter to be agnostic of the underlying output format.
type RowEncoder interface {
	// WriteHeader writes the initial column headers to the output.
	// This should be called exactly once before any rows are written.
	WriteHeader(columns []string) error

	// WriteRow writes a single row of data.
	// The values slice length must match the headers length.
	WriteRow(values []interface{}) error

	// Flush ensures all buffered data is written to the underlying writer.
	// This is critical for buffered writers like CSV or JSON streams.
	Flush() error

	// Error returns the first error that occurred during encoding, if any.
	// This allows for cleaner loops where error checking can happen at the end.
	Error() error

	// Close flushes the encoder and releases any resources.
	// For Excel, this might write the central directory/zip footer.
	io.Closer
}
