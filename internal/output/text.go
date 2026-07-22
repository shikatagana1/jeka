package output

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/shikatagana1/jeka/internal/model"
)

type textWriter struct {
	w io.Writer
}

func newTextWriter(w io.Writer) *textWriter { return &textWriter{w: w} }

func (t *textWriter) Write(results []model.Result) error {
	for i, res := range results {
		if i > 0 {
			if _, err := fmt.Fprintln(t.w); err != nil {
				return err
			}
		}
		if err := t.writeOne(res); err != nil {
			return err
		}
	}
	return nil
}

func (t *textWriter) writeOne(res model.Result) error {
	if _, err := fmt.Fprintf(t.w, "== %s ==\n", res.IP); err != nil {
		return err
	}

	if len(res.Current) == 0 {
		if _, err := fmt.Fprintln(t.w, "Current: (none)"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(t.w, "Current:"); err != nil {
			return err
		}
		for _, d := range res.Current {
			marker := "  - "
			if d == res.Last {
				marker = "  * "
			}
			if _, err := fmt.Fprintf(t.w, "%s%s\n", marker, d); err != nil {
				return err
			}
		}
	}

	if len(res.History) > 0 {
		if _, err := fmt.Fprintln(t.w, "History:"); err != nil {
			return err
		}
		tw := tabwriter.NewWriter(t.w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "  DOMAIN\tFIRST SEEN\tLAST SEEN\tSOURCE")
		for _, r := range res.History {
			name := "  " + r.Domain
			if r.Domain == res.Last {
				name = "* " + r.Domain
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				name, fmtTime(r.FirstSeen), fmtTime(r.LastSeen), r.Source)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}

	if res.Last != "" {
		if _, err := fmt.Fprintf(t.w, "Last: %s\n", res.Last); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(t.w, "Last: (none)"); err != nil {
			return err
		}
	}
	for _, e := range res.Errors {
		if _, err := fmt.Fprintf(t.w, "! %s\n", e); err != nil {
			return err
		}
	}
	return nil
}

func fmtTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format("2006-01-02")
}
