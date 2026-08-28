try:
    import ustruct as struct
except ImportError:
    import struct

MAGIC = b"MFH4"
VERSION = 2
HEADER_SIZE = 98
MAX_PAYLOAD = 8192
MAX_RESOURCE_NAME = 255
MAX_CONTENT_TYPE = 127
MAX_SCHEMA = 255
MAX_CAPABILITY = 128

PHASE_REQUEST = 1
PHASE_CONTROL = 2
PHASE_RESPONSE = 3
PHASE_EVENT = 4

OP_JOIN = 1
OP_JOIN_ACK = 2
OP_SUBSCRIBE = 5
OP_UNSUBSCRIBE = 6
OP_SUBSCRIBE_ACK = 7
OP_RESOURCE_EVENT = 8
OP_RESOURCE_GAP = 9
OP_OPERATE = 10
OP_OPERATE_RESULT = 11
OP_SESSION_OPEN = 12
OP_SESSION_OPEN_RESULT = 13
OP_SESSION_DATA = 14
OP_SESSION_CLOSE = 15
OP_ERROR = 16
OP_HEARTBEAT = 17

_HEADER = ">4sHBBBBHHIQQQQq16s16sQ"


def _phase_allows(phase, operation):
    if operation == OP_JOIN:
        return phase == PHASE_REQUEST
    if operation in (OP_JOIN_ACK, OP_SUBSCRIBE_ACK, OP_OPERATE_RESULT, OP_SESSION_OPEN_RESULT, OP_ERROR):
        return phase == PHASE_RESPONSE
    if operation in (3, 4, OP_RESOURCE_GAP, OP_HEARTBEAT):
        return phase == PHASE_EVENT
    if operation == OP_RESOURCE_EVENT:
        return phase in (PHASE_RESPONSE, PHASE_EVENT)
    if operation in (OP_SESSION_DATA, OP_SESSION_CLOSE):
        return phase in (PHASE_REQUEST, PHASE_CONTROL, PHASE_RESPONSE)
    if operation in (OP_SUBSCRIBE, OP_UNSUBSCRIBE, OP_OPERATE, OP_SESSION_OPEN):
        return phase in (PHASE_REQUEST, PHASE_CONTROL)
    return False


def validate(envelope, max_payload=MAX_PAYLOAD):
    if envelope.get("version") != VERSION:
        raise ValueError("unsupported MFH4 version")
    phase, operation = envelope.get("phase"), envelope.get("operation")
    if not _phase_allows(phase, operation):
        raise ValueError("phase does not allow operation")
    message_id = envelope.get("message_id", b"")
    correlation_id = envelope.get("correlation_id", bytes(16))
    if len(message_id) != 16 or not any(message_id) or len(correlation_id) != 16:
        raise ValueError("message identifiers are invalid")
    if not envelope.get("source") or not envelope.get("target"):
        raise ValueError("source and target are required")
    requires_resource = OP_SUBSCRIBE <= operation <= OP_SESSION_CLOSE
    name = envelope.get("resource_name", "")
    owner = envelope.get("resource_owner", 0)
    if requires_resource != bool(owner and name):
        raise ValueError("resource identity does not match operation")
    capability = envelope.get("capability", "")
    if requires_resource != bool(capability) or len(capability.encode()) > MAX_CAPABILITY:
        raise ValueError("resource capability does not match operation")
    if len(name.encode()) > MAX_RESOURCE_NAME or name.startswith("/") or name.endswith("/") or "//" in name:
        raise ValueError("resource name is invalid")
    if any(part in ("", ".", "..") for part in name.split("/")) and name:
        raise ValueError("resource segments are invalid")
    content_type = envelope.get("content_type", "").encode()
    schema = envelope.get("schema", "").encode()
    payload = envelope.get("payload", b"")
    if len(content_type) > MAX_CONTENT_TYPE or len(schema) > MAX_SCHEMA or len(payload) > max_payload:
        raise ValueError("envelope exceeds embedded profile limits")
    if envelope.get("deadline_unix_ms", 0) < 0:
        raise ValueError("deadline cannot be negative")
    if operation in (OP_JOIN_ACK, OP_SUBSCRIBE_ACK, OP_RESOURCE_EVENT, OP_RESOURCE_GAP, OP_OPERATE_RESULT,
                     OP_SESSION_OPEN_RESULT, OP_SESSION_DATA, OP_SESSION_CLOSE, OP_ERROR) and not any(correlation_id):
        raise ValueError("correlation identifier is required")


