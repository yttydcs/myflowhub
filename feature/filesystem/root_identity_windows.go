//go:build windows

package filesystem

import (
	"os"
	"syscall"
)

type rootIdentity struct {
	info              os.FileInfo
	creationTimeNanos int64
}

func captureRootIdentity(info os.FileInfo) rootIdentity {
	identity := rootIdentity{info: info}
	if attributes, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		identity.creationTimeNanos = attributes.CreationTime.Nanoseconds()
	}
	return identity
}

func (identity rootIdentity) matches(info os.FileInfo) bool {
	if !os.SameFile(identity.info, info) {
		return false
	}
	attributes, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && identity.creationTimeNanos != 0 && identity.creationTimeNanos == attributes.CreationTime.Nanoseconds()
}
