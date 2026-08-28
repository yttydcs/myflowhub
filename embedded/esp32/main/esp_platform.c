#include "esp_platform.h"

#include "cJSON.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_random.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "mbedtls/base64.h"
#include "nvs.h"
#include "nvs_flash.h"
#include "sodium.h"

#include <arpa/inet.h>
#include <errno.h>
#include <netdb.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

#define MFH_STATE_NAMESPACE "myflowhub"
#define MFH_IDENTITY_KEY "identity"
#define MFH_CONFIG_KEY "config"
#define MFH_CONFIG_VERSION 1u
#define WIFI_CONNECTED_BIT BIT0
#define WIFI_FAILED_BIT BIT1

static const char *TAG = "mfh-platform";
static EventGroupHandle_t wifi_events;
static int wifi_attempts;

typedef struct {
    uint32_t version;
    uint32_t revision;
    uint64_t node_id;
    uint64_t parent_id;
    uint16_t port;
    char host[64];
    uint8_t parent_public_key[MFH_PUBLIC_KEY_SIZE];
} persistent_config_t;

static void write_u64(uint8_t *output, uint64_t value) {
    for (size_t i = 0; i < 8; ++i) output[i] = (uint8_t)(value >> (56u - (unsigned)i * 8u));
}

static int decode_base64_exact(const char *encoded, uint8_t *output, size_t expected) {
    size_t written = 0;
    if (encoded == NULL || encoded[0] == '\0' || mbedtls_base64_decode(output, expected, &written, (const unsigned char *)encoded, strlen(encoded)) != 0) return -1;
    return written == expected ? 0 : -1;
}

static int init_nvs(void) {
    esp_err_t result = nvs_flash_init();
    if (result == ESP_ERR_NVS_NO_FREE_PAGES || result == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        result = nvs_flash_init();
    }
    return result == ESP_OK ? 0 : -1;
}

static int load_config(nvs_handle_t handle, persistent_config_t *config) {
    persistent_config_t compiled = {
        .version = MFH_CONFIG_VERSION,
        .revision = CONFIG_MFH_CONFIG_REVISION,
        .node_id = CONFIG_MFH_NODE_ID,
        .parent_id = CONFIG_MFH_PARENT_NODE_ID,
        .port = CONFIG_MFH_PARENT_PORT,
    };
    if (strlen(CONFIG_MFH_PARENT_HOST) >= sizeof(compiled.host) ||
        decode_base64_exact(CONFIG_MFH_PARENT_PUBLIC_KEY, compiled.parent_public_key, sizeof(compiled.parent_public_key)) != 0) {
        ESP_LOGE(TAG, "parent host or Ed25519 public key configuration is invalid");
        return -1;
    }
    memcpy(compiled.host, CONFIG_MFH_PARENT_HOST, strlen(CONFIG_MFH_PARENT_HOST) + 1);
    size_t length = sizeof(*config);
    esp_err_t result = nvs_get_blob(handle, MFH_CONFIG_KEY, config, &length);
    if (result == ESP_ERR_NVS_NOT_FOUND) {
        *config = compiled;
        if (nvs_set_blob(handle, MFH_CONFIG_KEY, config, sizeof(*config)) != ESP_OK || nvs_commit(handle) != ESP_OK) return -1;
        return 0;
    }
    if (result != ESP_OK || length != sizeof(*config) || config->version != MFH_CONFIG_VERSION ||
        config->revision != CONFIG_MFH_CONFIG_REVISION || config->node_id != CONFIG_MFH_NODE_ID ||
        config->parent_id != CONFIG_MFH_PARENT_NODE_ID || config->port != CONFIG_MFH_PARENT_PORT ||
        memchr(config->host, '\0', sizeof(config->host)) == NULL ||
        strcmp(config->host, CONFIG_MFH_PARENT_HOST) != 0 || memcmp(config->parent_public_key, compiled.parent_public_key, MFH_PUBLIC_KEY_SIZE) != 0) {
        ESP_LOGE(TAG, "persistent config mismatch; increment revision and enable explicit reset once");
        return -1;
    }
    return 0;
}

static int make_identity(mfh_identity_t *identity) {
    if (crypto_sign_keypair(identity->public_key, identity->private_key) != 0) return -1;
    identity->version = 1;
    identity->node_id = CONFIG_MFH_NODE_ID;
    return 0;
}

static int load_identity(nvs_handle_t handle, mfh_identity_t *identity) {
    uint8_t encoded[112];
    size_t length = sizeof(encoded);
    esp_err_t result = nvs_get_blob(handle, MFH_IDENTITY_KEY, encoded, &length);
    if (result == ESP_ERR_NVS_NOT_FOUND) {
        if (make_identity(identity) != 0 || mfh_identity_encode(identity, encoded, sizeof(encoded), &length) != MFH_OK) return -1;
        return nvs_set_blob(handle, MFH_IDENTITY_KEY, encoded, length) == ESP_OK && nvs_commit(handle) == ESP_OK ? 0 : -1;
    }
    if (result != ESP_OK || mfh_identity_decode(encoded, length, identity) != MFH_OK || identity->node_id != CONFIG_MFH_NODE_ID ||
        memcmp(identity->private_key + 32, identity->public_key, MFH_PUBLIC_KEY_SIZE) != 0) {
        ESP_LOGE(TAG, "stored identity is invalid; refusing silent replacement");
        return -1;
    }
    return 0;
}

