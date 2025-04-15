package db

import (
	"database/sql"
	"time"
)

// ToNullable helpers for converting Go types to SQL nullable types.
func ToNullableFloat64(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{Valid: false}
	}

	return sql.NullFloat64{
		Float64: *value,
		Valid:   true,
	}
}

// ToNullableTime converts a pointer to an int64 to sql.NullTime.
func ToNullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{Valid: false}
	}

	return sql.NullTime{
		Time:  *value,
		Valid: true,
	}
}

// ToNullableString converts a pointer to a string to sql.NullString.
func ToNullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{Valid: false}
	}

	return sql.NullString{
		String: *value,
		Valid:  true,
	}
}

// ToFloat64Ptr converts a sql.NullFloat64 to a pointer to a float64.
func ToFloat64Ptr(v sql.NullFloat64) *float64 {
	if v.Valid {
		return &v.Float64
	}

	return nil
}

// ToTimePtr converts a sql.NullTime to a pointer to a time.Time.
func ToTimePtr(v sql.NullTime) *time.Time {
	if v.Valid {
		return &v.Time
	}

	return nil
}

// ToStringPtr converts a sql.NullString to a pointer to a string.
func ToStringPtr(v sql.NullString) *string {
	if v.Valid {
		return &v.String
	}

	return nil
}
