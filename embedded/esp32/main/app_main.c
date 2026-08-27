#include "esp_platform.h"

#include "esp_log.h"
#include "myflowhub/mfh.h"

#include <stdint.h>
#include <string.h>

static const char *TAG = "mfh-demo";
static uint8_t frame[MFH_MAX_FRAME];

void app_main(void) {
    mfh_esp_platform_t platform;
    mfh_identity_t identity;
    if (mfh_esp_platform_init(&platform, &identity) != 0) { ESP_LOGE(TAG, "platform/identity initialization failed"); return; }
    mfh_esp_log_identity(&identity);
    if (mfh_esp_connect_wifi() != 0) { ESP_LOGE(TAG, "Wi-Fi connection failed"); return; }
    if (mfh_esp_connect_tcp(&platform, CONFIG_MFH_PARENT_HOST, CONFIG_MFH_PARENT_PORT) != 0) { ESP_LOGE(TAG, "parent TCP connection failed"); return; }

    mfh_client_t client;
    if (mfh_client_init(&client, &identity, CONFIG_MFH_PARENT_NODE_ID, 1, &platform, mfh_esp_read, mfh_esp_write,
                        &platform, mfh_esp_random, mfh_esp_sign, mfh_esp_verify_ack) != MFH_OK) {
        ESP_LOGE(TAG, "client initialization failed"); mfh_esp_close(&platform); return;
    }
    const char *permit = CONFIG_MFH_PERMIT_JSON[0] == '\0' ? NULL : CONFIG_MFH_PERMIT_JSON;
    if (mfh_client_join(&client, permit, frame, sizeof(frame)) != MFH_OK) { ESP_LOGE(TAG, "signed join failed"); mfh_esp_close(&platform); return; }

    uint8_t subscription_id[MFH_MESSAGE_ID_SIZE];
    if (mfh_client_subscribe(&client, CONFIG_MFH_PARENT_NODE_ID, "system/health", 60000, 8, subscription_id, frame, sizeof(frame)) != MFH_OK) {
        ESP_LOGE(TAG, "health subscription failed"); mfh_esp_close(&platform); return;
    }
    static const uint8_t request[] = "{\"version\":1,\"channel\":\"embedded\",\"content_type\":\"text/plain\",\"body\":\"aGVsbG8=\"}";
    uint8_t command_id[MFH_MESSAGE_ID_SIZE];
    if (mfh_client_invoke(&client, CONFIG_MFH_PARENT_NODE_ID, "notifications/publish", "mfh.notification.publish.v1",
                          request, sizeof(request) - 1, INT64_MAX, command_id, frame, sizeof(frame)) != MFH_OK) {
        ESP_LOGE(TAG, "notification command failed to send"); mfh_esp_close(&platform); return;
    }
    ESP_LOGI(TAG, "joined; health subscription and notification Command are active");
    for (;;) {
        mfh_envelope_t envelope;
        int result = mfh_client_receive(&client, &envelope, frame, sizeof(frame));
        if (result != MFH_OK) { ESP_LOGE(TAG, "receive failed: %d", result); break; }
        ESP_LOGI(TAG, "operation=%u payload_bytes=%u", envelope.operation, (unsigned)envelope.payload.len);
    }
    mfh_esp_close(&platform);
}
