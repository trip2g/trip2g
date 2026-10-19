package main

import "time"

// slowNoteQuery is how long a named note query may take before it is logged.
const slowNoteQuery = time.Second

func (a *app) ObserveNoteQuery(query, result string, elapsed time.Duration, paramsBytes int) {
	a.noteQueryMetrics.ObserveNoteQuery(query, result, elapsed.Seconds())

	if elapsed >= slowNoteQuery {
		a.log.Warn("slow note query",
			"query", query,
			"result", result,
			"seconds", elapsed.Seconds(),
			"params_bytes", paramsBytes,
		)
	}
}
