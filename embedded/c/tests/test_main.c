#include "myflowhub/mfh.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define CHECK(expression) do { if (!(expression)) { fprintf(stderr, "check failed at %s:%d: %s\n", __FILE__, __LINE__, #expression); return 1; } } while (0)

typedef struct {
    uint8_t input[MFH_MAX_FRAME];
    size_t input_len;
    size_t input_offset;
    uint8_t output[MFH_MAX_FRAME];
    size_t output_len;
} memory_io_t;

static int memory_read(void *context, uint8_t *output, size_t length) {
    memory_io_t *io = (memory_io_t *)context;
    if (io->input_offset > io->input_len || length > io->input_len - io->input_offset) return -1;
    memcpy(output, io->input + io->input_offset, length); io->input_offset += length; return 0;
}

static int memory_write(void *context, const uint8_t *input, size_t length) {
    memory_io_t *io = (memory_io_t *)context;
    if (length > sizeof(io->output)) return -1;
    memcpy(io->output, input, length); io->output_len = length; return 0;
}

static int fake_random(void *context, uint8_t *output, size_t length) {
    (void)context; for (size_t i = 0; i < length; ++i) output[i] = (uint8_t)(i + 1); return 0;
}

static int fake_sign(void *context, const uint8_t *message, size_t message_len, uint8_t signature[MFH_SIGNATURE_SIZE]) {
    (void)context; if (message_len != 57 || memcmp(message, "MFH3-JOIN", 9) != 0) return -1;
    memset(signature, 0x5a, MFH_SIGNATURE_SIZE); return 0;
}

static int fake_verify(void *context, const uint8_t *payload, size_t payload_len, uint64_t parent_id, uint64_t child_id, const uint8_t nonce[MFH_NONCE_SIZE], uint64_t epoch) {
    (void)context;
    return payload_len == 2 && memcmp(payload, "{}", 2) == 0 && parent_id == 1 && child_id == 2 && nonce[0] == 1 && epoch == 7 ? 0 : -1;
}

static bool slice_contains(const uint8_t *data, size_t length, const char *text) {
    size_t text_len = strlen(text);
    if (text_len > length) return false;
    for (size_t i = 0; i <= length - text_len; ++i) if (memcmp(data + i, text, text_len) == 0) return true;
    return false;
}

static mfh_envelope_t sample_envelope(void) {
    static const uint8_t content[] = "application/json";
    static const uint8_t schema[] = "mfh.test.v1";
    static const uint8_t name[] = "device/led/set";
    static const uint8_t payload[] = "{\"version\":1,\"value\":\"true\"}";
    mfh_envelope_t envelope; memset(&envelope, 0, sizeof(envelope));
    envelope.version = 1; envelope.phase = MFH_PHASE_REQUEST; envelope.operation = MFH_OP_COMMAND_CALL;
    for (size_t i = 0; i < MFH_MESSAGE_ID_SIZE; ++i) envelope.message_id[i] = (uint8_t)i;
    envelope.message_id[0] = 0x10; envelope.source = 2; envelope.target = 1; envelope.resource_owner = 1; envelope.deadline_unix_ms = 1000;
    envelope.content_type = (mfh_slice_t){content, sizeof(content) - 1}; envelope.schema = (mfh_slice_t){schema, sizeof(schema) - 1};
    envelope.resource_name = (mfh_slice_t){name, sizeof(name) - 1}; envelope.payload = (mfh_slice_t){payload, sizeof(payload) - 1};
    return envelope;
}

static int hex_digit(int value) {
    if (value >= '0' && value <= '9') return value - '0';
    if (value >= 'a' && value <= 'f') return value - 'a' + 10;
    if (value >= 'A' && value <= 'F') return value - 'A' + 10;
    return -1;
}

static int load_frame_fixture(uint8_t *output, size_t capacity, size_t *length) {
    char path[1024];
    int path_len = snprintf(path, sizeof(path), "%s/embedded/envelope-command-v1.hex", MFH_FIXTURE_ROOT);
    if (path_len < 0 || (size_t)path_len >= sizeof(path)) return -1;
    FILE *source = fopen(path, "rb");
    if (source == NULL) return -1;
    size_t written = 0;
    int high = -1;
    int value;
    while ((value = fgetc(source)) != EOF) {
        if (value == '\r' || value == '\n' || value == ' ' || value == '\t') continue;
        int digit = hex_digit(value);
        if (digit < 0) { fclose(source); return -1; }
        if (high < 0) {
            high = digit;
        } else {
            if (written >= capacity) { fclose(source); return -1; }
            output[written++] = (uint8_t)((high << 4) | digit);
            high = -1;
        }
    }
    if (fclose(source) != 0 || high >= 0) return -1;
    *length = written;
    return 0;
}

