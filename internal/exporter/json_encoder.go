package exporter

import (
	"encoding/json"
	"io"
)

// JSONEncoder implements RowEncoder for JSON Lines format.
// Each row is exported as a JSON object on a new line.
type JSONEncoder struct {
	w       io.Writer
	columns []string
	err     error
}

// NewJSONEncoder creates a new JSON Lines encoder.
func NewJSONEncoder(w io.Writer) *JSONEncoder {
	return &JSONEncoder{w: w}
}

// WriteHeader captures the column names to be used as JSON keys.
// Unlike CSV, JSON doesn't write a header row, but needs the names for object properties.
func (e *JSONEncoder) WriteHeader(columns []string) error {
	e.columns = columns
	return nil
}

func (e *JSONEncoder) WriteRow(values []interface{}) error {
	if e.err != nil {
		return e.err
	}

	rowMap := make(map[string]interface{}, len(values))
	for i, v := range values {
		// Use column names as keys
		colName := "column_" + string(rune(i))
		if i < len(e.columns) {
			colName = e.columns[i]
		}

		// Some types might need special handling for JSON (like []byte)
		if b, ok := v.([]byte); ok {
			rowMap[colName] = string(b)
		} else {
			rowMap[colName] = v
		}
	}

	data, err := json.Marshal(rowMap)
	if err != nil {
		e.err = err
		return err
	}

	_, err = e.w.Write(data)
	if err != nil {
		e.err = err
		return err
	}
	_, err = e.w.Write([]byte("\n"))
	if err != nil {
		e.err = err
		return err
	}

	return nil
}

func (e *JSONEncoder) Flush() error {
	return nil
}

func (e *JSONEncoder) Error() error {
	return e.err
}

func (e *JSONEncoder) Close() error {
	return e.Flush()
}
