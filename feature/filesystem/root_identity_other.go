//go:build !windows

package filesystem

import "os"

type rootIdentity struct {
	info os.FileInfo
}

func captureRootIdentity(info os.FileInfo) rootIdentity {
	return rootIdentity{info: info}
}

func (identity rootIdentity) matches(info os.FileInfo) bool {
	return os.SameFile(identity.info, info)
}
