#include "myflowhub/mfh.h"

#include <string.h>

static const uint8_t mfh_magic[4] = {'M', 'F', 'H', '3'};

static void put_u16(uint8_t *p, uint16_t v) { p[0] = (uint8_t)(v >> 8); p[1] = (uint8_t)v; }
static void put_u32(uint8_t *p, uint32_t v) { p[0] = (uint8_t)(v >> 24); p[1] = (uint8_t)(v >> 16); p[2] = (uint8_t)(v >> 8); p[3] = (uint8_t)v; }
static void put_u64(uint8_t *p, uint64_t v) { for (size_t i = 0; i < 8; ++i) p[i] = (uint8_t)(v >> (56u - (unsigned)i * 8u)); }
static uint16_t get_u16(const uint8_t *p) { return (uint16_t)(((uint16_t)p[0] << 8) | p[1]); }
static uint32_t get_u32(const uint8_t *p) { return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | p[3]; }
static uint64_t get_u64(const uint8_t *p) { uint64_t v = 0; for (size_t i = 0; i < 8; ++i) v = (v << 8) | p[i]; return v; }
static bool nonzero(const uint8_t *p, size_t n) { for (size_t i = 0; i < n; ++i) if (p[i] != 0) return true; return false; }

static bool phase_allows(uint8_t phase, uint8_t op) {
    if (op == MFH_OP_JOIN) return phase == MFH_PHASE_REQUEST;
    if (op == MFH_OP_JOIN_ACK || op == MFH_OP_SUBSCRIBE_ACK || op == MFH_OP_VARIABLE_SNAPSHOT || op == MFH_OP_COMMAND_RESULT || op == MFH_OP_ERROR) return phase == MFH_PHASE_RESPONSE;
    if (op == MFH_OP_ROUTE_ANNOUNCE || op == MFH_OP_ROUTE_WITHDRAW || op == MFH_OP_VARIABLE_UPDATE || op == MFH_OP_STREAM_EVENT || op == MFH_OP_STREAM_GAP || op == MFH_OP_HEARTBEAT) return phase == MFH_PHASE_EVENT;
    if (op == MFH_OP_SUBSCRIBE || op == MFH_OP_UNSUBSCRIBE || op == MFH_OP_COMMAND_CALL) return phase == MFH_PHASE_REQUEST || phase == MFH_PHASE_CONTROL;
    return false;
}

static bool requires_resource(uint8_t op) { return op >= MFH_OP_SUBSCRIBE && op <= MFH_OP_COMMAND_RESULT; }
static bool requires_correlation(uint8_t op) { return op == MFH_OP_JOIN_ACK || op == MFH_OP_SUBSCRIBE_ACK || op == MFH_OP_VARIABLE_SNAPSHOT || op == MFH_OP_COMMAND_RESULT || op == MFH_OP_ERROR; }

int mfh_envelope_validate(const mfh_envelope_t *e, size_t max_payload) {
    if (e == NULL) return MFH_ERR_ARGUMENT;
    if (e->version != MFH_VERSION) return MFH_ERR_VERSION;
    if (e->phase < MFH_PHASE_REQUEST || e->phase > MFH_PHASE_EVENT || !phase_allows(e->phase, e->operation)) return MFH_ERR_PHASE;
    if (e->operation < MFH_OP_JOIN || e->operation > MFH_OP_HEARTBEAT) return MFH_ERR_OPERATION;
    if (!nonzero(e->message_id, MFH_MESSAGE_ID_SIZE) || e->source == 0 || e->target == 0) return MFH_ERR_ID;
    if (requires_correlation(e->operation) && !nonzero(e->correlation_id, MFH_MESSAGE_ID_SIZE)) return MFH_ERR_ID;
    if (requires_resource(e->operation)) {
        if (e->resource_owner == 0 || e->resource_name.data == NULL || e->resource_name.len == 0 || e->resource_name.len > MFH_MAX_RESOURCE_NAME) return MFH_ERR_RESOURCE;
    } else if (e->resource_owner != 0 || e->resource_name.len != 0) return MFH_ERR_RESOURCE;
    if (e->content_type.len > MFH_MAX_CONTENT_TYPE || e->schema.len > MFH_MAX_SCHEMA) return MFH_ERR_LIMIT;
    if (max_payload == 0) max_payload = MFH_MAX_PAYLOAD;
    if (e->payload.len > max_payload || (e->payload.len != 0 && e->payload.data == NULL)) return MFH_ERR_LIMIT;
    if (e->deadline_unix_ms < 0) return MFH_ERR_ARGUMENT;
    return MFH_OK;
}

