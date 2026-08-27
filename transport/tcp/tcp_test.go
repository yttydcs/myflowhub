package tcp

import (
	"testing"

	"github.com/yttydcs/myflowhub/internal/transporttest"
	"github.com/yttydcs/myflowhub/runtime/link"
)

func TestTransportContract(t *testing.T) {
	transporttest.Run(t, Driver{}, link.Endpoint("127.0.0.1:0"))
}
