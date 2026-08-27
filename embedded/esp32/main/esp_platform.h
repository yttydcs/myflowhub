#ifndef MYFLOWHUB_ESP_PLATFORM_H
#define MYFLOWHUB_ESP_PLATFORM_H

#include "myflowhub/mfh.h"

#include <stddef.h>
#include <stdint.h>

typedef struct {
    int socket_fd;
    mfh_identity_t identity;
    uint8_t parent_public_key[MFH_PUBLIC_KEY_SIZE];
} mfh_esp_platform_t;

int mfh_esp_platform_init(mfh_esp_platform_t *platform, mfh_identity_t *identity);
int mfh_esp_connect_wifi(void);
int mfh_esp_connect_tcp(mfh_esp_platform_t *platform, const char *host, uint16_t port);
void mfh_esp_close(mfh_esp_platform_t *platform);
void mfh_esp_log_identity(const mfh_identity_t *identity);

int mfh_esp_read(void *context, uint8_t *output, size_t length);
int mfh_esp_write(void *context, const uint8_t *input, size_t length);
int mfh_esp_random(void *context, uint8_t *output, size_t length);
int mfh_esp_sign(void *context, const uint8_t *message, size_t message_len, uint8_t signature[MFH_SIGNATURE_SIZE]);
int mfh_esp_verify_ack(void *context, const uint8_t *payload, size_t payload_len, uint64_t parent_id, uint64_t child_id,
                       const uint8_t nonce[MFH_NONCE_SIZE], uint64_t epoch);

#endif
