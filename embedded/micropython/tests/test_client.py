import binascii
import json
import os
import sys
import tempfile
import unittest

ROOT = os.path.dirname(os.path.dirname(__file__))
sys.path.insert(0, ROOT)

from myflowhub import Client, Identity, protocol


class FakeCrypto:
    def generate_key(self):
        return bytes([0x11]) * 32, bytes([0x22]) * 64

    def random(self, length):
        return bytes(range(1, length + 1))

    def sign(self, private_key, message):
        assert private_key == bytes([0x22]) * 64
        assert message.startswith(b"MFH4-JOIN")
        return bytes([0x5A]) * 64

    def verify(self, public_key, message, signature):
        return public_key == bytes([0x44]) * 32 and message.startswith(b"MFH4-ACK") and signature == bytes([0x33]) * 64


class ScriptedTransport:
    def __init__(self):
        self.connected = False
        self.inbound = bytearray()
        self.writes = []

    def connect(self):
        self.connected = True

    def close(self):
        self.connected = False

    def write_all(self, frame):
        if not self.connected:
            raise OSError("offline")
        request = protocol.decode(frame)
        self.writes.append(request)
        message_id = bytes([0x80]) * 16
        if request["operation"] == protocol.OP_JOIN:
            claim = json.loads(request["payload"].decode())
            payload = json.dumps({
                "node_id": 1, "child_id": 2, "child_nonce": claim["nonce"], "topology_epoch": claim["topology_epoch"],
                "signature": binascii.b2a_base64(bytes([0x33]) * 64).strip().decode(),
            }, separators=(",", ":")).encode()
            response = self.response(protocol.OP_JOIN_ACK, request, message_id, payload, "join-ack.v1")
            self.inbound.extend(protocol.encode(response))
        elif request["operation"] == protocol.OP_SUBSCRIBE:
            payload = b'{"revision":1,"value":"eyJ2ZXJzaW9uIjoxfQ=="}'
            response = self.response(protocol.OP_RESOURCE_EVENT, request, message_id, payload, "mfh.resource-event.v2", request["resource_owner"], request["resource_name"])
            self.inbound.extend(protocol.encode(response))
        elif request["operation"] == protocol.OP_OPERATE:
            response = self.response(protocol.OP_OPERATE_RESULT, request, message_id, b'{"version":1,"status":"ok"}', request["schema"], request["resource_owner"], request["resource_name"])
            self.inbound.extend(protocol.encode(response))

    def response(self, operation, request, message_id, payload, schema, owner=0, name=""):
        return {
            "version": protocol.VERSION, "phase": protocol.PHASE_RESPONSE, "operation": operation,
            "message_id": message_id, "correlation_id": request["message_id"], "source": request["target"], "principal": 0,
            "target": request["source"], "resource_owner": owner, "topology_epoch": 0, "deadline_unix_ms": 0,
            "content_type": "application/json", "schema": schema, "capability": request.get("capability", ""), "resource_name": name, "payload": payload,
        }

    def read_exact(self, length):
        if len(self.inbound) < length:
            raise OSError("scripted transport underflow")
        value = bytes(self.inbound[:length])
        del self.inbound[:length]
        return value


class ClientTests(unittest.TestCase):
    def setUp(self):
        self.crypto = FakeCrypto()
        self.identity = Identity(2, bytes([0x11]) * 32, bytes([0x22]) * 64)
        self.transport = ScriptedTransport()
        self.client = Client(self.identity, 1, bytes([0x44]) * 32, self.transport, self.crypto)

    def test_identity_persists_and_rejects_node_mismatch(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "identity.json")
            first = Identity.load_or_create(path, 2, self.crypto)
            second = Identity.load_or_create(path, 2, self.crypto)
            self.assertEqual(first.public_key, second.public_key)
            with self.assertRaises(ValueError):
                Identity.load_or_create(path, 3, self.crypto)

    def test_join_subscribe_operate_and_reconnect(self):
        self.client.connect()
        self.assertTrue(self.client.joined)
        subscription = self.client.subscribe(1, "system/health")
        event = self.client.poll()
        self.assertEqual(event["operation"], protocol.OP_RESOURCE_EVENT)
        self.assertEqual(event["correlation_id"], subscription)
        result = self.client.operate(1, "device/led/set", "invoke", "mfh.device.led.v1", {"version": 1, "value": "true"}, 1000)
        self.assertEqual(result["status"], "ok")
        before = len([item for item in self.transport.writes if item["operation"] == protocol.OP_SUBSCRIBE])
        self.client.reconnect(1, lambda _: None)
        after = len([item for item in self.transport.writes if item["operation"] == protocol.OP_SUBSCRIBE])
        self.assertEqual(after, before + 1)

    def test_unjoined_operations_fail(self):
        with self.assertRaises(OSError):
            self.client.subscribe(1, "system/health")


if __name__ == "__main__":
    unittest.main()
