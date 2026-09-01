//go:build windows

package secretstore

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

type windowsStore struct {
	service string
}

type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32   = windows.NewLazySystemDLL("advapi32.dll")
	credWrite  = advapi32.NewProc("CredWriteW")
	credRead   = advapi32.NewProc("CredReadW")
	credDelete = advapi32.NewProc("CredDeleteW")
	credFree   = advapi32.NewProc("CredFree")
)

func Open(service string) Store {
	return &windowsStore{service: service}
}

func (s *windowsStore) target(key string) (*uint16, error) {
	return windows.UTF16PtrFromString(s.service + ":" + key)
}

func (s *windowsStore) Set(key, value string) error {
	target, err := s.target(key)
	if err != nil {
		return fmt.Errorf("credential target: %w", err)
	}
	user, _ := windows.UTF16PtrFromString("cc-connect")
	blob := []byte(value)
	var blobPtr *byte
	if len(blob) > 0 {
		blobPtr = &blob[0]
	}
	cred := credential{
		Type:               credTypeGeneric,
		TargetName:         target,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     blobPtr,
		Persist:            credPersistLocalMachine,
		UserName:           user,
	}
	r1, _, callErr := credWrite.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if r1 == 0 {
		return fmt.Errorf("write Windows credential: %w", callErr)
	}
	return nil
}

func (s *windowsStore) Get(key string) (string, error) {
	target, err := s.target(key)
	if err != nil {
		return "", fmt.Errorf("credential target: %w", err)
	}
	var ptr *credential
	r1, _, callErr := credRead.Call(
		uintptr(unsafe.Pointer(target)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&ptr)),
	)
	if r1 == 0 {
		return "", fmt.Errorf("read Windows credential: %w", callErr)
	}
	defer credFree.Call(uintptr(unsafe.Pointer(ptr)))
	if ptr.CredentialBlobSize == 0 || ptr.CredentialBlob == nil {
		return "", nil
	}
	blob := unsafe.Slice(ptr.CredentialBlob, int(ptr.CredentialBlobSize))
	return string(blob), nil
}

func (s *windowsStore) Delete(key string) error {
	target, err := s.target(key)
	if err != nil {
		return fmt.Errorf("credential target: %w", err)
	}
	r1, _, callErr := credDelete.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if r1 == 0 && callErr != syscall.ERROR_NOT_FOUND {
		return fmt.Errorf("delete Windows credential: %w", callErr)
	}
	return nil
}
