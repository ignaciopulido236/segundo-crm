//go:build !windows

package crm

import (
	"errors"
	"os"
)

func protect([]byte) ([]byte, error)    { return nil, errors.New("DPAPI requires Windows") }
func unprotect([]byte) ([]byte, error)  { return nil, errors.New("DPAPI requires Windows") }
func replaceFile(from, to string) error { return os.Rename(from, to) }
