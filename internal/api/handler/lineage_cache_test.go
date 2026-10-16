package handler

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compliance-framework/api/internal/service/leverage"
	"github.com/compliance-framework/api/internal/service/relational"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLineageBuilds counts builds and hands out a distinct engine per build.
type fakeLineageBuilds struct {
	calls atomic.Int32
	err   error
}

func (f *fakeLineageBuilds) build(_, _ *uuid.UUID) (*lineageEngine, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	return &lineageEngine{}, nil
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestLineageCache(ttl time.Duration, maxEntries int) (*lineageEngineCache, *fakeLineageBuilds, *fakeClock) {
	builds := &fakeLineageBuilds{}
	clock := &fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	c := newLineageEngineCache(ttl, maxEntries, builds.build)
	c.now = clock.now
	return c, builds, clock
}

func TestLineageEngineCacheReusesWithinTTL(t *testing.T) {
	c, builds, clock := newTestLineageCache(time.Minute, 4)

	first, firstAt, err := c.get(nil, nil, false)
	require.NoError(t, err)
	clock.advance(59 * time.Second)
	second, secondAt, err := c.get(nil, nil, false)
	require.NoError(t, err)

	assert.Same(t, first, second)
	assert.Equal(t, firstAt, secondAt)
	assert.EqualValues(t, 1, builds.calls.Load())

	clock.advance(time.Second)
	third, thirdAt, err := c.get(nil, nil, false)
	require.NoError(t, err)
	assert.NotSame(t, first, third, "expired entry is rebuilt")
	assert.Equal(t, clock.t, thirdAt)
	assert.EqualValues(t, 2, builds.calls.Load())
}

func TestLineageEngineCacheRefreshRebuilds(t *testing.T) {
	c, builds, clock := newTestLineageCache(time.Minute, 4)

	first, _, err := c.get(nil, nil, false)
	require.NoError(t, err)
	clock.advance(time.Second)
	refreshed, refreshedAt, err := c.get(nil, nil, true)
	require.NoError(t, err)
	assert.NotSame(t, first, refreshed)
	assert.Equal(t, clock.t, refreshedAt)

	again, _, err := c.get(nil, nil, false)
	require.NoError(t, err)
	assert.Same(t, refreshed, again, "the refreshed engine replaces the cached one")
	assert.EqualValues(t, 2, builds.calls.Load())
}

func TestLineageEngineCacheKeysByScope(t *testing.T) {
	c, builds, _ := newTestLineageCache(time.Minute, 4)
	ssp, component := uuid.New(), uuid.New()

	global, _, _ := c.get(nil, nil, false)
	scoped, _, _ := c.get(&ssp, nil, false)
	both, _, _ := c.get(&ssp, &component, false)
	scopedAgain, _, _ := c.get(&ssp, nil, false)

	assert.NotSame(t, global, scoped)
	assert.NotSame(t, scoped, both)
	assert.Same(t, scoped, scopedAgain)
	assert.EqualValues(t, 3, builds.calls.Load())
}

func TestLineageEngineCacheEvictsOldest(t *testing.T) {
	c, builds, clock := newTestLineageCache(time.Hour, 2)
	a, b, d := uuid.New(), uuid.New(), uuid.New()

	_, _, _ = c.get(&a, nil, false)
	clock.advance(time.Second)
	_, _, _ = c.get(&b, nil, false)
	clock.advance(time.Second)
	_, _, _ = c.get(&d, nil, false) // evicts a

	_, _, _ = c.get(&b, nil, false)
	_, _, _ = c.get(&d, nil, false)
	assert.EqualValues(t, 3, builds.calls.Load(), "b and d stay cached")
	_, _, _ = c.get(&a, nil, false)
	assert.EqualValues(t, 4, builds.calls.Load(), "a was evicted")
	assert.Len(t, c.entries, 2)
}

func TestLineageEngineCacheZeroTTLAlwaysBuilds(t *testing.T) {
	c, builds, _ := newTestLineageCache(0, 4)

	first, _, _ := c.get(nil, nil, false)
	second, _, _ := c.get(nil, nil, false)
	assert.NotSame(t, first, second)
	assert.EqualValues(t, 2, builds.calls.Load())
	assert.Empty(t, c.entries)
}

func TestLineageEngineCacheDoesNotCacheErrors(t *testing.T) {
	c, builds, _ := newTestLineageCache(time.Minute, 4)
	builds.err = errors.New("db down")

	_, _, err := c.get(nil, nil, false)
	require.Error(t, err)
	builds.err = nil
	engine, _, err := c.get(nil, nil, false)
	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.EqualValues(t, 2, builds.calls.Load())
}

func TestLineageEngineCacheSharesConcurrentBuilds(t *testing.T) {
	release := make(chan struct{})
	var calls atomic.Int32
	c := newLineageEngineCache(time.Minute, 4, func(_, _ *uuid.UUID) (*lineageEngine, error) {
		calls.Add(1)
		<-release
		return &lineageEngine{}, nil
	})

	const callers = 8
	engines := make([]*lineageEngine, callers)
	var started, done sync.WaitGroup
	for i := 0; i < callers; i++ {
		started.Add(1)
		done.Add(1)
		go func(i int) {
			defer done.Done()
			started.Done()
			engines[i], _, _ = c.get(nil, nil, false)
		}(i)
	}
	started.Wait()
	// Callers that reach the build late hit the stored entry instead, so the
	// assertions hold whatever the scheduling; the pause just makes overlap likely.
	time.Sleep(20 * time.Millisecond)
	close(release)
	done.Wait()

	assert.EqualValues(t, 1, calls.Load())
	for _, e := range engines {
		assert.Same(t, engines[0], e)
	}
}

func TestLineageEngineCacheReleasesExpiredEntryOnGet(t *testing.T) {
	for name, buildErr := range map[string]error{
		"rebuild succeeds": nil,
		"rebuild fails":    errors.New("db down"),
	} {
		t.Run(name, func(t *testing.T) {
			c, builds, clock := newTestLineageCache(time.Minute, 4)
			expired, _, err := c.get(nil, nil, false)
			require.NoError(t, err)
			clock.advance(time.Minute)

			heldDuringRebuild := true
			c.build = func(sspID, componentID *uuid.UUID) (*lineageEngine, error) {
				c.mu.Lock()
				_, heldDuringRebuild = c.entries[lineageScopeKey{}]
				c.mu.Unlock()
				builds.err = buildErr
				return builds.build(sspID, componentID)
			}

			engine, _, err := c.get(nil, nil, false)
			assert.False(t, heldDuringRebuild, "the expired engine is released before the rebuild")
			if buildErr != nil {
				require.Error(t, err)
				assert.Empty(t, c.entries, "the expired engine is gone even though the rebuild failed")
				return
			}
			require.NoError(t, err)
			assert.NotSame(t, expired, engine)
			require.Len(t, c.entries, 1)
			assert.Same(t, engine, c.entries[lineageScopeKey{}].engine)
		})
	}
}

// TestLineageEngineCacheRefreshDoesNotJoinInFlightBuild covers a refresh arriving while
// an ordinary miss is still building: the refresh starts its own build, and its engine
// is what stays cached even when the older ordinary build finishes last.
func TestLineageEngineCacheRefreshDoesNotJoinInFlightBuild(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}
	started := make(chan int, 2)
	release := []chan struct{}{make(chan struct{}), make(chan struct{})}
	var calls atomic.Int32
	c := newLineageEngineCache(time.Minute, 4, func(_, _ *uuid.UUID) (*lineageEngine, error) {
		n := int(calls.Add(1)) - 1
		started <- n
		<-release[n]
		return &lineageEngine{}, nil
	})
	c.now = clock.now

	awaitStart := func(want int) {
		t.Helper()
		select {
		case n := <-started:
			require.Equal(t, want, n)
		case <-time.After(5 * time.Second):
			t.Fatalf("build %d never started", want)
		}
	}

	var ordinary, refreshed *lineageEngine
	ordinaryDone, refreshDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(ordinaryDone)
		ordinary, _, _ = c.get(nil, nil, false)
	}()
	awaitStart(0)

	clock.advance(time.Second) // the user saves a change, then presses Refresh
	go func() {
		defer close(refreshDone)
		refreshed, _, _ = c.get(nil, nil, true)
	}()
	awaitStart(1)

	close(release[1])
	<-refreshDone
	close(release[0])
	<-ordinaryDone

	require.NotNil(t, refreshed)
	assert.NotSame(t, ordinary, refreshed)
	assert.EqualValues(t, 2, calls.Load())
	cached, cachedAt, err := c.get(nil, nil, false)
	require.NoError(t, err)
	assert.Same(t, refreshed, cached, "the older ordinary build does not overwrite the refresh")
	assert.Equal(t, clock.t, cachedAt)
	assert.EqualValues(t, 2, calls.Load())
}