int mfh_encode(const mfh_envelope_t *e, uint8_t *output, size_t capacity, size_t *written, size_t max_payload) {
    int result = mfh_envelope_validate(e, max_payload);
    if (result != MFH_OK || output == NULL || written == NULL) return result == MFH_OK ? MFH_ERR_ARGUMENT : result;
    size_t total = MFH_HEADER_SIZE + e->content_type.len + e->schema.len + e->resource_name.len + e->payload.len;
    if (capacity < total || e->payload.len > UINT32_MAX) return MFH_ERR_BUFFER;
    memset(output, 0, MFH_HEADER_SIZE);
    memcpy(output, mfh_magic, 4);
    put_u16(output + 4, e->version); output[6] = e->phase; output[7] = e->operation;
    output[8] = (uint8_t)e->content_type.len; output[9] = (uint8_t)e->schema.len;
    put_u16(output + 10, (uint16_t)e->resource_name.len); put_u32(output + 12, (uint32_t)e->payload.len);
    put_u64(output + 16, e->source); put_u64(output + 24, e->principal); put_u64(output + 32, e->target);
    put_u64(output + 40, e->topology_epoch); put_u64(output + 48, (uint64_t)e->deadline_unix_ms);
    memcpy(output + 56, e->message_id, MFH_MESSAGE_ID_SIZE); memcpy(output + 72, e->correlation_id, MFH_MESSAGE_ID_SIZE);
    put_u64(output + 88, e->resource_owner);
    size_t offset = MFH_HEADER_SIZE;
    const mfh_slice_t parts[] = {e->content_type, e->schema, e->resource_name, e->payload};
    for (size_t i = 0; i < sizeof(parts) / sizeof(parts[0]); ++i) { if (parts[i].len != 0) memcpy(output + offset, parts[i].data, parts[i].len); offset += parts[i].len; }
    *written = total;
    return MFH_OK;
}

int mfh_decode(const uint8_t *input, size_t input_len, mfh_envelope_t *e, size_t *consumed, size_t max_payload) {
    if (input == NULL || e == NULL || consumed == NULL) return MFH_ERR_ARGUMENT;
    if (input_len < MFH_HEADER_SIZE || memcmp(input, mfh_magic, 4) != 0) return MFH_ERR_WIRE;
    size_t content_len = input[8], schema_len = input[9], name_len = get_u16(input + 10), payload_len = get_u32(input + 12);
    if (content_len > MFH_MAX_CONTENT_TYPE || schema_len > MFH_MAX_SCHEMA || name_len > MFH_MAX_RESOURCE_NAME) return MFH_ERR_LIMIT;
    if (max_payload == 0) max_payload = MFH_MAX_PAYLOAD;
    if (payload_len > max_payload) return MFH_ERR_LIMIT;
    size_t total = MFH_HEADER_SIZE + content_len + schema_len + name_len + payload_len;
    if (total < MFH_HEADER_SIZE || input_len < total) return MFH_ERR_BUFFER;
    memset(e, 0, sizeof(*e));
    e->version = get_u16(input + 4); e->phase = input[6]; e->operation = input[7];
    e->source = get_u64(input + 16); e->principal = get_u64(input + 24); e->target = get_u64(input + 32);
    e->topology_epoch = get_u64(input + 40); e->deadline_unix_ms = (int64_t)get_u64(input + 48); e->resource_owner = get_u64(input + 88);
    memcpy(e->message_id, input + 56, MFH_MESSAGE_ID_SIZE); memcpy(e->correlation_id, input + 72, MFH_MESSAGE_ID_SIZE);
    size_t offset = MFH_HEADER_SIZE;
    e->content_type.data = input + offset; e->content_type.len = content_len; offset += content_len;
    e->schema.data = input + offset; e->schema.len = schema_len; offset += schema_len;
    e->resource_name.data = input + offset; e->resource_name.len = name_len; offset += name_len;
    e->payload.data = input + offset; e->payload.len = payload_len;
    int result = mfh_envelope_validate(e, max_payload);
    if (result != MFH_OK) return result;
    *consumed = total;
    return MFH_OK;
}

int mfh_identity_encode(const mfh_identity_t *identity, uint8_t *output, size_t capacity, size_t *written) {
    const size_t size = 4 + 4 + 8 + MFH_PUBLIC_KEY_SIZE + MFH_PRIVATE_KEY_SIZE;
    if (identity == NULL || output == NULL || written == NULL || identity->version != 1 || identity->node_id == 0) return MFH_ERR_ARGUMENT;
    if (capacity < size) return MFH_ERR_BUFFER;
    memcpy(output, "MFHI", 4); put_u32(output + 4, identity->version); put_u64(output + 8, identity->node_id);
    memcpy(output + 16, identity->public_key, MFH_PUBLIC_KEY_SIZE); memcpy(output + 48, identity->private_key, MFH_PRIVATE_KEY_SIZE);
    *written = size;
    return MFH_OK;
}

int mfh_identity_decode(const uint8_t *input, size_t input_len, mfh_identity_t *identity) {
    const size_t size = 4 + 4 + 8 + MFH_PUBLIC_KEY_SIZE + MFH_PRIVATE_KEY_SIZE;
    if (input == NULL || identity == NULL || input_len != size || memcmp(input, "MFHI", 4) != 0) return MFH_ERR_ARGUMENT;
    identity->version = get_u32(input + 4); identity->node_id = get_u64(input + 8);
    if (identity->version != 1 || identity->node_id == 0) return MFH_ERR_VERSION;
    memcpy(identity->public_key, input + 16, MFH_PUBLIC_KEY_SIZE); memcpy(identity->private_key, input + 48, MFH_PRIVATE_KEY_SIZE);
    return MFH_OK;
}
