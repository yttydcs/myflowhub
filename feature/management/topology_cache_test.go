package management

import (
	"strconv"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/tree"
)

func TestTopologyDepthCacheIsBoundedAndIsolated(t *testing.T) {
	snapshot, err := buildTopologySnapshot(tree.TopologySnapshot{Local: 1, Version: tree.TopologyVersion{MembershipEpoch: 1}}, "", strings.Repeat("a", 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	for depth := 2; depth < 50; depth++ {
		result := snapshot.query(depth)
		if result.err != nil {
			t.Fatal(result.err)
		}
		if len(snapshot.cache) > topologyCacheEntries || snapshot.cacheBytes > topologyCacheBytes {
			t.Fatal("depth cache grew without bound")
		}
	}
	for _, depth := range []int{0, 1} {
		if _, exists := snapshot.cache[depth]; exists {
			t.Fatal("precomputed depths consumed dynamic slots")
		}
	}
}

func BenchmarkTopologySnapshot(b *testing.B) {
	for _, count := range []int{16, 4000} {
		state := tree.TopologySnapshot{Local: 1, Version: tree.TopologyVersion{MembershipEpoch: 1}}
		state.Relations = append(state.Relations, tree.Relation{Node: 2, Parent: 1})
		for id := 3; id <= count; id++ {
			state.Relations = append(state.Relations, tree.Relation{Node: protocol.NodeID(id), Parent: 2})
		}
		snapshot, err := buildTopologySnapshot(state, "", strings.Repeat("a", 32), 1)
		if err != nil {
			b.Fatal(err)
		}
		for _, name := range []string{"shallow-hit", "full-hit", "depth-miss", "rebuild"} {
			b.Run(strings.Join([]string{strconv.Itoa(count) + "-nodes", name}, "/"), func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(float64(len(snapshot.shallow.payload)), "shallow-bytes")
				b.ReportMetric(float64(len(snapshot.legacy)), "legacy-bytes")
				for i := 0; i < b.N; i++ {
					switch name {
					case "shallow-hit":
						snapshot.query(1)
					case "full-hit":
						snapshot.query(0)
					case "depth-miss":
						snapshot.encode(2)
					case "rebuild":
						if _, err := buildTopologySnapshot(state, "", snapshot.instanceID, 1); err != nil {
							b.Fatal(err)
						}
					}
				}
			})
		}
	}
}
