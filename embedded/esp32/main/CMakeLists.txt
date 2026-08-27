idf_component_register(
    SRCS "app_main.c" "esp_platform.c"
    INCLUDE_DIRS "."
    REQUIRES myflowhub_embedded esp_event esp_netif esp_wifi json libsodium mbedtls nvs_flash
)