int mfh_esp_platform_init(mfh_esp_platform_t *platform, mfh_identity_t *identity) {
    if (platform == NULL || identity == NULL || init_nvs() != 0 || sodium_init() < 0) return -1;
    memset(platform, 0, sizeof(*platform));
    platform->socket_fd = -1;
    nvs_handle_t handle;
    if (nvs_open(MFH_STATE_NAMESPACE, NVS_READWRITE, &handle) != ESP_OK) return -1;
    if (CONFIG_MFH_RESET_STATE_ON_BOOT) {
        if (nvs_erase_all(handle) != ESP_OK || nvs_commit(handle) != ESP_OK) { nvs_close(handle); return -1; }
        ESP_LOGW(TAG, "explicit MyFlowHub state reset completed; disable reset before the next boot");
    }
    persistent_config_t config;
    int result = load_config(handle, &config);
    if (result == 0) result = load_identity(handle, identity);
    if (result == 0) {
        platform->identity = *identity;
        memcpy(platform->parent_public_key, config.parent_public_key, sizeof(platform->parent_public_key));
    }
    nvs_close(handle);
    return result;
}

static void wifi_event(void *argument, esp_event_base_t base, int32_t id, void *data) {
    (void)argument; (void)data;
    if (base == WIFI_EVENT && id == WIFI_EVENT_STA_START) {
        esp_wifi_connect();
    } else if (base == WIFI_EVENT && id == WIFI_EVENT_STA_DISCONNECTED) {
        if (++wifi_attempts <= 10) esp_wifi_connect(); else xEventGroupSetBits(wifi_events, WIFI_FAILED_BIT);
    } else if (base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) {
        wifi_attempts = 0;
        xEventGroupSetBits(wifi_events, WIFI_CONNECTED_BIT);
    }
}

int mfh_esp_connect_wifi(void) {
    if (CONFIG_MFH_WIFI_SSID[0] == '\0') return -1;
    wifi_events = xEventGroupCreate();
    if (wifi_events == NULL || esp_netif_init() != ESP_OK || esp_event_loop_create_default() != ESP_OK || esp_netif_create_default_wifi_sta() == NULL) return -1;
    wifi_init_config_t init = WIFI_INIT_CONFIG_DEFAULT();
    if (esp_wifi_init(&init) != ESP_OK) return -1;
    esp_event_handler_instance_t any_wifi, got_ip;
    ESP_ERROR_CHECK(esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID, wifi_event, NULL, &any_wifi));
    ESP_ERROR_CHECK(esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP, wifi_event, NULL, &got_ip));
    wifi_config_t config = {0};
    if (strlen(CONFIG_MFH_WIFI_SSID) >= sizeof(config.sta.ssid) || strlen(CONFIG_MFH_WIFI_PASSWORD) >= sizeof(config.sta.password)) return -1;
    memcpy(config.sta.ssid, CONFIG_MFH_WIFI_SSID, strlen(CONFIG_MFH_WIFI_SSID));
    memcpy(config.sta.password, CONFIG_MFH_WIFI_PASSWORD, strlen(CONFIG_MFH_WIFI_PASSWORD));
    config.sta.pmf_cfg.capable = true;
    if (esp_wifi_set_mode(WIFI_MODE_STA) != ESP_OK || esp_wifi_set_config(WIFI_IF_STA, &config) != ESP_OK || esp_wifi_start() != ESP_OK) return -1;
    EventBits_t bits = xEventGroupWaitBits(wifi_events, WIFI_CONNECTED_BIT | WIFI_FAILED_BIT, pdFALSE, pdFALSE, pdMS_TO_TICKS(30000));
    return (bits & WIFI_CONNECTED_BIT) != 0 ? 0 : -1;
}

int mfh_esp_connect_tcp(mfh_esp_platform_t *platform, const char *host, uint16_t port) {
    char service[6];
    snprintf(service, sizeof(service), "%u", (unsigned)port);
    struct addrinfo hints = {.ai_family = AF_UNSPEC, .ai_socktype = SOCK_STREAM};
    struct addrinfo *addresses = NULL;
    if (getaddrinfo(host, service, &hints, &addresses) != 0) return -1;
    int socket_fd = -1;
    for (struct addrinfo *address = addresses; address != NULL; address = address->ai_next) {
        socket_fd = socket(address->ai_family, address->ai_socktype, address->ai_protocol);
        if (socket_fd >= 0 && connect(socket_fd, address->ai_addr, address->ai_addrlen) == 0) break;
        if (socket_fd >= 0) close(socket_fd);
        socket_fd = -1;
    }
    freeaddrinfo(addresses);
    platform->socket_fd = socket_fd;
    return socket_fd >= 0 ? 0 : -1;
}

