//go:build windows

package crm

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

type blob struct {
	Size uint32
	Data *byte
}

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func crypt(in []byte, decrypt bool) ([]byte, error) {
	if len(in) == 0 {
		return nil, errors.New("empty credential")
	}
	input := blob{uint32(len(in)), &in[0]}
	var output blob
	proc := crypt32.NewProc("CryptProtectData")
	if decrypt {
		proc = crypt32.NewProc("CryptUnprotectData")
	}
	// Flags=CRYPTPROTECT_UI_FORBIDDEN, never CRYPTPROTECT_LOCAL_MACHINE.
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&output)))
	runtime.KeepAlive(in)
	if ok == 0 {
		return nil, errors.New("DPAPI failed")
	}
	defer kernel32.NewProc("LocalFree").Call(uintptr(unsafe.Pointer(output.Data)))
	result := append([]byte(nil), unsafe.Slice(output.Data, output.Size)...)
	clear(unsafe.Slice(output.Data, output.Size))
	return result, nil
}
func protect(b []byte) ([]byte, error)   { return crypt(b, false) }
func unprotect(b []byte) ([]byte, error) { return crypt(b, true) }
func replaceFile(from, to string) error {
	f, e := syscall.UTF16PtrFromString(from)
	if e != nil {
		return e
	}
	t, e := syscall.UTF16PtrFromString(to)
	if e != nil {
		return e
	}
	ok, _, err := kernel32.NewProc("MoveFileExW").Call(uintptr(unsafe.Pointer(f)), uintptr(unsafe.Pointer(t)), 0x1|0x8)
	if ok == 0 {
		return err
	}
	return nil
}