def encode(envelope, max_payload=MAX_PAYLOAD):
    validate(envelope, max_payload)
    content_type = envelope.get("content_type", "").encode()
    schema = envelope.get("schema", "").encode()
    capability = envelope.get("capability", "").encode()
    name = envelope.get("resource_name", "").encode()
    payload = envelope.get("payload", b"")
    header = struct.pack(
        _HEADER, MAGIC, VERSION, envelope["phase"], envelope["operation"], len(content_type), len(schema), len(capability), len(name), len(payload),
        envelope["source"], envelope.get("principal", 0), envelope["target"], envelope.get("topology_epoch", 0),
        envelope.get("deadline_unix_ms", 0), envelope["message_id"], envelope.get("correlation_id", bytes(16)), envelope.get("resource_owner", 0),
    )
    return header + content_type + schema + capability + name + payload


def decode(frame, max_payload=MAX_PAYLOAD):
    if len(frame) < HEADER_SIZE:
        raise ValueError("truncated MFH4 header")
    values = struct.unpack(_HEADER, frame[:HEADER_SIZE])
    magic, version, phase, operation, content_len, schema_len, capability_len, name_len, payload_len = values[:9]
    if magic != MAGIC:
        raise ValueError("invalid MFH4 magic")
    if content_len > MAX_CONTENT_TYPE or schema_len > MAX_SCHEMA or capability_len > MAX_CAPABILITY or name_len > MAX_RESOURCE_NAME or payload_len > max_payload:
        raise ValueError("frame metadata or payload exceeds embedded profile")
    total = HEADER_SIZE + content_len + schema_len + capability_len + name_len + payload_len
    if len(frame) != total:
        raise ValueError("truncated or trailing MFH4 frame")
    offset = HEADER_SIZE
    content_type = frame[offset:offset + content_len].decode(); offset += content_len
    schema = frame[offset:offset + schema_len].decode(); offset += schema_len
    capability = frame[offset:offset + capability_len].decode(); offset += capability_len
    name = frame[offset:offset + name_len].decode(); offset += name_len
    envelope = {
        "version": version, "phase": phase, "operation": operation,
        "source": values[9], "principal": values[10], "target": values[11], "topology_epoch": values[12],
        "deadline_unix_ms": values[13], "message_id": values[14], "correlation_id": values[15], "resource_owner": values[16],
        "content_type": content_type, "schema": schema, "capability": capability, "resource_name": name, "payload": frame[offset:offset + payload_len],
    }
    validate(envelope, max_payload)
    return envelope


def read_frame(transport, max_payload=MAX_PAYLOAD):
    header = transport.read_exact(HEADER_SIZE)
    if len(header) != HEADER_SIZE or header[:4] != MAGIC:
        raise OSError("invalid or truncated MFH4 header")
    content_len, schema_len = header[8], header[9]
    capability_len = struct.unpack(">H", header[10:12])[0]
    name_len = struct.unpack(">H", header[12:14])[0]
    payload_len = struct.unpack(">I", header[14:18])[0]
    if payload_len > max_payload:
        raise ValueError("frame payload exceeds embedded profile")
    body_len = content_len + schema_len + capability_len + name_len + payload_len
    if body_len > MAX_CONTENT_TYPE + MAX_SCHEMA + MAX_CAPABILITY + MAX_RESOURCE_NAME + max_payload:
        raise ValueError("frame body exceeds embedded profile")
    return decode(header + transport.read_exact(body_len), max_payload)
