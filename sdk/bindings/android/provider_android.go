//go:build android

package androidbinding

import "github.com/yttydcs/myflowhub/transport/rfcomm"

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

func SetRFCOMMProvider(provider RFCOMMProvider) {
	if provider == nil {
		rfcomm.SetAndroidRFCOMMProvider(nil)
		return
	}
	rfcomm.SetAndroidRFCOMMProvider(providerAdapter{provider})
}

type providerAdapter struct{ RFCOMMProvider }

func (a providerAdapter) Listen(uuid string, secure bool) (rfcomm.AndroidRFCOMMListener, error) {
	listener, err := a.RFCOMMProvider.Listen(uuid, secure)
	if err != nil {
		return nil, err
	}
	return listenerAdapter{listener}, nil
}

func (a providerAdapter) Dial(bdaddr, uuid string, channel int, secure bool) (rfcomm.AndroidRFCOMMPipe, error) {
	pipe, err := a.RFCOMMProvider.Dial(bdaddr, uuid, channel, secure)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

type listenerAdapter struct{ RFCOMMListener }

func (a listenerAdapter) Accept() (rfcomm.AndroidRFCOMMPipe, error) {
	pipe, err := a.RFCOMMListener.Accept()
	if err != nil {
		return nil, err
	}
	return pipe, nil
}
