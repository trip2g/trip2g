package model

import "github.com/CloudyKit/jet/v6"

type Layout struct {
	View   *jet.Template
	Assets []string
}

type Layouts struct {
	Map map[string]Layout
}