// TestLineageEngineMemoIsConcurrencySafe exercises the memo maps a shared engine
// fills while serving requests; run with -race.
func TestLineageEngineMemoIsConcurrencySafe(t *testing.T) {
	catalog := uuid.New()
	refs := make([]relational.ControlRef, 50)
	for i := range refs {
		refs[i] = relational.ControlRef{CatalogID: catalog, ControlID: uuid.NewString()}
	}
	ssps := []uuid.UUID{uuid.New(), uuid.New()}
	e := &lineageEngine{
		graph:             relational.NewControlLinkGraph(nil),
		closureCache:      map[relational.ControlRef][]relational.ControlRef{},
		assessCache:       map[assessKey]sspAssessment{},
		implStatusBySSP:   map[uuid.UUID]map[string]string{},
		leverageByControl: map[leverage.ControlKey]leverage.ControlAggregate{},
		filtersByControl:  map[relational.ControlRef][]uuid.UUID{},
		filterSSP:         map[uuid.UUID]*uuid.UUID{},
		filterMatched:     map[uuid.UUID][]int{},
	}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, ref := range refs {
				for _, ssp := range ssps {
					e.assessSSP(ref, ssp, map[string]struct{}{scopeKey(ref): {}})
				}
			}
		}()
	}
	wg.Wait()
	assert.Len(t, e.assessCache, len(refs)*len(ssps))
}
