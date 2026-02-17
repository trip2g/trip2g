package model

type HotAuthToken struct {
	Email      string `json:"e"`
	AdminEnter bool   `json:"ae,omitempty"`
}
