package models

import "errors"

const (
	DefaultMaxChangedLines = 1000
	DefaultMaxFileLines    = 3000
	DefaultMaxFileKB       = 1024
)

var ErrFileTooLarge = errors.New("file is too large to show")

type DiffLimits struct {
	MaxChangedLines int
	MaxFileLines    int
	MaxFileKB       int
	Collapse        []string
}

func (l DiffLimits) WithDefaults() DiffLimits {
	if l.MaxChangedLines <= 0 {
		l.MaxChangedLines = DefaultMaxChangedLines
	}
	if l.MaxFileLines <= 0 {
		l.MaxFileLines = DefaultMaxFileLines
	}
	if l.MaxFileKB <= 0 {
		l.MaxFileKB = DefaultMaxFileKB
	}
	return l
}

func (l DiffLimits) MaxFileBytes() int64 {
	return int64(l.WithDefaults().MaxFileKB) * 1024
}
