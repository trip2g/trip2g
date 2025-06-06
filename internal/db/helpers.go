package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

func IsNoFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func IsUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ToNullableFloat64 converts a float64 pointer to sql.NullFloat64.
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
	if value == nil || value.IsZero() {
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

// ToNullableInt64 converts a pointer to an int64 to sql.NullInt64.
func ToNullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{Valid: false}
	}

	return sql.NullInt64{
		Int64: *value,
		Valid: true,
	}
}

// ToFloat64Ptr converts a sql.NullFloat64 to a pointer to a float64.
func ToFloat64Ptr(v sql.NullFloat64) *float64 {
	if v.Valid {
		return &v.Float64
	}

	return nil
}

// ToInt64Ptr converts a sql.NullInt64 to a pointer to an int64.
func ToInt64Ptr(v sql.NullInt64) *int64 {
	if v.Valid {
		return &v.Int64
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
