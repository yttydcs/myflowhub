#include "myflowhub/mfh.h"

#include <stdio.h>
#include <string.h>

static const uint8_t json_content[] = "application/json";

static uint32_t read_u32(const uint8_t *p) { return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | p[3]; }
static uint16_t read_u16(const uint8_t *p) { return (uint16_t)(((uint16_t)p[0] << 8) | p[1]); }
static void write_u64(uint8_t *p, uint64_t v) { for (size_t i = 0; i < 8; ++i) p[i] = (uint8_t)(v >> (56u - (unsigned)i * 8u)); }

static int append_text(uint8_t *output, size_t capacity, size_t *offset, const char *text) {
    size_t length = strlen(text);
    if (*offset > capacity || length > capacity - *offset) return MFH_ERR_BUFFER;
    memcpy(output + *offset, text, length); *offset += length; return MFH_OK;
}

static int append_format(uint8_t *output, size_t capacity, size_t *offset, const char *format, unsigned long long value) {
    if (*offset >= capacity) return MFH_ERR_BUFFER;
    int count = snprintf((char *)output + *offset, capacity - *offset, format, value);
    if (count < 0 || (size_t)count >= capacity - *offset) return MFH_ERR_BUFFER;
    *offset += (size_t)count; return MFH_OK;
}

