try:
    import ujson as json
except ImportError:
    import json
try:
    import ubinascii as binascii
except ImportError:
    import binascii
try:
    import ustruct as struct
except ImportError:
    import struct

from . import protocol


def _b64encode(value):
    return binascii.b2a_base64(value).strip().decode()


def _b64decode(value):
    return binascii.a2b_base64(value)


def _json(value):
    try:
        return json.dumps(value, separators=(",", ":"))
    except TypeError:
        return json.dumps(value)


class Client:
    def __init__(self, identity, parent_id, parent_public_key, transport, crypto, max_payload=protocol.MAX_PAYLOAD):
        if parent_id <= 0 or len(parent_public_key) != 32 or max_payload <= 0 or max_payload > protocol.MAX_PAYLOAD:
            raise ValueError("parent identity or payload limit is invalid")
        self.identity = identity
        self.parent_id = parent_id
        self.parent_public_key = bytes(parent_public_key)
        self.transport = transport
        self.crypto = crypto
        self.max_payload = max_payload
        self.epoch = 1
        self.joined = False
        self.subscriptions = {}

    def _message_id(self):
        value = bytes(self.crypto.random(16))
        if len(value) != 16 or not any(value):
            raise ValueError("crypto random returned invalid message ID")
        return value

    def _base(self, phase, operation, target, message_id=None):
        return {
            "version": protocol.VERSION, "phase": phase, "operation": operation, "message_id": message_id or self._message_id(),
            "correlation_id": bytes(16), "source": self.identity.node_id, "principal": 0, "target": target,
            "resource_owner": 0, "topology_epoch": 0, "deadline_unix_ms": 0,
            "content_type": "application/json", "schema": "", "capability": "", "resource_name": "", "payload": b"",
        }

    def _send(self, envelope):
        frame = protocol.encode(envelope, self.max_payload)
        if len(frame) > protocol.HEADER_SIZE + protocol.MAX_CONTENT_TYPE + protocol.MAX_SCHEMA + protocol.MAX_RESOURCE_NAME + self.max_payload:
            raise ValueError("frame exceeds embedded profile")
        self.transport.write_all(frame)

    def connect(self, permit=None):
        self.transport.connect()
        nonce = bytes(self.crypto.random(32))
        if len(nonce) != 32:
            raise ValueError("crypto random returned invalid join nonce")
        message = b"MFH4-JOIN" + struct.pack(">Q", self.identity.node_id) + nonce + struct.pack(">Q", self.epoch)
        signature = bytes(self.crypto.sign(self.identity.private_key, message))
        if len(signature) != 64:
            raise ValueError("crypto signer returned invalid Ed25519 signature")
        claim = {"node_id": self.identity.node_id, "public_key": _b64encode(self.identity.public_key), "nonce": list(nonce), "topology_epoch": self.epoch}
        if permit is not None:
            claim["permit"] = permit
        claim["signature"] = _b64encode(signature)
        request = self._base(protocol.PHASE_REQUEST, protocol.OP_JOIN, self.parent_id)
        request["schema"] = "join.v1"; request["payload"] = _json(claim).encode()
        self._send(request)
        response = protocol.read_frame(self.transport, self.max_payload)
        if response["operation"] != protocol.OP_JOIN_ACK or response["source"] != self.parent_id or response["target"] != self.identity.node_id or response["correlation_id"] != request["message_id"]:
            self.transport.close(); raise OSError("join acknowledgement does not match request")
        ack = json.loads(response["payload"].decode())
        if ack.get("node_id") != self.parent_id or ack.get("child_id") != self.identity.node_id or ack.get("child_nonce") != list(nonce) or ack.get("topology_epoch") != self.epoch:
            self.transport.close(); raise OSError("join acknowledgement identity mismatch")
        ack_message = b"MFH4-ACK" + struct.pack(">Q", self.parent_id) + struct.pack(">Q", self.identity.node_id) + nonce + struct.pack(">Q", self.epoch)
        if not self.crypto.verify(self.parent_public_key, ack_message, _b64decode(ack["signature"])):
            self.transport.close(); raise OSError("join acknowledgement signature invalid")
        self.joined = True
        previous = list(self.subscriptions.values())
        self.subscriptions = {}
        for owner, name, capability, lease_ms, queue in previous:
            self.subscribe(owner, name, capability, lease_ms, queue)

    def close(self):
        self.joined = False
        self.transport.close()

    def reconnect(self, attempts, sleep_ms, permit=None):
        last_error = None
        for attempt in range(attempts):
            try:
                self.close(); self.epoch += 1; self.connect(permit); return
            except Exception as error:
                last_error = error
                sleep_ms(min(30000, 250 * (2 ** attempt)))
        raise OSError("reconnect exhausted: %s" % last_error)

    def _resource(self, operation, owner, name, capability):
        if not self.joined or owner <= 0 or not name or not capability:
            raise OSError("client is not joined or resource is invalid")
        envelope = self._base(protocol.PHASE_REQUEST, operation, owner)
        envelope["resource_owner"] = owner; envelope["resource_name"] = name; envelope["capability"] = capability
        return envelope

    def subscribe(self, owner, name, capability="subscribe", lease_ms=60000, queue=8):
        if lease_ms <= 0 or queue <= 0 or queue > 64:
            raise ValueError("subscription lease or queue is invalid")
        request = self._resource(protocol.OP_SUBSCRIBE, owner, name, capability)
        request["schema"] = "mfh.subscribe.v2"; request["payload"] = _json({"version": 2, "lease_ms": lease_ms, "queue": queue}).encode()
        self._send(request)
        self.subscriptions[request["message_id"]] = (owner, name, capability, lease_ms, queue)
        return request["message_id"]

    def unsubscribe(self, subscription_id):
        details = self.subscriptions.get(subscription_id)
        if details is None:
            raise KeyError("subscription not found")
        request = self._resource(protocol.OP_UNSUBSCRIBE, details[0], details[1], details[2]); request["correlation_id"] = subscription_id
        request["schema"] = "unsubscribe.v1"; request["payload"] = b"{}"; self._send(request); del self.subscriptions[subscription_id]

    def operate(self, owner, name, capability, schema, request, deadline_unix_ms):
        envelope = self._resource(protocol.OP_OPERATE, owner, name, capability)
        envelope["schema"] = schema; envelope["deadline_unix_ms"] = deadline_unix_ms; envelope["payload"] = _json(request).encode(); self._send(envelope)
        while True:
            response = self.poll()
            if response["correlation_id"] != envelope["message_id"]:
                continue
            if response["operation"] == protocol.OP_ERROR:
                raise OSError("remote resource error: %s" % response["payload"].decode())
            if response["operation"] != protocol.OP_OPERATE_RESULT:
                raise OSError("unexpected resource operation response")
            return json.loads(response["payload"].decode())

    def poll(self):
        envelope = protocol.read_frame(self.transport, self.max_payload)
        if envelope["operation"] == protocol.OP_ERROR:
            return envelope
        if envelope["operation"] in (protocol.OP_RESOURCE_EVENT, protocol.OP_RESOURCE_GAP):
            details = self.subscriptions.get(envelope["correlation_id"])
            if details is None:
                raise OSError("event has unknown subscription")
        return envelope