void mfh_esp_close(mfh_esp_platform_t *platform) {
    if (platform != NULL && platform->socket_fd >= 0) { shutdown(platform->socket_fd, SHUT_RDWR); close(platform->socket_fd); platform->socket_fd = -1; }
}

void mfh_esp_log_identity(const mfh_identity_t *identity) {
    unsigned char encoded[64]; size_t written = 0;
    if (mbedtls_base64_encode(encoded, sizeof(encoded), &written, identity->public_key, MFH_PUBLIC_KEY_SIZE) == 0) {
        encoded[written] = '\0';
        ESP_LOGI(TAG, "node_id=%llu public_key=%s", (unsigned long long)identity->node_id, encoded);
    }
}

int mfh_esp_read(void *context, uint8_t *output, size_t length) {
    mfh_esp_platform_t *platform = context; size_t offset = 0;
    while (offset < length) { ssize_t count = recv(platform->socket_fd, output + offset, length - offset, 0); if (count <= 0) return -1; offset += (size_t)count; }
    return 0;
}

int mfh_esp_write(void *context, const uint8_t *input, size_t length) {
    mfh_esp_platform_t *platform = context; size_t offset = 0;
    while (offset < length) { ssize_t count = send(platform->socket_fd, input + offset, length - offset, 0); if (count <= 0) return -1; offset += (size_t)count; }
    return 0;
}

int mfh_esp_random(void *context, uint8_t *output, size_t length) { (void)context; esp_fill_random(output, length); return 0; }

int mfh_esp_sign(void *context, const uint8_t *message, size_t message_len, uint8_t signature[MFH_SIGNATURE_SIZE]) {
    const mfh_esp_platform_t *platform = context; unsigned long long written = 0;
    return crypto_sign_detached(signature, &written, message, message_len, platform->identity.private_key) == 0 && written == MFH_SIGNATURE_SIZE ? 0 : -1;
}

int mfh_esp_verify_ack(void *context, const uint8_t *payload, size_t payload_len, uint64_t parent_id, uint64_t child_id,
                       const uint8_t nonce[MFH_NONCE_SIZE], uint64_t epoch) {
    const mfh_esp_platform_t *platform = context;
    char json[1024]; if (payload_len == 0 || payload_len >= sizeof(json)) return -1;
    memcpy(json, payload, payload_len); json[payload_len] = '\0';
    cJSON *root = cJSON_Parse(json); if (root == NULL) return -1;
    const cJSON *node = cJSON_GetObjectItemCaseSensitive(root, "node_id"), *child = cJSON_GetObjectItemCaseSensitive(root, "child_id");
    const cJSON *child_nonce = cJSON_GetObjectItemCaseSensitive(root, "child_nonce"), *topology = cJSON_GetObjectItemCaseSensitive(root, "topology_epoch");
    const cJSON *signature_json = cJSON_GetObjectItemCaseSensitive(root, "signature");
    bool valid = cJSON_IsNumber(node) && cJSON_IsNumber(child) && cJSON_IsArray(child_nonce) && cJSON_IsNumber(topology) && cJSON_IsString(signature_json) &&
                 node->valuedouble == (double)parent_id && child->valuedouble == (double)child_id && topology->valuedouble == (double)epoch && cJSON_GetArraySize(child_nonce) == MFH_NONCE_SIZE;
    for (size_t i = 0; valid && i < MFH_NONCE_SIZE; ++i) { const cJSON *value = cJSON_GetArrayItem(child_nonce, (int)i); valid = cJSON_IsNumber(value) && value->valueint == nonce[i]; }
    uint8_t signature[MFH_SIGNATURE_SIZE]; size_t signature_len = 0;
    if (!valid || mbedtls_base64_decode(signature, sizeof(signature), &signature_len, (const unsigned char *)signature_json->valuestring, strlen(signature_json->valuestring)) != 0 || signature_len != sizeof(signature)) { cJSON_Delete(root); return -1; }
    uint8_t message[8 + 8 + 8 + MFH_NONCE_SIZE + 8]; size_t offset = 0;
    memcpy(message + offset, "MFH4-ACK", 8); offset += 8; write_u64(message + offset, parent_id); offset += 8; write_u64(message + offset, child_id); offset += 8;
    memcpy(message + offset, nonce, MFH_NONCE_SIZE); offset += MFH_NONCE_SIZE; write_u64(message + offset, epoch);
    int result = crypto_sign_verify_detached(signature, message, sizeof(message), platform->parent_public_key);
    cJSON_Delete(root);
    return result;
}
