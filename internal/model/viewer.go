package model

import "trip2g/internal/usertoken"

type Viewer struct {
	UserID    *int64
	UserToken *usertoken.Data
}

func (v *Viewer) ID() string {
	return "viewer"
}
