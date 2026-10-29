package nodehost

import "testing"

func TestProductRuntimeCapacityKeepsPendingHeadroomAndValidatesOverrides(t *testing.T) {
	for _, value := range []string{"1", "2000", "4096", "65536"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MFH_MAX_SUBSCRIPTIONS", value)
			config, err := ProductRuntimeConfig()
			if err != nil || config.MaxPending != config.Subscriptions.MaxSubscriptions+256 {
				t.Fatalf("capacity: %+v %v", config, err)
			}
			if !config.Session.IdleAfterPeerHeartbeat {
				t.Fatal("product would disconnect a healthy legacy peer without heartbeat support")
			}
		})
	}
	for _, value := range []string{"", "0", "-1", "65537", "2.5", "many"} {
		t.Run("invalid_"+value, func(t *testing.T) {
			t.Setenv("MFH_MAX_SUBSCRIPTIONS", value)
			if _, err := ProductRuntimeConfig(); err == nil {
				t.Fatal("invalid external capacity silently accepted")
			}
		})
	}
}
