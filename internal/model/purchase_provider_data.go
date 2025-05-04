package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type PurchaseProviderData struct {
}

func (p PurchaseProviderData) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *PurchaseProviderData) Scan(src interface{}) error {
	var data []byte

	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}

	return json.Unmarshal(data, p)
}
