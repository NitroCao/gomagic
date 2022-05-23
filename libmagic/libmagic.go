package libmagic

// #cgo pkg-config: libmagic
// #include <magic.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

type Magic struct {
	handle C.magic_t
	lock   *sync.Mutex
}

const (
	MagicNone  = 0
	MagicDebug = 1 << (iota - 1)
	MagicSymlink
	MagicCompress
	MagicDevices
	MagicMimeType
	MagicContinue
	MagicCheck
	MagicPreserveAtime
	MagicRaw
	MagicError
	MagicMimeEncoding
	MagicMime
	MagicApple
	MagicNoCheckCompress
	MagicNoCheckTar
	MagicNoCheckSoft
	MagicNoCheckAppType
	MagicNoCheckElf
	MagicNoCheckText
	MagicNoCheckCdf
	MagicNoCheckTokens
	MagicNoCheckEncoding
)

func NewMagic(flags int) (*Magic, error) {
	handle := C.magic_open(C.int(flags))
	if handle == nil {
		return nil, fmt.Errorf("failed to create a magic cookie")
	}

	return &Magic{
		handle: handle,
		lock:   &sync.Mutex{},
	}, nil
}

func (m *Magic) MagicLoad(files []string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	cFiles := prepareFiles(files)
	if cFiles != nil {
		defer C.free(unsafe.Pointer(cFiles))
	}
	if C.magic_load(m.handle, cFiles) == C.int(-1) {
		return m.magicError("failed to load database files")
	}
	return nil
}

func (m *Magic) Close() {
	m.lock.Lock()
	m.lock.Unlock()
	if m.handle != nil {
		C.magic_close(m.handle)
	}
}

func (m *Magic) MagicFile(filename string) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	result := C.magic_file(m.handle, cFilename)
	if result == nil {
		return "", m.magicError(fmt.Sprintf("failed to detect file %s", filename))
	}
	return C.GoString(result), nil
}

func (m *Magic) MagicBuffer(content []byte) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cContent := C.CBytes(content)
	defer C.free(cContent)

	result := C.magic_buffer(m.handle, cContent, C.ulong(len(content)))
	if result == nil {
		return "", m.magicError("failed to detect buffer")
	}
	return C.GoString(result), nil
}

func (m *Magic) MagicDescriptor(fd int) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	result := C.magic_descriptor(m.handle, C.int(fd))
	if result == nil {
		return "", m.magicError("failed to detect fd")
	}
	return C.GoString(result), nil
}

func (m *Magic) MagicCompile(files []string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	cFiles := prepareFiles(files)
	if cFiles != nil {
		defer C.free(unsafe.Pointer(cFiles))
	}

	if C.magic_compile(m.handle, cFiles) == C.int(-1) {
		return m.magicError("failed to load database files")
	}
	return nil
}

func (m *Magic) magicError(errStr string) error {
	err := C.magic_error(m.handle)
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", errStr, C.GoString(err))
}

func (m *Magic) MagicList(files []string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	cFiles := prepareFiles(files)
	if cFiles != nil {
		defer C.free(unsafe.Pointer(cFiles))
	}

	if C.magic_list(m.handle, cFiles) == C.int(-1) {
		return m.magicError("failed to list entries")
	}
	return nil
}

func (m *Magic) MagicCheck(files []string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	cFiles := prepareFiles(files)
	if cFiles != nil {
		defer C.free(unsafe.Pointer(cFiles))
	}
	if C.magic_check(m.handle, cFiles) == C.int(-1) {
		return m.magicError("invalid database files")
	}
	return nil
}

func (m *Magic) MagicGetFlags() int {
	m.lock.Lock()
	defer m.lock.Unlock()
	return int(C.magic_getflags(m.handle))
}

func (m *Magic) MagicSetFlags(flags int) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if C.magic_setflags(m.handle, C.int(flags)) == C.int(-1) {
		return m.magicError("failed to set flags")
	}
	return nil
}

func prepareFiles(files []string) (cFiles *C.char) {
	if len(files) != 0 {
		cFiles = C.CString(strings.Join(files, ":"))
	}

	return
}