static int append_base64(uint8_t *output, size_t capacity, size_t *offset, const uint8_t *input, size_t length) {
    static const char alphabet[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    size_t required = ((length + 2u) / 3u) * 4u;
    if (*offset > capacity || required > capacity - *offset) return MFH_ERR_BUFFER;
    for (size_t i = 0; i < length; i += 3) {
        uint32_t value = (uint32_t)input[i] << 16;
        size_t remain = length - i;
        if (remain > 1) value |= (uint32_t)input[i + 1] << 8;
        if (remain > 2) value |= input[i + 2];
        output[(*offset)++] = (uint8_t)alphabet[(value >> 18) & 63u];
        output[(*offset)++] = (uint8_t)alphabet[(value >> 12) & 63u];
        output[(*offset)++] = remain > 1 ? (uint8_t)alphabet[(value >> 6) & 63u] : (uint8_t)'=';
        output[(*offset)++] = remain > 2 ? (uint8_t)alphabet[value & 63u] : (uint8_t)'=';
    }
    return MFH_OK;
}

static int next_id(mfh_client_t *client, uint8_t output[MFH_MESSAGE_ID_SIZE]) {
    if (client->random(client->crypto_context, output, MFH_MESSAGE_ID_SIZE) != 0) return MFH_ERR_CRYPTO;
    bool nonzero = false; for (size_t i = 0; i < MFH_MESSAGE_ID_SIZE; ++i) nonzero = nonzero || output[i] != 0;
    return nonzero ? MFH_OK : MFH_ERR_CRYPTO;
}

static int send_envelope(mfh_client_t *client, const mfh_envelope_t *envelope, uint8_t *buffer, size_t capacity) {
    size_t length = 0;
    int result = mfh_encode(envelope, buffer, capacity, &length, client->max_payload);
    if (result != MFH_OK) return result;
    return client->write(client->io_context, buffer, length) == 0 ? MFH_OK : MFH_ERR_IO;
}

int mfh_client_init(mfh_client_t *client, const mfh_identity_t *identity, uint64_t parent_id, uint64_t topology_epoch,
                    void *io_context, mfh_read_fn read_fn, mfh_write_fn write_fn,
                    void *crypto_context, mfh_random_fn random_fn, mfh_sign_fn sign_fn, mfh_verify_ack_fn verify_ack_fn) {
    if (client == NULL || identity == NULL || identity->version != 1 || identity->node_id == 0 || parent_id == 0 || topology_epoch == 0 ||
        read_fn == NULL || write_fn == NULL || random_fn == NULL || sign_fn == NULL || verify_ack_fn == NULL) return MFH_ERR_ARGUMENT;
    memset(client, 0, sizeof(*client));
    client->identity = *identity; client->parent_id = parent_id; client->topology_epoch = topology_epoch; client->max_payload = MFH_MAX_PAYLOAD;
    client->io_context = io_context; client->crypto_context = crypto_context; client->read = read_fn; client->write = write_fn;
    client->random = random_fn; client->sign = sign_fn; client->verify_ack = verify_ack_fn;
    return MFH_OK;
}

int mfh_client_build_join(mfh_client_t *client, const char *permit_json, uint8_t *payload, size_t capacity, size_t *written) {
    if (client == NULL || payload == NULL || written == NULL) return MFH_ERR_ARGUMENT;
    if (client->random(client->crypto_context, client->join_nonce, MFH_NONCE_SIZE) != 0) return MFH_ERR_CRYPTO;
    uint8_t message[9 + 8 + MFH_NONCE_SIZE + 8];
    memcpy(message, "MFH3-JOIN", 9); write_u64(message + 9, client->identity.node_id);
    memcpy(message + 17, client->join_nonce, MFH_NONCE_SIZE); write_u64(message + 49, client->topology_epoch);
    uint8_t signature[MFH_SIGNATURE_SIZE];
    if (client->sign(client->crypto_context, message, sizeof(message), signature) != 0) return MFH_ERR_CRYPTO;
    size_t offset = 0;
    int result = append_text(payload, capacity, &offset, "{\"node_id\":");
    if (result == MFH_OK) result = append_format(payload, capacity, &offset, "%llu", (unsigned long long)client->identity.node_id);
    if (result == MFH_OK) result = append_text(payload, capacity, &offset, ",\"public_key\":\"");
    if (result == MFH_OK) result = append_base64(payload, capacity, &offset, client->identity.public_key, MFH_PUBLIC_KEY_SIZE);
    if (result == MFH_OK) result = append_text(payload, capacity, &offset, "\",\"nonce\":[");
    for (size_t i = 0; result == MFH_OK && i < MFH_NONCE_SIZE; ++i) {
        if (i != 0) result = append_text(payload, capacity, &offset, ",");
        if (result == MFH_OK) result = append_format(payload, capacity, &offset, "%llu", client->join_nonce[i]);
    }
    if (result == MFH_OK) result = append_text(payload, capacity, &offset, "],\"topology_epoch\":");
    if (result == MFH_OK) result = append_format(payload, capacity, &offset, "%llu", (unsigned long long)client->topology_epoch);
    if (result == MFH_OK && permit_json != NULL && permit_json[0] != '\0') {
        result = append_text(payload, capacity, &offset, ",\"permit\":");
        if (result == MFH_OK) result = append_text(payload, capacity, &offset, permit_json);
    }
    if (result == MFH_OK) result = append_text(payload, capacity, &offset, ",\"signature\":\"");
    if (result == MFH_OK) result = append_base64(payload, capacity, &offset, signature, MFH_SIGNATURE_SIZE);
    if (result == MFH_OK) result = append_text(payload, capacity, &offset, "\"}");
    if (result != MFH_OK || offset > client->max_payload) return result == MFH_OK ? MFH_ERR_LIMIT : result;
    *written = offset;
    return MFH_OK;
}

int mfh_client_receive(mfh_client_t *client, mfh_envelope_t *envelope, uint8_t *frame_buffer, size_t frame_capacity) {
    if (client == NULL || envelope == NULL || frame_buffer == NULL || frame_capacity < MFH_HEADER_SIZE) return MFH_ERR_ARGUMENT;
    if (client->read(client->io_context, frame_buffer, MFH_HEADER_SIZE) != 0) return MFH_ERR_IO;
    if (memcmp(frame_buffer, "MFH3", 4) != 0) return MFH_ERR_WIRE;
    size_t body = frame_buffer[8] + frame_buffer[9] + read_u16(frame_buffer + 10) + read_u32(frame_buffer + 12);
    if (body > frame_capacity - MFH_HEADER_SIZE || read_u32(frame_buffer + 12) > client->max_payload) return MFH_ERR_LIMIT;
    if (client->read(client->io_context, frame_buffer + MFH_HEADER_SIZE, body) != 0) return MFH_ERR_IO;
    size_t consumed = 0;
    return mfh_decode(frame_buffer, MFH_HEADER_SIZE + body, envelope, &consumed, client->max_payload);
}

int mfh_client_join(mfh_client_t *client, const char *permit_json, uint8_t *frame_buffer, size_t frame_capacity) {
    if (client == NULL || frame_buffer == NULL) return MFH_ERR_ARGUMENT;
    uint8_t payload[2048]; size_t payload_len = 0;
    int result = mfh_client_build_join(client, permit_json, payload, sizeof(payload), &payload_len);
    if (result != MFH_OK) return result;
    mfh_envelope_t request; memset(&request, 0, sizeof(request));
    request.version = MFH_VERSION; request.phase = MFH_PHASE_REQUEST; request.operation = MFH_OP_JOIN;
    result = next_id(client, request.message_id);
    if (result != MFH_OK) return result;
    memcpy(client->next_message_id, request.message_id, MFH_MESSAGE_ID_SIZE);
    request.source = client->identity.node_id; request.target = client->parent_id;
    request.content_type = (mfh_slice_t){json_content, sizeof(json_content) - 1};
    static const uint8_t schema[] = "join.v1"; request.schema = (mfh_slice_t){schema, sizeof(schema) - 1};
    request.payload = (mfh_slice_t){payload, payload_len};
    result = send_envelope(client, &request, frame_buffer, frame_capacity);
    if (result != MFH_OK) return result;
    mfh_envelope_t response;
    result = mfh_client_receive(client, &response, frame_buffer, frame_capacity);
    if (result != MFH_OK) return result;
    if (response.operation != MFH_OP_JOIN_ACK || response.phase != MFH_PHASE_RESPONSE || response.source != client->parent_id || response.target != client->identity.node_id ||
        memcmp(response.correlation_id, request.message_id, MFH_MESSAGE_ID_SIZE) != 0) return MFH_ERR_REMOTE;
    if (client->verify_ack(client->crypto_context, response.payload.data, response.payload.len, client->parent_id, client->identity.node_id, client->join_nonce, client->topology_epoch) != 0) return MFH_ERR_CRYPTO;
    client->joined = true;
    return MFH_OK;
}

static int resource_envelope(mfh_client_t *client, mfh_envelope_t *envelope, uint8_t operation, uint64_t owner, const char *name, const uint8_t *payload, size_t payload_len) {
    if (!client->joined || owner == 0 || name == NULL || name[0] == '\0') return MFH_ERR_STATE;
    memset(envelope, 0, sizeof(*envelope));
    envelope->version = MFH_VERSION; envelope->phase = MFH_PHASE_REQUEST; envelope->operation = operation;
    int result = next_id(client, envelope->message_id);
    if (result != MFH_OK) return result;
    envelope->source = client->identity.node_id; envelope->target = owner; envelope->resource_owner = owner;
    envelope->resource_name = (mfh_slice_t){(const uint8_t *)name, strlen(name)};
    envelope->content_type = (mfh_slice_t){json_content, sizeof(json_content) - 1};
    envelope->payload = (mfh_slice_t){payload, payload_len};
    return MFH_OK;
}

int mfh_client_subscribe(mfh_client_t *client, uint64_t owner, const char *name, int64_t lease_ms, int queue, uint8_t message_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity) {
    if (lease_ms <= 0 || queue <= 0 || queue > 64 || message_id == NULL) return MFH_ERR_ARGUMENT;
    uint8_t payload[80]; int count = snprintf((char *)payload, sizeof(payload), "{\"lease_ms\":%lld,\"queue\":%d}", (long long)lease_ms, queue);
    if (count <= 0 || (size_t)count >= sizeof(payload)) return MFH_ERR_BUFFER;
    mfh_envelope_t envelope;
    int result = resource_envelope(client, &envelope, MFH_OP_SUBSCRIBE, owner, name, payload, (size_t)count);
    if (result != MFH_OK) return result;
    static const uint8_t schema[] = "subscribe.v1";
    envelope.schema = (mfh_slice_t){schema, sizeof(schema) - 1};
    memcpy(message_id, envelope.message_id, MFH_MESSAGE_ID_SIZE); return send_envelope(client, &envelope, frame_buffer, frame_capacity);
}

int mfh_client_unsubscribe(mfh_client_t *client, uint64_t owner, const char *name, const uint8_t subscription_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity) {
    if (subscription_id == NULL) return MFH_ERR_ARGUMENT;
    static const uint8_t payload[] = "{}";
    mfh_envelope_t envelope;
    int result = resource_envelope(client, &envelope, MFH_OP_UNSUBSCRIBE, owner, name, payload, sizeof(payload) - 1);
    if (result != MFH_OK) return result;
    memcpy(envelope.correlation_id, subscription_id, MFH_MESSAGE_ID_SIZE);
    static const uint8_t schema[] = "unsubscribe.v1";
    envelope.schema = (mfh_slice_t){schema, sizeof(schema) - 1};
    return send_envelope(client, &envelope, frame_buffer, frame_capacity);
}

int mfh_client_invoke(mfh_client_t *client, uint64_t owner, const char *name, const char *schema, const uint8_t *json, size_t json_len, int64_t deadline_unix_ms, uint8_t message_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity) {
    if (schema == NULL || json == NULL || json_len == 0 || deadline_unix_ms <= 0 || message_id == NULL) return MFH_ERR_ARGUMENT;
    mfh_envelope_t envelope;
    int result = resource_envelope(client, &envelope, MFH_OP_COMMAND_CALL, owner, name, json, json_len);
    if (result != MFH_OK) return result;
    envelope.schema = (mfh_slice_t){(const uint8_t *)schema, strlen(schema)};
    envelope.deadline_unix_ms = deadline_unix_ms;
    memcpy(message_id, envelope.message_id, MFH_MESSAGE_ID_SIZE); return send_envelope(client, &envelope, frame_buffer, frame_capacity);
}
