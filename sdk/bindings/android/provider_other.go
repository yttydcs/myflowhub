//go:build !android

package androidbinding

import "errors"

type RFCOMMPipe interface {
	Read(data []byte) (int, error)
	Write(data []byte) (int, error)
	Close() error
	RemoteBDAddr() string
}

type RFCOMMListener interface {
	Accept() (RFCOMMPipe, error)
	Close() error
	Addr() string
}

type RFCOMMProvider interface {
	Listen(uuid string, secure bool) (RFCOMMListener, error)
	Dial(bdaddr, uuid string, channel int, secure bool) (RFCOMMPipe, error)
}

func SetRFCOMMProvider(RFCOMMProvider) error {
	return errors.New("Android RFCOMM provider is unavailable on this platform")
}
