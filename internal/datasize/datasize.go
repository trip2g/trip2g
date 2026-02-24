// Package datasize implements a type representing data sizes in bytes.
//
// Adapted from https://github.com/kythe/kythe/blob/master/kythe/go/util/datasize/datasize.go
// Copyright 2015 The Kythe Authors. Apache License 2.0.
package datasize

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type sizeFlag struct{ *Size }

// Flag defines a Size flag with specified name, default value, and usage string.
func Flag(name, value, description string) *Size {
	sz, err := Parse(value)
	if err != nil {
		panic(fmt.Sprintf("Invalid default Size value for flag --%q: %q", name, value))
	}
	return FlagVar(flag.CommandLine, &sz, name, sz, description)
}

// FlagVar defines a Size flag with specified name, default value, and usage string
// into the provided FlagSet.
func FlagVar(fs *flag.FlagSet, s *Size, name string, value Size, description string) *Size {
	*s = value
	f := &sizeFlag{s}
	fs.Var(f, name, description)
	return f.Size
}

// Get implements part of the flag.Getter interface.
func (f *sizeFlag) Get() any {
	return *f.Size
}

// Set implements part of the flag.Value interface.
func (f *sizeFlag) Set(s string) error {
	sz, err := Parse(s)
	if err != nil {
		return err
	}
	*f.Size = sz
	return nil
}

// String implements part of the flag.Value interface.
func (f *sizeFlag) String() string {
	if f.Size == nil {
		return "0B"
	}
	return f.Size.String()
}

// Size represents the size of data in bytes.
type Size uint64

var sizeRE = regexp.MustCompile(`([0-9]*)(\.[0-9]*)?([a-z]+)`)

// Parse parses a Size from a string. A Size is an unsigned decimal number with
// an optional fraction and a unit suffix. Examples: "0", "10B", "1kB", "4GB",
// "5GiB". Valid units are "B", (decimal: "kB", "MB", "GB", "TB", "PB"),
// (binary: "KiB", "MiB", "GiB", "TiB", "PiB").
func Parse(s string) (Size, error) {
	if s == "" {
		return 0, errors.New("datasize: invalid Size: empty")
	}

	num, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return Size(num), nil
	}

	ss := sizeRE.FindStringSubmatch(strings.ToLower(s))
	if len(ss) == 0 {
		return 0, fmt.Errorf("datasize: invalid Size format %q", s)
	}

	num, err = strconv.ParseFloat(ss[1]+ss[2], 64)
	if err != nil {
		return 0, err
	}

	sz, err := suffixSize(ss[3])
	if err != nil {
		return 0, err
	}

	return Size(num * float64(sz)), nil
}

func suffixSize(suffix string) (Size, error) {
	switch suffix {
	case "b":
		return Byte, nil
	case "kb":
		return Kilobyte, nil
	case "mb":
		return Megabyte, nil
	case "gb":
		return Gigabyte, nil
	case "tb":
		return Terabyte, nil
	case "pb":
		return Petabyte, nil
	case "kib":
		return Kibibyte, nil
	case "mib":
		return Mebibyte, nil
	case "gib":
		return Gibibyte, nil
	case "tib":
		return Tebibyte, nil
	case "pib":
		return Pebibyte, nil
	default:
		return 0, fmt.Errorf("unknown datasize unit suffix: %q", suffix)
	}
}

// allUnits lists all units from highest to lowest (excluding Byte).
var allUnits = []Size{ //nolint:gochecknoglobals // used for formatting only
	Pebibyte,
	Petabyte,
	Tebibyte,
	Terabyte,
	Gibibyte,
	Gigabyte,
	Mebibyte,
	Megabyte,
	Kibibyte,
	Kilobyte,
}

// Common decimal data sizes.
const (
	Kilobyte Size = 1000 * Byte
	Megabyte      = 1000 * Kilobyte
	Gigabyte      = 1000 * Megabyte
	Terabyte      = 1000 * Gigabyte
	Petabyte      = 1000 * Terabyte
)

// Common binary data sizes.
const (
	Byte     Size = 1
	Kibibyte      = 1024 * Byte
	Mebibyte      = 1024 * Kibibyte
	Gibibyte      = 1024 * Mebibyte
	Tebibyte      = 1024 * Gibibyte
	Pebibyte      = 1024 * Tebibyte
)

// Floor returns a Size nearest to a whole unit less than or equal to itself.
func (s Size) Floor() Size {
	for _, unit := range allUnits {
		if s >= unit {
			return (s / unit) * unit
		}
	}
	return s
}

// Round returns a Size nearest to a whole unit.
func (s Size) Round() Size {
	for _, unit := range allUnits {
		if s >= unit {
			return Size(math.Round(float64(s)/float64(unit))) * unit
		}
	}
	return s
}

// String implements the Stringer interface.
func (s Size) String() string {
	switch {
	case s == 0:
		return "0B"
	case s%Petabyte == 0:
		return format(s.Petabytes(), "PB")
	case s >= Pebibyte:
		return format(s.Pebibytes(), "PiB")
	case s%Terabyte == 0:
		return format(s.Terabytes(), "TB")
	case s >= Tebibyte:
		return format(s.Tebibytes(), "TiB")
	case s%Gigabyte == 0:
		return format(s.Gigabytes(), "GB")
	case s >= Gibibyte:
		return format(s.Gibibytes(), "GiB")
	case s%Megabyte == 0:
		return format(s.Megabytes(), "MB")
	case s >= Mebibyte:
		return format(s.Mebibytes(), "MiB")
	case s%Kilobyte == 0:
		return format(s.Kilobytes(), "kB")
	case s >= Kibibyte:
		return format(s.Kibibytes(), "KiB")
	}
	return fmt.Sprintf("%dB", s)
}

func format(sz float64, suffix string) string {
	if math.Floor(sz) == sz {
		return fmt.Sprintf("%d%s", int64(sz), suffix)
	}
	return fmt.Sprintf("%.2f%s", sz, suffix)
}

// Bytes returns s in the equivalent number of bytes.
func (s Size) Bytes() uint64 { return uint64(s) }

// Kilobytes returns s in the equivalent number of kilobytes.
func (s Size) Kilobytes() float64 { return float64(s) / float64(Kilobyte) }

// Megabytes returns s in the equivalent number of megabytes.
func (s Size) Megabytes() float64 { return float64(s) / float64(Megabyte) }

// Gigabytes returns s in the equivalent number of gigabytes.
func (s Size) Gigabytes() float64 { return float64(s) / float64(Gigabyte) }

// Terabytes returns s in the equivalent number of terabytes.
func (s Size) Terabytes() float64 { return float64(s) / float64(Terabyte) }

// Petabytes returns s in the equivalent number of petabytes.
func (s Size) Petabytes() float64 { return float64(s) / float64(Petabyte) }

// Kibibytes returns s in the equivalent number of kibibytes.
func (s Size) Kibibytes() float64 { return float64(s) / float64(Kibibyte) }

// Mebibytes returns s in the equivalent number of mebibytes.
func (s Size) Mebibytes() float64 { return float64(s) / float64(Mebibyte) }

// Gibibytes returns s in the equivalent number of gibibytes.
func (s Size) Gibibytes() float64 { return float64(s) / float64(Gibibyte) }

// Tebibytes returns s in the equivalent number of tebibytes.
func (s Size) Tebibytes() float64 { return float64(s) / float64(Tebibyte) }

// Pebibytes returns s in the equivalent number of pebibytes.
func (s Size) Pebibytes() float64 { return float64(s) / float64(Pebibyte) }
