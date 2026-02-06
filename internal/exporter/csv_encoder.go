package exporter

import (
	"bufio"
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

// CSVEncoder wraps encoding/csv with type-aware, low-allocation logic.
// It uses a bufio.Writer to minimize IO syscalls, which is crucial for high-throughput exporting.
type CSVEncoder struct {
	w       *csv.Writer
	buf     *bufio.Writer
	columns []string
}

// NewCSVEncoder creates a new CSV encoder that writes to the provided io.Writer.
// It initializes a 64KB buffer to optimize write performance.
func NewCSVEncoder(w io.Writer) *CSVEncoder {
	buf := bufio.NewWriterSize(w, 64*1024) // 64KB buffer
	cw := csv.NewWriter(buf)
	return &CSVEncoder{
		w:   cw,
		buf: buf,
	}
}

// WriteHeader writes the CSV header row.
func (e *CSVEncoder) WriteHeader(columns []string) error {
	e.columns = columns
	return e.w.Write(columns)
}

// WriteRow writes a single row of values, defined as interface{} to handle SQL driver types.
// It converts types to string efficiently without fmt.Sprintf.
func (e *CSVEncoder) WriteRow(values []interface{}) error {
	record := make([]string, len(values)) // Re-using this buffer would be an optimization, but encoding/csv copies anyway.

	for i, v := range values {
		record[i] = toString(v)
	}

	return e.w.Write(record)
}

// Flush ensures all data is written to the underlying writer.
func (e *CSVEncoder) Flush() error {
	e.w.Flush()
	if err := e.w.Error(); err != nil {
		return err
	}
	return e.buf.Flush()
}

// Error returns any error stored in the CSV writer.
func (e *CSVEncoder) Error() error {
	return e.w.Error()
}

// Close flushes and satisfies io.Closer.
func (e *CSVEncoder) Close() error {
	return e.Flush()
}

func toString(val interface{}) string {
	var s string
	if val == nil {
		s = "NULL"
	} else {
		switch v := val.(type) {
		case []byte:
			s = string(v)
		case string:
			s = v
		case time.Time:
			s = v.Format("2006-01-02 15:04:05")
		case int64:
			s = strconv.FormatInt(v, 10)
		case int:
			s = strconv.Itoa(v)
		case float64:
			s = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			if v {
				s = "1"
			} else {
				s = "0"
			}
		default:
			s = ""
		}
	}

	// Formula Injection Mitigation (CSV Injection)
	// If the string starts with =, +, -, or @, prefix it with a single quote.
	if len(s) > 0 {
		first := s[0]
		if first == '=' || first == '+' || first == '-' || first == '@' {
			s = "'" + s
		}
	}
	return s
}
