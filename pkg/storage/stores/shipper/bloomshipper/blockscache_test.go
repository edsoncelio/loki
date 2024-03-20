package bloomshipper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/flagext"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/grafana/loki/pkg/storage/chunk/cache"
)

var (
	logger = log.NewNopLogger()
)

func TestBlocksCacheConfig_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc string
		cfg  BlocksCacheConfig
		err  error
	}{
		{
			desc: "not enabled does not yield error when incorrectly configured",
			cfg:  BlocksCacheConfig{Enabled: false},
			err:  nil,
		},
		{
			desc: "ttl not set",
			cfg: BlocksCacheConfig{
				Enabled:   true,
				SoftLimit: 1,
				HardLimit: 2,
			},
			err: errors.New("blocks cache ttl must not be 0"),
		},
		{
			desc: "soft limit not set",
			cfg: BlocksCacheConfig{
				Enabled:   true,
				TTL:       1,
				HardLimit: 2,
			},
			err: errors.New("blocks cache soft_limit must not be 0"),
		},
		{
			desc: "hard limit not set",
			cfg: BlocksCacheConfig{
				Enabled:   true,
				TTL:       1,
				SoftLimit: 1,
			},
			err: errors.New("blocks cache soft_limit must not be greater than hard_limit"),
		},
		{
			desc: "soft limit greater than hard limit",
			cfg: BlocksCacheConfig{
				Enabled:   true,
				TTL:       1,
				SoftLimit: 2,
				HardLimit: 1,
			},
			err: errors.New("blocks cache soft_limit must not be greater than hard_limit"),
		},
		{
			desc: "all good",
			cfg: BlocksCacheConfig{
				Enabled:   true,
				TTL:       1,
				SoftLimit: 1,
				HardLimit: 2,
			},
			err: nil,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBlocksCache_ErrorCases(t *testing.T) {
	cfg := BlocksCacheConfig{
		Enabled:   true,
		TTL:       time.Hour,
		SoftLimit: flagext.Bytes(100),
		HardLimit: flagext.Bytes(200),
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	t.Cleanup(cache.Stop)

	t.Run("cancelled context", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err := cache.Put(ctx, "key", BlockDirectory{})
		require.ErrorContains(t, err, "context canceled")

		err = cache.PutMany(ctx, []string{"key"}, []BlockDirectory{{}})
		require.ErrorContains(t, err, "context canceled")

		_, ok := cache.Get(ctx, "key")
		require.False(t, ok)
	})

	t.Run("duplicate keys", func(t *testing.T) {
		ctx := context.Background()

		err := cache.Put(ctx, "key", CacheValue("a", 10))
		require.NoError(t, err)

		err = cache.Put(ctx, "key", CacheValue("b", 10))
		require.ErrorContains(t, err, fmt.Sprintf(errAlreadyExists, "key"))
	})

	t.Run("multierror when putting many fails", func(t *testing.T) {
		ctx := context.Background()

		err := cache.PutMany(
			ctx,
			[]string{"x", "y", "x", "z"},
			[]BlockDirectory{
				CacheValue("x", 2),
				CacheValue("y", 2),
				CacheValue("x", 2),
				CacheValue("z", 250),
			},
		)
		require.ErrorContains(t, err, "2 errors: entry already exists: x; entry exceeds hard limit: z")
	})

	// TODO(chaudum): Implement blocking evictions
	t.Run("todo: blocking evictions", func(t *testing.T) {
		ctx := context.Background()

		err := cache.Put(ctx, "a", CacheValue("a", 5))
		require.NoError(t, err)

		err = cache.Put(ctx, "b", CacheValue("b", 10))
		require.NoError(t, err)

		err = cache.Put(ctx, "c", CacheValue("c", 190))
		require.Error(t, err, "todo: implement waiting for evictions to free up space")
	})
}

func CacheValue(path string, size int64) BlockDirectory {
	return BlockDirectory{
		Path: path,
		size: size,
	}
}

func TestBlocksCache_PutAndGet(t *testing.T) {
	cfg := BlocksCacheConfig{
		Enabled:   true,
		TTL:       time.Hour,
		SoftLimit: flagext.Bytes(10),
		HardLimit: flagext.Bytes(20),
		// no need for TTL evictions
		PurgeInterval: time.Minute,
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	t.Cleanup(cache.Stop)

	ctx := context.Background()
	err := cache.PutMany(
		ctx,
		[]string{"a", "b", "c"},
		[]BlockDirectory{CacheValue("a", 1), CacheValue("b", 2), CacheValue("c", 3)},
	)
	require.NoError(t, err)

	// key does not exist
	_, found := cache.Get(ctx, "d")
	require.False(t, found)

	// existing keys
	_, found = cache.Get(ctx, "b")
	require.True(t, found)
	_, found = cache.Get(ctx, "c")
	require.True(t, found)
	_, found = cache.Get(ctx, "a")
	require.True(t, found)

	require.Equal(t, 3, cache.lru.Len())

	// check LRU order
	elem := cache.lru.Front()
	require.Equal(t, "a", elem.Value.(*Entry).Key)
	require.Equal(t, int32(1), elem.Value.(*Entry).refCount.Load())

	elem = elem.Next()
	require.Equal(t, "c", elem.Value.(*Entry).Key)
	require.Equal(t, int32(1), elem.Value.(*Entry).refCount.Load())

	elem = elem.Next()
	require.Equal(t, "b", elem.Value.(*Entry).Key)
	require.Equal(t, int32(1), elem.Value.(*Entry).refCount.Load())

	// fetch more
	_, _ = cache.Get(ctx, "a")
	_, _ = cache.Get(ctx, "a")
	_, _ = cache.Get(ctx, "b")

	// LRU order changed
	elem = cache.lru.Front()
	require.Equal(t, "b", elem.Value.(*Entry).Key)
	require.Equal(t, int32(2), elem.Value.(*Entry).refCount.Load())

	elem = elem.Next()
	require.Equal(t, "a", elem.Value.(*Entry).Key)
	require.Equal(t, int32(3), elem.Value.(*Entry).refCount.Load())

	elem = elem.Next()
	require.Equal(t, "c", elem.Value.(*Entry).Key)
	require.Equal(t, int32(1), elem.Value.(*Entry).refCount.Load())

}

func TestBlocksCache_TTLEviction(t *testing.T) {
	cfg := BlocksCacheConfig{
		Enabled:   true,
		TTL:       100 * time.Millisecond,
		SoftLimit: flagext.Bytes(10),
		HardLimit: flagext.Bytes(20),

		PurgeInterval: 100 * time.Millisecond,
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	t.Cleanup(cache.Stop)

	ctx := context.Background()

	err := cache.Put(ctx, "a", CacheValue("a", 5))
	require.NoError(t, err)
	time.Sleep(75 * time.Millisecond)

	err = cache.Put(ctx, "b", CacheValue("b", 5))
	require.NoError(t, err)
	time.Sleep(75 * time.Millisecond)

	// "a" got evicted
	_, found := cache.Get(ctx, "a")
	require.False(t, found)

	// "b" is still in cache
	_, found = cache.Get(ctx, "b")
	require.True(t, found)

	require.Equal(t, 1, cache.lru.Len())
	require.Equal(t, 1, len(cache.entries))
}

func TestBlocksCache_LRUEviction(t *testing.T) {
	cfg := BlocksCacheConfig{
		Enabled:   true,
		TTL:       time.Hour,
		SoftLimit: flagext.Bytes(15),
		HardLimit: flagext.Bytes(20),
		// no need for TTL evictions
		PurgeInterval: time.Minute,
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	t.Cleanup(cache.Stop)

	ctx := context.Background()

	// oldest with refcount - will not be evicted
	err := cache.PutInc(ctx, "a", CacheValue("a", 4))
	require.NoError(t, err)
	// will become newest with Get() call
	err = cache.Put(ctx, "b", CacheValue("b", 4))
	require.NoError(t, err)
	// oldest without refcount - will be evicted
	err = cache.Put(ctx, "c", CacheValue("c", 4))
	require.NoError(t, err)

	// increase ref counter on "b"
	_, found := cache.Get(ctx, "b")
	require.True(t, found)

	// exceed soft limit
	err = cache.Put(ctx, "d", CacheValue("d", 4))
	require.NoError(t, err)

	time.Sleep(time.Second)

	require.Equal(t, 3, cache.lru.Len())
	require.Equal(t, 3, len(cache.entries))

	// key "b" was evicted because it was the oldest
	// and it had no ref counts
	_, found = cache.Get(ctx, "c")
	require.False(t, found)

	require.Equal(t, int64(12), cache.currSizeBytes)
}

func TestBlocksCache_RefCounter(t *testing.T) {
	cfg := BlocksCacheConfig{
		Enabled:   true,
		TTL:       time.Hour,
		SoftLimit: flagext.Bytes(10),
		HardLimit: flagext.Bytes(20),
		// no need for TTL evictions
		PurgeInterval: time.Minute,
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	t.Cleanup(cache.Stop)

	ctx := context.Background()

	_ = cache.PutInc(ctx, "a", CacheValue("a", 5))
	require.Equal(t, int32(1), cache.entries["a"].Value.(*Entry).refCount.Load())

	_, _ = cache.Get(ctx, "a")
	require.Equal(t, int32(2), cache.entries["a"].Value.(*Entry).refCount.Load())

	_ = cache.Release(ctx, "a")
	require.Equal(t, int32(1), cache.entries["a"].Value.(*Entry).refCount.Load())

	_ = cache.Release(ctx, "a")
	require.Equal(t, int32(0), cache.entries["a"].Value.(*Entry).refCount.Load())
}

func prepareBenchmark(b *testing.B) map[string]BlockDirectory {
	b.Helper()

	entries := make(map[string]BlockDirectory)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("block-%04x", i)
		entries[key] = BlockDirectory{
			BlockRef:                    BlockRef{},
			Path:                        fmt.Sprintf("blooms/%s", key),
			removeDirectoryTimeout:      time.Minute,
			refCount:                    atomic.NewInt32(0),
			logger:                      logger,
			activeQueriersCheckInterval: time.Minute,
			size:                        4 << 10,
		}
	}
	return entries
}

func Benchmark_BlocksCacheOld(b *testing.B) {
	prepareBenchmark(b)
	b.StopTimer()
	cfg := cache.EmbeddedCacheConfig{
		Enabled:       true,
		MaxSizeMB:     100,
		MaxSizeItems:  10000,
		TTL:           time.Hour,
		PurgeInterval: time.Hour,
	}
	cache := NewBlocksCache(cfg, nil, logger)
	entries := prepareBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.StartTimer()

	// write
	for k, v := range entries {
		err := cache.Store(ctx, []string{k}, []BlockDirectory{v})
		require.NoError(b, err)
	}
	for i := 0; i < b.N; i++ {
		// read
		for k := range entries {
			_, _ = cache.Get(ctx, k)
		}
	}

}

func Benchmark_BlocksCacheNew(b *testing.B) {
	prepareBenchmark(b)
	b.StopTimer()
	cfg := BlocksCacheConfig{
		Enabled:       true,
		SoftLimit:     100 << 20,
		HardLimit:     120 << 20,
		TTL:           time.Hour,
		PurgeInterval: time.Hour,
	}
	cache := NewFsBlocksCache(cfg, nil, logger)
	entries := prepareBenchmark(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.StartTimer()

	// write
	for k, v := range entries {
		_ = cache.PutMany(ctx, []string{k}, []BlockDirectory{v})
	}
	// read
	for i := 0; i < b.N; i++ {
		for k := range entries {
			_, _ = cache.Get(ctx, k)
		}
	}
}
