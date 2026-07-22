package output

import (
	"encoding/json"
	"io"

	"github.com/shikatagana1/jeka/internal/model"
)

type jsonWriter struct {
	w io.Writer
}

func newJSONWriter(w io.Writer) *jsonWriter { return &jsonWriter{w: w} }

func (j *jsonWriter) Write(results []model.Result) error {
	if results == nil {
		results = []model.Result{}
	}
	b, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	if _, err := j.w.Write(b); err != nil {
		return err
	}
	_, err = j.w.Write([]byte{'\n'})
	return err
}
