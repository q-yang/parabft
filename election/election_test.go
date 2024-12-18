package election

import (
	"fmt"
	"testing"

	"github.com/gitferry/bamboo/types"
	"github.com/stretchr/testify/require"
)

func TestRotation_IsLeader(t *testing.T) {
	elect := NewRotation(4)
	leaderID := elect.FindLeaderFor(3)
	fmt.Println(leaderID.Node())
	require.True(t, elect.IsLeader(leaderID, 13))

	leaderID = elect.FindLeaderFor(4)
	//require.Equal(t, "3", leaderID)
	require.Equal(t, "3", string(leaderID))

	leaderID = elect.FindLeaderFor(3)
	require.True(t, elect.IsLeader(leaderID, 3))
}

func TestRotation_LeaderList(t *testing.T) {
	elect := NewRotation(4)

	for i := 1; i <= 10000; i++ {
		leaderID := elect.FindLeaderFor(types.View(i))
		fmt.Printf("view: %v, node id: %v\n", i, leaderID.Node())
	}
}
