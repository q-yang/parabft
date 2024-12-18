package pacemaker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// receive only one tmo
func TestRemoteTmo1(t *testing.T) {
	pm := NewPacemaker(4)
	tmo1 := &TMO{
		View:   2,
		NodeID: "1",
	}
	isBuilt, tc := pm.ProcessRemoteTmo(tmo1)
	fmt.Println(isBuilt)
	require.False(t, isBuilt)
	require.Nil(t, tc)
}

// receive only two tmo
func TestRemoteTmo2(t *testing.T) {
	pm := NewPacemaker(4)
	tmo1 := &TMO{
		View:   2,
		NodeID: "1",
	}
	isBuilt, tc := pm.ProcessRemoteTmo(tmo1)
	fmt.Println("收到一个超时消息 ", isBuilt)
	tmo2 := &TMO{
		View:   2,
		NodeID: "2",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo2)
	fmt.Println("收到两个超时消息 ", isBuilt)
	require.False(t, isBuilt)
	require.Nil(t, tc)
}

// receive only three tmo
func TestRemoteTmo3(t *testing.T) {
	pm := NewPacemaker(4)
	tmo1 := &TMO{
		View:   2,
		NodeID: "1",
	}
	isBuilt, tc := pm.ProcessRemoteTmo(tmo1)
	fmt.Println("收到一个超时消息 ", isBuilt)
	tmo2 := &TMO{
		View:   2,
		NodeID: "2",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo2)
	fmt.Println("收到两个超时消息 ", isBuilt)
	tmo3 := &TMO{
		View:   2,
		NodeID: "3",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo3)
	fmt.Println("收到三个超时消息 ", isBuilt)
	fmt.Println(tc)
	require.True(t, isBuilt)
	require.NotNil(t, tc)
}

// receive four tmo
func TestRemoteTmo4(t *testing.T) {
	pm := NewPacemaker(4)
	tmo1 := &TMO{
		View:   2,
		NodeID: "1",
	}
	isBuilt, tc := pm.ProcessRemoteTmo(tmo1)
	fmt.Println("收到一个超时消息 ", isBuilt)
	tmo2 := &TMO{
		View:   2,
		NodeID: "2",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo2)
	fmt.Println("收到两个超时消息 ", isBuilt)
	tmo3 := &TMO{
		View:   2,
		NodeID: "3",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo3)
	fmt.Println("收到三个超时消息 ", isBuilt)

	tmo4 := &TMO{
		View:   2,
		NodeID: "4",
	}
	isBuilt, tc = pm.ProcessRemoteTmo(tmo4)
	fmt.Println("收到四个超时消息 ", isBuilt)
	fmt.Println(tc)
	require.True(t, isBuilt)
	require.NotNil(t, tc)
	// require.False(t, isBuilt)
	// require.NotNil(t, tc)
}
