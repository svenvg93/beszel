//go:build testing

package agent

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/henrygd/beszel/internal/entities/speedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpeedtestManagerRunNowIsAsync(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		sm := newSpeedtestManagerWithRunner(func(ctx context.Context, serverID uint32) (speedtest.Result, error) {
			<-release
			return speedtest.Result{Download: 100, ServerID: serverID}, nil
		})
		defer sm.Stop()

		require.NoError(t, sm.UpsertSpeedtest(speedtest.Config{ID: "a", ServerID: 7, Interval: 60}, true))
		synctest.Wait()
		assert.Nil(t, sm.GetResults(), "upsert must return before the run finishes")

		close(release)
		synctest.Wait()
		results := sm.GetResults()
		require.Contains(t, results, "a")
		assert.Equal(t, uint64(100), results["a"].Download)
		assert.Equal(t, uint32(7), results["a"].ServerID)
		assert.NotZero(t, results["a"].RunAt)
	})
}

func TestSpeedtestManagerSerializesRuns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var active, maxActive atomic.Int32
		sm := newSpeedtestManagerWithRunner(func(ctx context.Context, serverID uint32) (speedtest.Result, error) {
			n := active.Add(1)
			defer active.Add(-1)
			for {
				current := maxActive.Load()
				if n <= current || maxActive.CompareAndSwap(current, n) {
					break
				}
			}
			time.Sleep(30 * time.Second)
			return speedtest.Result{Download: uint64(serverID)}, nil
		})
		defer sm.Stop()

		for i, id := range []string{"a", "b", "c"} {
			require.NoError(t, sm.UpsertSpeedtest(speedtest.Config{ID: id, ServerID: uint32(i + 1), Interval: 60}, true))
		}
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		assert.Len(t, sm.GetResults(), 3)
		assert.Equal(t, int32(1), maxActive.Load())
	})
}

func TestSpeedtestManagerRecordsErrors(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sm := newSpeedtestManagerWithRunner(func(context.Context, uint32) (speedtest.Result, error) {
			return speedtest.Result{}, errors.New("not installed")
		})
		defer sm.Stop()
		require.NoError(t, sm.UpsertSpeedtest(speedtest.Config{ID: "a", ServerID: 3, Interval: 60}, true))
		synctest.Wait()
		result := sm.GetResults()["a"]
		assert.Equal(t, "not installed", result.Error)
		assert.Equal(t, uint32(3), result.ServerID)
		assert.NotZero(t, result.RunAt)
	})
}

func TestSpeedtestManagerSync(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		runs := map[uint32]int{}
		sm := newSpeedtestManagerWithRunner(func(ctx context.Context, serverID uint32) (speedtest.Result, error) {
			mu.Lock()
			runs[serverID]++
			mu.Unlock()
			return speedtest.Result{ServerID: serverID}, nil
		})
		defer sm.Stop()

		sm.SyncSpeedtests([]speedtest.Config{{ID: "a", ServerID: 1, Interval: 15}, {ID: "b", ServerID: 2, Interval: 15}, {ID: ""}})
		assert.Len(t, sm.tasks, 2)
		// First runs are staggered but capped.
		time.Sleep(speedtestMaxStartDelay + time.Second)
		synctest.Wait()
		assert.Len(t, sm.GetResults(), 2)

		// Changing the interval keeps the latest result; removed speedtests stop.
		sm.SyncSpeedtests([]speedtest.Config{{ID: "a", ServerID: 1, Interval: 30}})
		assert.Len(t, sm.tasks, 1)
		assert.Contains(t, sm.GetResults(), "a")

		require.NoError(t, sm.HandleSyncRequest(speedtest.SyncRequest{Action: monitor.SyncActionDelete, Config: speedtest.Config{ID: "a"}}))
		assert.Empty(t, sm.tasks)
		assert.Error(t, sm.HandleSyncRequest(speedtest.SyncRequest{Action: monitor.SyncActionDelete}))
		assert.Error(t, sm.HandleSyncRequest(speedtest.SyncRequest{Action: 99}))

		mu.Lock()
		before := runs[2]
		mu.Unlock()
		time.Sleep(time.Hour)
		synctest.Wait()
		mu.Lock()
		assert.Equal(t, before, runs[2], "removed speedtest must not run again")
		mu.Unlock()
	})
}

func TestSpeedtestManagerIntervalFloor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var runs atomic.Int32
		sm := newSpeedtestManagerWithRunner(func(context.Context, uint32) (speedtest.Result, error) {
			runs.Add(1)
			return speedtest.Result{}, nil
		})
		defer sm.Stop()
		require.NoError(t, sm.UpsertSpeedtest(speedtest.Config{ID: "a", Interval: 0}, true))
		time.Sleep(10*time.Duration(speedtest.MinInterval)*time.Minute - time.Second)
		synctest.Wait()
		// One immediate run plus nine more at the minimum interval.
		assert.Equal(t, int32(10), runs.Load())
	})
}
