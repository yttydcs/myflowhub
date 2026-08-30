package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const fixedHeaderSize = 98

const FrameMagic = "MFH4"

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
	copy(header[:4], FrameMagic)
	binary.BigEndian.PutUint16(header[4:6], envelope.Version)
	header[6] = byte(envelope.Phase)
	header[7] = byte(envelope.Operation)
	header[8] = byte(len(envelope.ContentType))
	header[9] = byte(len(envelope.Schema))
	binary.BigEndian.PutUint16(header[10:12], uint16(len(envelope.Capability)))
	binary.BigEndian.PutUint16(header[12:14], uint16(len(envelope.Resource.Name)))
	binary.BigEndian.PutUint32(header[14:18], uint32(len(envelope.Payload)))
	binary.BigEndian.PutUint64(header[18:26], uint64(envelope.Source))
	binary.BigEndian.PutUint64(header[26:34], uint64(envelope.Principal))
	binary.BigEndian.PutUint64(header[34:42], uint64(envelope.Target))
	binary.BigEndian.PutUint64(header[42:50], envelope.TopologyEpoch)
	binary.BigEndian.PutUint64(header[50:58], uint64(envelope.DeadlineUnixMS))
	copy(header[58:74], envelope.MessageID[:])
	copy(header[74:90], envelope.CorrelationID[:])
	binary.BigEndian.PutUint64(header[90:98], uint64(envelope.Resource.Owner))
	for _, part := range [][]byte{header, []byte(envelope.ContentType), []byte(envelope.Schema), []byte(envelope.Capability), []byte(envelope.Resource.Name), envelope.Payload} {
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
	if string(header[:4]) != FrameMagic {
		return Envelope{}, errors.New("decode frame: invalid magic")
	}
	contentLen := int(header[8])
	schemaLen := int(header[9])
	capabilityLen := int(binary.BigEndian.Uint16(header[10:12]))
	nameLen := int(binary.BigEndian.Uint16(header[12:14]))
	payloadLength := binary.BigEndian.Uint32(header[14:18])
	if contentLen > MaxContentTypeBytes || schemaLen > MaxSchemaBytes || capabilityLen > MaxCapabilityBytes || nameLen > MaxResourceNameBytes {
		return Envelope{}, errors.New("decode frame: metadata exceeds protocol limit")
	}
	if uint64(payloadLength) > uint64(c.payloadLimit()) {
		return Envelope{}, fmt.Errorf("decode frame: %w: got %d, max %d", ErrPayloadTooLarge, payloadLength, c.payloadLimit())
	}
	payloadLen := int(payloadLength)
	metadataAndPayload := make([]byte, contentLen+schemaLen+capabilityLen+nameLen+payloadLen)
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
		Source:         NodeID(binary.BigEndian.Uint64(header[18:26])),
		Principal:      NodeID(binary.BigEndian.Uint64(header[26:34])),
		Target:         NodeID(binary.BigEndian.Uint64(header[34:42])),
		TopologyEpoch:  binary.BigEndian.Uint64(header[42:50]),
		DeadlineUnixMS: int64(binary.BigEndian.Uint64(header[50:58])),
		Resource: ResourceID{
			Owner: NodeID(binary.BigEndian.Uint64(header[90:98])),
		},
	}
	copy(envelope.MessageID[:], header[58:74])
	copy(envelope.CorrelationID[:], header[74:90])
	envelope.ContentType = string(take(contentLen))
	envelope.Schema = string(take(schemaLen))
	envelope.Capability = CapabilityID(take(capabilityLen))
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
