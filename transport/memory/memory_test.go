package memory

import (
	"testing"

	"github.com/yttydcs/myflowhub/internal/transporttest"
	"github.com/yttydcs/myflowhub/runtime/link"
)

func TestTransportContract(t *testing.T) {
	network := NewNetwork()
	defer network.Close()
	transporttest.Run(t, network, link.Endpoint("contract"))
}
