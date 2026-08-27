#ifndef MYFLOWHUB_MFH_H
#define MYFLOWHUB_MFH_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define MFH_VERSION 1u
#define MFH_HEADER_SIZE 96u
#define MFH_MESSAGE_ID_SIZE 16u
#define MFH_PUBLIC_KEY_SIZE 32u
#define MFH_PRIVATE_KEY_SIZE 64u
#define MFH_SIGNATURE_SIZE 64u
#define MFH_NONCE_SIZE 32u
#define MFH_MAX_RESOURCE_NAME 255u
#define MFH_MAX_CONTENT_TYPE 127u
#define MFH_MAX_SCHEMA 255u
#ifndef MFH_MAX_PAYLOAD
#define MFH_MAX_PAYLOAD 8192u
#endif
#define MFH_MAX_FRAME (MFH_HEADER_SIZE + MFH_MAX_CONTENT_TYPE + MFH_MAX_SCHEMA + MFH_MAX_RESOURCE_NAME + MFH_MAX_PAYLOAD)

typedef enum {
    MFH_OK = 0,
    MFH_ERR_ARGUMENT = -1,
    MFH_ERR_VERSION = -2,
    MFH_ERR_PHASE = -3,
    MFH_ERR_OPERATION = -4,
    MFH_ERR_ID = -5,
    MFH_ERR_RESOURCE = -6,
    MFH_ERR_LIMIT = -7,
    MFH_ERR_BUFFER = -8,
    MFH_ERR_WIRE = -9,
    MFH_ERR_IO = -10,
    MFH_ERR_CRYPTO = -11,
    MFH_ERR_STATE = -12,
    MFH_ERR_REMOTE = -13
} mfh_result_t;

typedef enum {
    MFH_PHASE_REQUEST = 1,
    MFH_PHASE_CONTROL = 2,
    MFH_PHASE_RESPONSE = 3,
    MFH_PHASE_EVENT = 4
} mfh_phase_t;

typedef enum {
    MFH_OP_JOIN = 1,
    MFH_OP_JOIN_ACK = 2,
    MFH_OP_ROUTE_ANNOUNCE = 3,
    MFH_OP_ROUTE_WITHDRAW = 4,
    MFH_OP_SUBSCRIBE = 5,
    MFH_OP_UNSUBSCRIBE = 6,
    MFH_OP_SUBSCRIBE_ACK = 7,
    MFH_OP_VARIABLE_SNAPSHOT = 8,
    MFH_OP_VARIABLE_UPDATE = 9,
    MFH_OP_STREAM_EVENT = 10,
    MFH_OP_STREAM_GAP = 11,
    MFH_OP_COMMAND_CALL = 12,
    MFH_OP_COMMAND_RESULT = 13,
    MFH_OP_ERROR = 14,
    MFH_OP_HEARTBEAT = 15
} mfh_operation_t;

typedef struct {
    const uint8_t *data;
    size_t len;
} mfh_slice_t;

typedef struct {
    uint16_t version;
    uint8_t phase;
    uint8_t operation;
    uint8_t message_id[MFH_MESSAGE_ID_SIZE];
    uint8_t correlation_id[MFH_MESSAGE_ID_SIZE];
    uint64_t source;
    uint64_t principal;
    uint64_t target;
    uint64_t resource_owner;
    uint64_t topology_epoch;
    int64_t deadline_unix_ms;
    mfh_slice_t content_type;
    mfh_slice_t schema;
    mfh_slice_t resource_name;
    mfh_slice_t payload;
} mfh_envelope_t;

typedef struct {
    uint32_t version;
    uint64_t node_id;
    uint8_t public_key[MFH_PUBLIC_KEY_SIZE];
    uint8_t private_key[MFH_PRIVATE_KEY_SIZE];
} mfh_identity_t;

typedef int (*mfh_read_fn)(void *context, uint8_t *output, size_t length);
typedef int (*mfh_write_fn)(void *context, const uint8_t *input, size_t length);
typedef int (*mfh_random_fn)(void *context, uint8_t *output, size_t length);
typedef int (*mfh_sign_fn)(void *context, const uint8_t *message, size_t message_len, uint8_t signature[MFH_SIGNATURE_SIZE]);
typedef int (*mfh_verify_ack_fn)(void *context, const uint8_t *payload, size_t payload_len, uint64_t parent_id, uint64_t child_id, const uint8_t nonce[MFH_NONCE_SIZE], uint64_t epoch);

typedef struct {
    mfh_identity_t identity;
    uint64_t parent_id;
    uint64_t topology_epoch;
    size_t max_payload;
    void *io_context;
    void *crypto_context;
    mfh_read_fn read;
    mfh_write_fn write;
    mfh_random_fn random;
    mfh_sign_fn sign;
    mfh_verify_ack_fn verify_ack;
    uint8_t next_message_id[MFH_MESSAGE_ID_SIZE];
    uint8_t join_nonce[MFH_NONCE_SIZE];
    bool joined;
} mfh_client_t;

int mfh_envelope_validate(const mfh_envelope_t *envelope, size_t max_payload);
int mfh_encode(const mfh_envelope_t *envelope, uint8_t *output, size_t capacity, size_t *written, size_t max_payload);
int mfh_decode(const uint8_t *input, size_t input_len, mfh_envelope_t *envelope, size_t *consumed, size_t max_payload);

int mfh_identity_encode(const mfh_identity_t *identity, uint8_t *output, size_t capacity, size_t *written);
int mfh_identity_decode(const uint8_t *input, size_t input_len, mfh_identity_t *identity);

int mfh_client_init(mfh_client_t *client, const mfh_identity_t *identity, uint64_t parent_id, uint64_t topology_epoch,
                    void *io_context, mfh_read_fn read_fn, mfh_write_fn write_fn,
                    void *crypto_context, mfh_random_fn random_fn, mfh_sign_fn sign_fn, mfh_verify_ack_fn verify_ack_fn);
int mfh_client_build_join(mfh_client_t *client, const char *permit_json, uint8_t *payload, size_t capacity, size_t *written);
int mfh_client_join(mfh_client_t *client, const char *permit_json, uint8_t *frame_buffer, size_t frame_capacity);
int mfh_client_subscribe(mfh_client_t *client, uint64_t owner, const char *name, int64_t lease_ms, int queue, uint8_t message_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity);
int mfh_client_unsubscribe(mfh_client_t *client, uint64_t owner, const char *name, const uint8_t subscription_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity);
int mfh_client_invoke(mfh_client_t *client, uint64_t owner, const char *name, const char *schema, const uint8_t *json, size_t json_len, int64_t deadline_unix_ms, uint8_t message_id[MFH_MESSAGE_ID_SIZE], uint8_t *frame_buffer, size_t frame_capacity);
int mfh_client_receive(mfh_client_t *client, mfh_envelope_t *envelope, uint8_t *frame_buffer, size_t frame_capacity);

#ifdef __cplusplus
}
#endif
#endif
