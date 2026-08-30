package link

import (
	"bytes"
	"io"
)

type prefixedPipe struct {
	Pipe
	reader io.Reader
}

func (pipe *prefixedPipe) Read(target []byte) (int, error) {
	return pipe.reader.Read(target)
}

func PrependRead(pipe Pipe, prefix []byte) Pipe {
	if pipe == nil || len(prefix) == 0 {
		return pipe
	}
	ownedPrefix := append([]byte(nil), prefix...)
	return &prefixedPipe{Pipe: pipe, reader: io.MultiReader(bytes.NewReader(ownedPrefix), pipe)}
}
