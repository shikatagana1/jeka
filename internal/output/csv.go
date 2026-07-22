package output

import (
	"encoding/csv"
	"io"
	"time"

	"github.com/shikatagana1/jeka/internal/model"
)

type csvWriter struct {
	w io.Writer
}

func newCSVWriter(w io.Writer) *csvWriter { return &csvWriter{w: w} }

func (c *csvWriter) Write(results []model.Result) error {
	cw := csv.NewWriter(c.w)
	if err := cw.Write([]string{
		"ip", "domain", "first_seen", "last_seen", "source", "kind", "is_last",
	}); err != nil {
		return err
	}

	for _, res := range results {
		ip := res.IP.String()
		for _, d := range res.Current {
			if err := cw.Write([]string{
				ip, d, "", "", "livePTR", "current", boolStr(d == res.Last),
			}); err != nil {
				return err
			}
		}
		for _, r := range res.History {
			if err := cw.Write([]string{
				ip, r.Domain, csvTime(r.FirstSeen), csvTime(r.LastSeen),
				r.Source, "history", boolStr(r.Domain == res.Last),
			}); err != nil {
				return err
			}
		}
	}

	cw.Flush()
	return cw.Error()
}

func csvTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
