import json
import os
import sys
import unittest

ROOT = os.path.dirname(os.path.dirname(__file__))
sys.path.insert(0, ROOT)

from myflowhub import protocol


class ProtocolTests(unittest.TestCase):
    def sample(self):
        return {
            "version": 1, "phase": protocol.PHASE_REQUEST, "operation": protocol.OP_COMMAND_CALL,
            "message_id": bytes([16] + list(range(1, 16))), "correlation_id": bytes(16),
            "source": 2, "principal": 0, "target": 1, "resource_owner": 1,
            "topology_epoch": 0, "deadline_unix_ms": 1000,
            "content_type": "application/json", "schema": "mfh.test.v1", "resource_name": "device/led/set",
            "payload": b'{"version":1,"value":"true"}',
        }

    def test_codec_matches_mfh3_layout(self):
        frame = protocol.encode(self.sample())
        fixture = os.path.join(ROOT, "..", "..", "tests", "fixtures", "embedded", "envelope-command-v1.hex")
        with open(fixture, "r", encoding="ascii") as source:
            self.assertEqual(frame.hex(), source.read().strip())
        self.assertEqual(len(frame), 96 + 16 + 11 + 14 + 28)
        self.assertEqual(protocol.decode(frame), self.sample())

    def test_codec_rejects_oversize_and_trailing_data(self):
        sample = self.sample()
        sample["payload"] = bytes(protocol.MAX_PAYLOAD + 1)
        with self.assertRaises(ValueError):
            protocol.encode(sample)
        valid = protocol.encode(self.sample())
        with self.assertRaises(ValueError):
            protocol.decode(valid + b"x")

    def test_shared_payload_fixture(self):
        fixture = os.path.join(ROOT, "..", "..", "tests", "fixtures", "protocol", "resource-catalog-v1.json")
        with open(fixture, "r", encoding="utf-8") as source:
            value = json.load(source)
        self.assertEqual(value["version"], 1)
        self.assertEqual([item["kind"] for item in value["resources"]], ["variable", "variable"])
        self.assertEqual(value["resources"][1]["name"], "system/catalog")


if __name__ == "__main__":
    unittest.main()