static int test_codec(void) {
    uint8_t frame[512], fixture[512]; size_t written = 0, fixture_len = 0, consumed = 0;
    mfh_envelope_t sample = sample_envelope();
    CHECK(mfh_encode(&sample, frame, sizeof(frame), &written, MFH_MAX_PAYLOAD) == MFH_OK);
    CHECK(load_frame_fixture(fixture, sizeof(fixture), &fixture_len) == 0);
    CHECK(written == fixture_len && memcmp(frame, fixture, written) == 0);
    CHECK(written == MFH_HEADER_SIZE + 16 + 11 + 14 + 28);
    CHECK(memcmp(frame, "MFH3\x00\x01\x01\x0c\x10\x0b\x00\x0e\x00\x00\x00\x1c", 16) == 0);
    CHECK(frame[23] == 2 && frame[39] == 1 && frame[55] == 0xe8 && frame[95] == 1);
    mfh_envelope_t decoded;
    CHECK(mfh_decode(frame, written, &decoded, &consumed, MFH_MAX_PAYLOAD) == MFH_OK);
    CHECK(consumed == written && decoded.operation == MFH_OP_COMMAND_CALL && decoded.source == 2 && decoded.target == 1);
    CHECK(decoded.resource_name.len == 14 && memcmp(decoded.resource_name.data, "device/led/set", 14) == 0);
    CHECK(decoded.payload.len == 28 && memcmp(decoded.payload.data, "{\"version\":1,\"value\":\"true\"}", 28) == 0);
    CHECK(mfh_decode(frame, MFH_HEADER_SIZE - 1, &decoded, &consumed, MFH_MAX_PAYLOAD) == MFH_ERR_WIRE);
    frame[12] = 0x7f; CHECK(mfh_decode(frame, written, &decoded, &consumed, MFH_MAX_PAYLOAD) == MFH_ERR_LIMIT);
    return 0;
}

static int test_identity(void) {
    mfh_identity_t identity; memset(&identity, 0, sizeof(identity)); identity.version = 1; identity.node_id = 42;
    for (size_t i = 0; i < MFH_PUBLIC_KEY_SIZE; ++i) identity.public_key[i] = (uint8_t)i;
    for (size_t i = 0; i < MFH_PRIVATE_KEY_SIZE; ++i) identity.private_key[i] = (uint8_t)(255u - i);
    uint8_t data[128]; size_t written = 0; CHECK(mfh_identity_encode(&identity, data, sizeof(data), &written) == MFH_OK); CHECK(written == 112);
    mfh_identity_t decoded; CHECK(mfh_identity_decode(data, written, &decoded) == MFH_OK);
    CHECK(decoded.version == identity.version && decoded.node_id == identity.node_id);
    CHECK(memcmp(identity.public_key, decoded.public_key, MFH_PUBLIC_KEY_SIZE) == 0);
    CHECK(memcmp(identity.private_key, decoded.private_key, MFH_PRIVATE_KEY_SIZE) == 0);
    data[0] = 'X'; CHECK(mfh_identity_decode(data, written, &decoded) == MFH_ERR_ARGUMENT);
    return 0;
}

static int test_client_join_and_operations(void) {
    mfh_identity_t identity; memset(&identity, 0, sizeof(identity)); identity.version = 1; identity.node_id = 2; memset(identity.public_key, 0x11, sizeof(identity.public_key));
    memory_io_t io; memset(&io, 0, sizeof(io)); mfh_client_t client;
    CHECK(mfh_client_init(&client, &identity, 1, 7, &io, memory_read, memory_write, NULL, fake_random, fake_sign, fake_verify) == MFH_OK);
    mfh_envelope_t ack; memset(&ack, 0, sizeof(ack)); ack.version = 1; ack.phase = MFH_PHASE_RESPONSE; ack.operation = MFH_OP_JOIN_ACK;
    for (size_t i = 0; i < MFH_MESSAGE_ID_SIZE; ++i) { ack.message_id[i] = (uint8_t)(0x80u + i); ack.correlation_id[i] = (uint8_t)(i + 1); }
    ack.source = 1; ack.target = 2; static const uint8_t content[] = "application/json", schema[] = "join-ack.v1", payload[] = "{}";
    ack.content_type = (mfh_slice_t){content, sizeof(content) - 1}; ack.schema = (mfh_slice_t){schema, sizeof(schema) - 1}; ack.payload = (mfh_slice_t){payload, sizeof(payload) - 1};
    CHECK(mfh_encode(&ack, io.input, sizeof(io.input), &io.input_len, MFH_MAX_PAYLOAD) == MFH_OK);
    uint8_t frame[MFH_MAX_FRAME]; CHECK(mfh_client_join(&client, NULL, frame, sizeof(frame)) == MFH_OK); CHECK(client.joined && io.output_len > MFH_HEADER_SIZE);
    mfh_envelope_t sent; size_t consumed = 0; CHECK(mfh_decode(io.output, io.output_len, &sent, &consumed, MFH_MAX_PAYLOAD) == MFH_OK);
    CHECK(sent.operation == MFH_OP_JOIN && sent.payload.len > 100 && slice_contains(sent.payload.data, sent.payload.len, "\"node_id\":2"));
    uint8_t id[16]; CHECK(mfh_client_subscribe(&client, 1, "system/health", 60000, 4, id, frame, sizeof(frame)) == MFH_OK);
    CHECK(mfh_decode(io.output, io.output_len, &sent, &consumed, MFH_MAX_PAYLOAD) == MFH_OK && sent.operation == MFH_OP_SUBSCRIBE);
    static const uint8_t command[] = "{\"version\":1}";
    CHECK(mfh_client_invoke(&client, 1, "device/led/set", "mfh.device.led.v1", command, sizeof(command) - 1, 1000, id, frame, sizeof(frame)) == MFH_OK);
    CHECK(mfh_decode(io.output, io.output_len, &sent, &consumed, MFH_MAX_PAYLOAD) == MFH_OK && sent.operation == MFH_OP_COMMAND_CALL);
    return 0;
}

int main(void) {
    if (test_codec() != 0 || test_identity() != 0 || test_client_join_and_operations() != 0) return EXIT_FAILURE;
    puts("mfh embedded C tests passed"); return EXIT_SUCCESS;
}
