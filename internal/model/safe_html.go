package model

import "github.com/CloudyKit/jet/v6"

// SafeHTML is markup a layout prints as is. Layouts escape every other value,
// so only HTML the server built itself (rendered notes, sections, helpers)
// may carry this type.
type SafeHTML string

func (s SafeHTML) Render(r *jet.Runtime) {
	_, _ = r.Writer.Write([]byte(s))
}

func (s SafeHTML) String() string {
	return string(s)
}
