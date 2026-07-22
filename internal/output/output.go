package output

import (
	"fmt"
	"io"

	"github.com/shikatagana1/jeka/internal/model"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
)

type Writer interface {
	Write(results []model.Result) error
}

func NewWriter(format Format, w io.Writer) (Writer, error) {
	switch format {
	case FormatText:
		return newTextWriter(w), nil
	case FormatJSON:
		return newJSONWriter(w), nil
	case FormatCSV:
		return newCSVWriter(w), nil
	default:
		return nil, fmt.Errorf("unknown output format %q", format)
	}
}
