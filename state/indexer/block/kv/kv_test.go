package kv_test

import (
	"context"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/types"
	ds "github.com/ipfs/go-datastore"
	ktds "github.com/ipfs/go-datastore/keytransform"
	"github.com/stretchr/testify/require"

	blockidxkv "github.com/LastL2/cuberollkit/state/indexer/block/kv"
	"github.com/LastL2/cuberollkit/store"
)

func TestBlockIndexer(t *testing.T) {
	kvStore, err := store.NewDefaultInMemoryKVStore()
	require.NoError(t, err)
	prefixStore := (ktds.Wrap(kvStore, ktds.PrefixTransform{Prefix: ds.NewKey("block_events")}).Children()[0]).(ds.TxnDatastore)
	indexer := blockidxkv.New(context.Background(), prefixStore)

	require.NoError(t, indexer.Index(types.EventDataNewBlockEvents{
		Height: 1,
		Events: []abci.Event{
			{
				Type: "begin_event",
				Attributes: []abci.EventAttribute{
					{
						Key:   "proposer",
						Value: "FCAA001",
						Index: true,
					},
				},
			},
			{
				Type: "end_event",
				Attributes: []abci.EventAttribute{
					{
						Key:   "foo",
						Value: "100",
						Index: true,
					},
				},
			},
		},
	}))

	for i := 2; i < 12; i++ {
		var index bool
		if i%2 == 0 {
			index = true
		}

		require.NoError(t, indexer.Index(types.EventDataNewBlockEvents{
			Height: int64(i),
			Events: []abci.Event{
				{
					Type: "begin_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   "proposer",
							Value: "FCAA001",
							Index: true,
						},
					},
				},
				{
					Type: "end_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   "foo",
							Value: fmt.Sprintf("%d", i),
							Index: index,
						},
					},
				},
			},
		}))
	}

	testCases := map[string]struct {
		q       *query.Query
		results []int64
	}{
		"block.height = 100": {
			q:       query.MustCompile(`block.height = 100`),
			results: []int64{},
		},
		"block.height = 5": {
			q:       query.MustCompile(`block.height = 5`),
			results: []int64{5},
		},
		"begin_event.key1 = 'value1'": {
			q:       query.MustCompile(`begin_event.key1 = 'value1'`),
			results: []int64{},
		},
		"begin_event.proposer = 'FCAA001'": {
			q:       query.MustCompile(`begin_event.proposer = 'FCAA001'`),
			results: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		"end_event.foo <= 5": {
			q:       query.MustCompile(`end_event.foo <= 5`),
			results: []int64{2, 4},
		},
		"end_event.foo >= 100": {
			q:       query.MustCompile(`end_event.foo >= 100`),
			results: []int64{1},
		},
		"block.height > 2 AND end_event.foo <= 8": {
			q:       query.MustCompile(`block.height > 2 AND end_event.foo <= 8`),
			results: []int64{4, 6, 8},
		},
		"end_event.foo > 100": {
			q:       query.MustCompile("end_event.foo > 100"),
			results: []int64{},
		},
		"block.height >= 2 AND end_event.foo < 8": {
			q:       query.MustCompile("block.height >= 2 AND end_event.foo < 8"),
			results: []int64{2, 4, 6},
		},
		"begin_event.proposer CONTAINS 'FFFFFFF'": {
			q:       query.MustCompile(`begin_event.proposer CONTAINS 'FFFFFFF'`),
			results: []int64{},
		},
		"begin_event.proposer CONTAINS 'FCAA001'": {
			q:       query.MustCompile(`begin_event.proposer CONTAINS 'FCAA001'`),
			results: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		"end_event.foo CONTAINS '1'": {
			q:       query.MustCompile("end_event.foo CONTAINS '1'"),
			results: []int64{1, 10},
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			results, err := indexer.Search(context.Background(), tc.q)
			require.NoError(t, err)
			require.Equal(t, tc.results, results)
		})
	}
}
