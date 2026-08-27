package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const fixedHeaderSize = 96

var frameMagic = [4]byte{'M', 'F', 'H', '3'}

type Codec struct {
	MaxPayload int
}

func (c Codec) payloadLimit() int {
	if c.MaxPayload <= 0 {
		return DefaultMaxPayload
	}
	return c.MaxPayload
}

func (c Codec) Encode(writer io.Writer, envelope Envelope) error {
	if writer == nil {
		return errors.New("encode frame: nil writer")
	}
	if err := envelope.Validate(c.payloadLimit()); err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}
	if uint64(len(envelope.Payload)) > math.MaxUint32 {
		return fmt.Errorf("encode frame: %w: wire length exceeds uint32", ErrPayloadTooLarge)
	}
	header := make([]byte, fixedHeaderSize)
	copy(header[:4], frameMagic[:])
	binary.BigEndian.PutUint16(header[4:6], envelope.Version)
	header[6] = byte(envelope.Phase)
	header[7] = byte(envelope.Operation)
	header[8] = byte(len(envelope.ContentType))
	header[9] = byte(len(envelope.Schema))
	binary.BigEndian.PutUint16(header[10:12], uint16(len(envelope.Resource.Name)))
	binary.BigEndian.PutUint32(header[12:16], uint32(len(envelope.Payload)))
	binary.BigEndian.PutUint64(header[16:24], uint64(envelope.Source))
	binary.BigEndian.PutUint64(header[24:32], uint64(envelope.Principal))
	binary.BigEndian.PutUint64(header[32:40], uint64(envelope.Target))
	binary.BigEndian.PutUint64(header[40:48], envelope.TopologyEpoch)
	binary.BigEndian.PutUint64(header[48:56], uint64(envelope.DeadlineUnixMS))
	copy(header[56:72], envelope.MessageID[:])
	copy(header[72:88], envelope.CorrelationID[:])
	binary.BigEndian.PutUint64(header[88:96], uint64(envelope.Resource.Owner))
	for _, part := range [][]byte{header, []byte(envelope.ContentType), []byte(envelope.Schema), []byte(envelope.Resource.Name), envelope.Payload} {
		if err := writeAll(writer, part); err != nil {
			return fmt.Errorf("encode frame: %w", err)
		}
	}
	return nil
}

func (c Codec) Decode(reader io.Reader) (Envelope, error) {
	if reader == nil {
		return Envelope{}, errors.New("decode frame: nil reader")
	}
	header := make([]byte, fixedHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return Envelope{}, fmt.Errorf("decode frame header: %w", err)
	}
	if string(header[:4]) != string(frameMagic[:]) {
		return Envelope{}, errors.New("decode frame: invalid magic")
	}
	contentLen := int(header[8])
	schemaLen := int(header[9])
	nameLen := int(binary.BigEndian.Uint16(header[10:12]))
	payloadLength := binary.BigEndian.Uint32(header[12:16])
	if contentLen > MaxContentTypeBytes || schemaLen > MaxSchemaBytes || nameLen > MaxResourceNameBytes {
		return Envelope{}, errors.New("decode frame: metadata exceeds protocol limit")
	}
	if uint64(payloadLength) > uint64(c.payloadLimit()) {
		return Envelope{}, fmt.Errorf("decode frame: %w: got %d, max %d", ErrPayloadTooLarge, payloadLength, c.payloadLimit())
	}
	payloadLen := int(payloadLength)
	metadataAndPayload := make([]byte, contentLen+schemaLen+nameLen+payloadLen)
	if _, err := io.ReadFull(reader, metadataAndPayload); err != nil {
		return Envelope{}, fmt.Errorf("decode frame body: %w", err)
	}
	offset := 0
	take := func(length int) []byte {
		part := metadataAndPayload[offset : offset+length]
		offset += length
		return part
	}
	envelope := Envelope{
		Version:        binary.BigEndian.Uint16(header[4:6]),
		Phase:          Phase(header[6]),
		Operation:      Operation(header[7]),
		Source:         NodeID(binary.BigEndian.Uint64(header[16:24])),
		Principal:      NodeID(binary.BigEndian.Uint64(header[24:32])),
		Target:         NodeID(binary.BigEndian.Uint64(header[32:40])),
		TopologyEpoch:  binary.BigEndian.Uint64(header[40:48]),
		DeadlineUnixMS: int64(binary.BigEndian.Uint64(header[48:56])),
		Resource: ResourceID{
			Owner: NodeID(binary.BigEndian.Uint64(header[88:96])),
		},
	}
	copy(envelope.MessageID[:], header[56:72])
	copy(envelope.CorrelationID[:], header[72:88])
	envelope.ContentType = string(take(contentLen))
	envelope.Schema = string(take(schemaLen))
	envelope.Resource.Name = string(take(nameLen))
	envelope.Payload = append([]byte(nil), take(payloadLen)...)
	if err := envelope.Validate(c.payloadLimit()); err != nil {
		return Envelope{}, fmt.Errorf("decode frame: %w", err)
	}
	return envelope, nil
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if written < 0 || written > len(data) {
			return errors.New("invalid writer count")
		}
		data = data[written:]
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}
