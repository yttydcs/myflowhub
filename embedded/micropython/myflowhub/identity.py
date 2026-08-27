try:
    import ujson as json
except ImportError:
    import json
try:
    import ubinascii as binascii
except ImportError:
    import binascii
import os


def _b64encode(value):
    return binascii.b2a_base64(value).strip().decode()


def _b64decode(value):
    return binascii.a2b_base64(value)


class Identity:
    def __init__(self, node_id, public_key, private_key):
        if not isinstance(node_id, int) or node_id <= 0 or len(public_key) != 32 or len(private_key) != 64:
            raise ValueError("identity requires positive NodeID and Ed25519 key pair")
        self.node_id = node_id
        self.public_key = bytes(public_key)
        self.private_key = bytes(private_key)

    def to_json(self):
        return json.dumps({"version": 1, "node_id": self.node_id, "public_key": _b64encode(self.public_key), "private_key": _b64encode(self.private_key)})

    @classmethod
    def from_json(cls, raw):
        value = json.loads(raw)
        if set(value.keys()) != {"version", "node_id", "public_key", "private_key"} or value["version"] != 1:
            raise ValueError("unsupported or unknown identity state")
        return cls(value["node_id"], _b64decode(value["public_key"]), _b64decode(value["private_key"]))

    @classmethod
    def load_or_create(cls, path, node_id, crypto):
        try:
            with open(path, "r") as source:
                value = cls.from_json(source.read())
            if value.node_id != node_id:
                raise ValueError("persisted identity NodeID mismatch")
            return value
        except OSError:
            public_key, private_key = crypto.generate_key()
            value = cls(node_id, public_key, private_key)
            temporary = path + ".tmp"
            with open(temporary, "w") as destination:
                destination.write(value.to_json())
                if hasattr(destination, "flush"):
                    destination.flush()
            try:
                os.remove(path)
            except OSError:
                pass
            os.rename(temporary, path)
            return value
