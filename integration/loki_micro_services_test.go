package integration

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/loki/integration/client"
	"github.com/grafana/loki/integration/cluster"
	"github.com/grafana/loki/pkg/storage"
	"github.com/grafana/loki/pkg/util/querylimits"
)

func TestMicroServicesIngestQuery(t *testing.T) {
	clu := cluster.New(nil)
	defer func() {
		assert.NoError(t, clu.Cleanup())
	}()

	// run initially the compactor, indexgateway, and distributor.
	var (
		tCompactor = clu.AddComponent(
			"compactor",
			"-target=compactor",
			"-boltdb.shipper.compactor.compaction-interval=1s",
			"-boltdb.shipper.compactor.retention-delete-delay=1s",
			// By default, a minute is added to the delete request start time. This compensates for that.
			"-boltdb.shipper.compactor.delete-request-cancel-period=-60s",
			"-compactor.deletion-mode=filter-and-delete",
		)
		tIndexGateway = clu.AddComponent(
			"index-gateway",
			"-target=index-gateway",
		)
		tDistributor = clu.AddComponent(
			"distributor",
			"-target=distributor",
		)
	)
	require.NoError(t, clu.Run())

	// then, run only the ingester and query scheduler.
	var (
		tIngester = clu.AddComponent(
			"ingester",
			"-target=ingester",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
		)
		tQueryScheduler = clu.AddComponent(
			"query-scheduler",
			"-target=query-scheduler",
			"-query-scheduler.use-scheduler-ring=false",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
		)
	)
	require.NoError(t, clu.Run())

	// finally, run the query-frontend and querier.
	var (
		tQueryFrontend = clu.AddComponent(
			"query-frontend",
			"-target=query-frontend",
			"-frontend.scheduler-address="+tQueryScheduler.GRPCURL(),
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
			"-common.compactor-address="+tCompactor.HTTPURL(),
			"-querier.per-request-limits-enabled=true",
		)
		_ = clu.AddComponent(
			"querier",
			"-target=querier",
			"-querier.scheduler-address="+tQueryScheduler.GRPCURL(),
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
			"-common.compactor-address="+tCompactor.HTTPURL(),
		)
	)
	require.NoError(t, clu.Run())

	tenantID := randStringRunes()

	now := time.Now()
	cliDistributor := client.New(tenantID, "", tDistributor.HTTPURL())
	cliDistributor.Now = now
	cliIngester := client.New(tenantID, "", tIngester.HTTPURL())
	cliIngester.Now = now
	cliQueryFrontend := client.New(tenantID, "", tQueryFrontend.HTTPURL())
	cliQueryFrontend.Now = now

	t.Run("ingest-logs", func(t *testing.T) {
		// ingest some log lines
		require.NoError(t, cliDistributor.PushLogLineWithTimestamp("lineA", now.Add(-45*time.Minute), map[string]string{"job": "fake"}))
		require.NoError(t, cliDistributor.PushLogLineWithTimestamp("lineB", now.Add(-45*time.Minute), map[string]string{"job": "fake"}))

		require.NoError(t, cliDistributor.PushLogLine("lineC", map[string]string{"job": "fake"}))
		require.NoError(t, cliDistributor.PushLogLine("lineD", map[string]string{"job": "fake"}))
	})

	t.Run("query", func(t *testing.T) {
		resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), `{job="fake"}`)
		require.NoError(t, err)
		assert.Equal(t, "streams", resp.Data.ResultType)

		var lines []string
		for _, stream := range resp.Data.Stream {
			for _, val := range stream.Values {
				lines = append(lines, val[1])
			}
		}
		assert.ElementsMatch(t, []string{"lineA", "lineB", "lineC", "lineD"}, lines)
	})

	t.Run("label-names", func(t *testing.T) {
		resp, err := cliQueryFrontend.LabelNames(context.Background())
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"job"}, resp)
	})

	t.Run("label-values", func(t *testing.T) {
		resp, err := cliQueryFrontend.LabelValues(context.Background(), "job")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"fake"}, resp)
	})

	t.Run("per-request-limits", func(t *testing.T) {
		queryLimitsPolicy := client.InjectHeadersOption(map[string][]string{querylimits.HTTPHeaderQueryLimitsKey: {`{"maxQueryLength": "1m"}`}})
		cliQueryFrontendLimited := client.New(tenantID, "", tQueryFrontend.HTTPURL(), queryLimitsPolicy)
		cliQueryFrontendLimited.Now = now

		_, err := cliQueryFrontendLimited.LabelNames(context.Background())
		require.ErrorContains(t, err, "the query time range exceeds the limit (query length")
	})
}

func TestMicroServicesMultipleBucketSingleProvider(t *testing.T) {
	for name, opt := range map[string]func(c *cluster.Cluster){
		"boltdb-index":    cluster.WithAdditionalBoltDBPeriod,
		"tsdb-index":      cluster.WithAdditionalTSDBPeriod,
		"boltdb-and-tsdb": cluster.WithBoltDBAndTSDBPeriods,
	} {
		t.Run(name, func(t *testing.T) {
			storage.ResetBoltDBIndexClientsWithShipper()
			clu := cluster.New(nil, opt)

			defer func() {
				storage.ResetBoltDBIndexClientsWithShipper()
				assert.NoError(t, clu.Cleanup())
			}()

			// initially, run only compactor and distributor.
			var (
				tCompactor = clu.AddComponent(
					"compactor",
					"-target=compactor",
					"-boltdb.shipper.compactor.compaction-interval=1s",
				)
				tDistributor = clu.AddComponent(
					"distributor",
					"-target=distributor",
				)
			)
			require.NoError(t, clu.Run())

			// then, run only ingester and query-scheduler.
			var (
				tIngester = clu.AddComponent(
					"ingester",
					"-target=ingester",
					"-ingester.flush-on-shutdown=true",
				)
				tQueryScheduler = clu.AddComponent(
					"query-scheduler",
					"-target=query-scheduler",
					"-query-scheduler.use-scheduler-ring=false",
				)
			)
			require.NoError(t, clu.Run())

			// finally, run the query-frontend and querier.
			var (
				tQueryFrontend = clu.AddComponent(
					"query-frontend",
					"-target=query-frontend",
					"-frontend.scheduler-address="+tQueryScheduler.GRPCURL(),
					"-frontend.default-validity=0s",
					"-common.compactor-address="+tCompactor.HTTPURL(),
				)
				tQuerier = clu.AddComponent(
					"querier",
					"-target=querier",
					"-querier.scheduler-address="+tQueryScheduler.GRPCURL(),
					"-common.compactor-address="+tCompactor.HTTPURL(),
				)
			)
			require.NoError(t, clu.Run())

			tenantID := randStringRunes()

			now := time.Now()
			cliDistributor := client.New(tenantID, "", tDistributor.HTTPURL())
			cliDistributor.Now = now
			cliIngester := client.New(tenantID, "", tIngester.HTTPURL())
			cliIngester.Now = now
			cliQueryFrontend := client.New(tenantID, "", tQueryFrontend.HTTPURL())
			cliQueryFrontend.Now = now

			t.Run("ingest-logs", func(t *testing.T) {
				// ingest logs to the previous period
				require.NoError(t, cliDistributor.PushLogLineWithTimestampAndMetadata("lineA", time.Now().Add(-48*time.Hour), map[string]string{"traceID": "123"}, map[string]string{"job": "fake"}))
				require.NoError(t, cliDistributor.PushLogLineWithTimestampAndMetadata("lineB", time.Now().Add(-36*time.Hour), map[string]string{"traceID": "456"}, map[string]string{"job": "fake"}))

				// ingest logs to the current period
				require.NoError(t, cliDistributor.PushLogLineWithMetadata("lineC", map[string]string{"traceID": "789"}, map[string]string{"job": "fake"}))
				require.NoError(t, cliDistributor.PushLogLineWithMetadata("lineD", map[string]string{"traceID": "123"}, map[string]string{"job": "fake"}))
			})

			t.Run("query-lookback-default", func(t *testing.T) {
				// queries ingesters with the default lookback period (3h)
				resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), `{job="fake"}`)
				require.NoError(t, err)
				assert.Equal(t, "streams", resp.Data.ResultType)

				var lines []string
				for _, stream := range resp.Data.Stream {
					for _, val := range stream.Values {
						lines = append(lines, val[1])
					}
				}
				assert.ElementsMatch(t, []string{"lineC", "lineD"}, lines)
			})

			t.Run("flush-logs-and-restart-ingester-querier", func(t *testing.T) {
				// restart ingester which should flush the chunks and index
				require.NoError(t, tIngester.Restart())

				// restart querier and index shipper to sync the index
				storage.ResetBoltDBIndexClientsWithShipper()
				tQuerier.AddFlags("-querier.query-store-only=true")
				require.NoError(t, tQuerier.Restart())
			})

			// Query lines
			t.Run("query again to verify logs being served from storage", func(t *testing.T) {
				resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), `{job="fake"}`)
				require.NoError(t, err)
				assert.Equal(t, "streams", resp.Data.ResultType)

				var lines []string
				for _, stream := range resp.Data.Stream {
					for _, val := range stream.Values {
						lines = append(lines, val[1])
					}
				}

				assert.ElementsMatch(t, []string{"lineA", "lineB", "lineC", "lineD"}, lines)
			})

		})
	}
}

func TestSchedulerRing(t *testing.T) {
	clu := cluster.New(nil)
	defer func() {
		assert.NoError(t, clu.Cleanup())
	}()

	// run initially the compactor, indexgateway, and distributor.
	var (
		tCompactor = clu.AddComponent(
			"compactor",
			"-target=compactor",
			"-boltdb.shipper.compactor.compaction-interval=1s",
			"-boltdb.shipper.compactor.retention-delete-delay=1s",
			// By default, a minute is added to the delete request start time. This compensates for that.
			"-boltdb.shipper.compactor.delete-request-cancel-period=-60s",
			"-compactor.deletion-mode=filter-and-delete",
		)
		tIndexGateway = clu.AddComponent(
			"index-gateway",
			"-target=index-gateway",
		)
		tDistributor = clu.AddComponent(
			"distributor",
			"-target=distributor",
		)
	)
	require.NoError(t, clu.Run())

	// then, run only the ingester and query scheduler.
	var (
		tIngester = clu.AddComponent(
			"ingester",
			"-target=ingester",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
		)
		tQueryScheduler = clu.AddComponent(
			"query-scheduler",
			"-target=query-scheduler",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
			"-query-scheduler.use-scheduler-ring=true",
		)
	)
	require.NoError(t, clu.Run())

	// finally, run the query-frontend and querier.
	var (
		tQueryFrontend = clu.AddComponent(
			"query-frontend",
			"-target=query-frontend",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
			"-common.compactor-address="+tCompactor.HTTPURL(),
			"-querier.per-request-limits-enabled=true",
			"-query-scheduler.use-scheduler-ring=true",
			"-frontend.scheduler-worker-concurrency=5",
		)
		_ = clu.AddComponent(
			"querier",
			"-target=querier",
			"-boltdb.shipper.index-gateway-client.server-address="+tIndexGateway.GRPCURL(),
			"-common.compactor-address="+tCompactor.HTTPURL(),
			"-query-scheduler.use-scheduler-ring=true",
			"-querier.max-concurrent=4",
		)
	)
	require.NoError(t, clu.Run())

	tenantID := randStringRunes()

	now := time.Now()
	cliDistributor := client.New(tenantID, "", tDistributor.HTTPURL())
	cliDistributor.Now = now
	cliIngester := client.New(tenantID, "", tIngester.HTTPURL())
	cliIngester.Now = now
	cliQueryFrontend := client.New(tenantID, "", tQueryFrontend.HTTPURL())
	cliQueryFrontend.Now = now
	cliQueryScheduler := client.New(tenantID, "", tQueryScheduler.HTTPURL())
	cliQueryScheduler.Now = now

	t.Run("verify-scheduler-connections", func(t *testing.T) {
		require.Eventually(t, func() bool {
			// Check metrics to see if query scheduler is connected with query-frontend
			metrics, err := cliQueryScheduler.Metrics()
			require.NoError(t, err)
			return getMetricValue(t, "cortex_query_scheduler_connected_frontend_clients", metrics) == 5
		}, 5*time.Second, 500*time.Millisecond)

		require.Eventually(t, func() bool {
			// Check metrics to see if query scheduler is connected with query-frontend
			metrics, err := cliQueryScheduler.Metrics()
			require.NoError(t, err)
			return getMetricValue(t, "cortex_query_scheduler_connected_querier_clients", metrics) == 4
		}, 5*time.Second, 500*time.Millisecond)
	})

	t.Run("ingest-logs", func(t *testing.T) {
		// ingest some log lines
		require.NoError(t, cliDistributor.PushLogLineWithTimestamp("lineA", now.Add(-45*time.Minute), map[string]string{"job": "fake"}))
		require.NoError(t, cliDistributor.PushLogLineWithTimestamp("lineB", now.Add(-45*time.Minute), map[string]string{"job": "fake"}))

		require.NoError(t, cliDistributor.PushLogLine("lineC", map[string]string{"job": "fake"}))
		require.NoError(t, cliDistributor.PushLogLine("lineD", map[string]string{"job": "fake"}))
	})

	t.Run("query", func(t *testing.T) {
		resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), `{job="fake"}`)
		require.NoError(t, err)
		assert.Equal(t, "streams", resp.Data.ResultType)

		var lines []string
		for _, stream := range resp.Data.Stream {
			for _, val := range stream.Values {
				lines = append(lines, val[1])
			}
		}
		assert.ElementsMatch(t, []string{"lineA", "lineB", "lineC", "lineD"}, lines)
	})
}

func TestNonIndexedMetadata(t *testing.T) {
	clu := cluster.New(nil, cluster.WithAdditionalTSDBPeriod)
	defer func() {
		storage.ResetBoltDBIndexClientsWithShipper()
		assert.NoError(t, clu.Cleanup())
	}()

	// initially, run only compactor and distributor.
	var (
		tCompactor = clu.AddComponent(
			"compactor",
			"-target=compactor",
			"-boltdb.shipper.compactor.compaction-interval=1s",
		)
		tDistributor = clu.AddComponent(
			"distributor",
			"-target=distributor",
		)
	)
	require.NoError(t, clu.Run())

	// then, run only ingester and query-scheduler.
	var (
		tIngester = clu.AddComponent(
			"ingester",
			"-target=ingester",
			"-ingester.flush-on-shutdown=true",
		)
		tQueryScheduler = clu.AddComponent(
			"query-scheduler",
			"-target=query-scheduler",
			"-query-scheduler.use-scheduler-ring=false",
		)
	)
	require.NoError(t, clu.Run())

	// finally, run the query-frontend and querier.
	var (
		tQueryFrontend = clu.AddComponent(
			"query-frontend",
			"-target=query-frontend",
			"-frontend.scheduler-address="+tQueryScheduler.GRPCURL(),
			"-frontend.default-validity=0s",
			"-common.compactor-address="+tCompactor.HTTPURL(),
		)
		tQuerier = clu.AddComponent(
			"querier",
			"-target=querier",
			"-querier.scheduler-address="+tQueryScheduler.GRPCURL(),
			"-common.compactor-address="+tCompactor.HTTPURL(),
		)
	)
	require.NoError(t, clu.Run())

	tenantID := randStringRunes()

	now := time.Now()
	cliDistributor := client.New(tenantID, "", tDistributor.HTTPURL())
	cliDistributor.Now = now
	cliIngester := client.New(tenantID, "", tIngester.HTTPURL())
	cliIngester.Now = now
	cliQueryFrontend := client.New(tenantID, "", tQueryFrontend.HTTPURL())
	cliQueryFrontend.Now = now

	entries := []struct {
		line     string
		ts       time.Time
		metadata map[string]string
		labels   map[string]string
	}{
		{"lineA", now.Add(-1 * time.Hour), map[string]string{"traceID": "123"}, map[string]string{"job": "fake"}},
		{"lineB", now.Add(-30 * time.Minute), map[string]string{"traceID": "456"}, map[string]string{"job": "fake"}},
		{"lineC", now, map[string]string{"traceID": "789"}, map[string]string{"job": "fake"}},
		{"lineD", now, map[string]string{"traceID": "123"}, map[string]string{"job": "fake"}},
	}

	for _, tc := range []struct {
		name  string
		query string

		expectedLines   []string
		expectedStreams []string
	}{
		{
			name:            "no-filter",
			query:           `{job="fake"}`,
			expectedLines:   []string{"lineA", "lineB", "lineC", "lineD"},
			expectedStreams: []string{`{job="fake", traceID="123"}`, `{job="fake", traceID="456"}`, `{job="fake", traceID="789"}`},
		},
		{
			name:            "filter",
			query:           `{job="fake"} | traceID="789"`,
			expectedLines:   []string{"lineC"},
			expectedStreams: []string{`{job="fake", traceID="789"}`},
		},
		{
			name:            "filter-regex-or",
			query:           `{job="fake"} | traceID=~"456|789"`,
			expectedLines:   []string{"lineB", "lineC"},
			expectedStreams: []string{`{job="fake", traceID="456"}`, `{job="fake", traceID="789"}`},
		},
		{
			name:            "filter-regex-contains",
			query:           `{job="fake"} | traceID=~".*5.*"`,
			expectedLines:   []string{"lineB"},
			expectedStreams: []string{`{job="fake", traceID="456"}`},
		},
		{
			name:            "filter-regex-complex",
			query:           `{job="fake"} | traceID=~"^[0-9]2.*"`,
			expectedLines:   []string{"lineA", "lineD"},
			expectedStreams: []string{`{job="fake", traceID="123"}`},
		},
		// TODO: Add test with KEEP clause
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("ingest-logs", func(t *testing.T) {
				for _, entry := range entries {
					require.NoError(t, cliDistributor.PushLogLineWithTimestampAndMetadata(entry.line, entry.ts, entry.metadata, entry.labels))
				}
			})

			t.Run("query-ingesters", func(t *testing.T) {
				// queries ingesters with the default lookback period (3h)
				resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), tc.query)
				require.NoError(t, err)
				assert.Equal(t, "streams", resp.Data.ResultType)

				var lines []string
				var streams []string
				for _, stream := range resp.Data.Stream {
					streams = append(streams, labels.FromMap(stream.Stream).String())
					for _, val := range stream.Values {
						lines = append(lines, val[1])
					}
				}
				assert.ElementsMatch(t, tc.expectedLines, lines)
				assert.ElementsMatch(t, tc.expectedStreams, streams)
			})

			t.Run("flush-logs-and-restart-ingester-querier", func(t *testing.T) {
				// restart ingester which should flush the chunks and index
				require.NoError(t, tIngester.Restart())

				// restart querier and index shipper to sync the index
				storage.ResetBoltDBIndexClientsWithShipper()
				tQuerier.AddFlags("-querier.query-store-only=true")
				require.NoError(t, tQuerier.Restart())
			})

			// Query lines
			t.Run("query-storage", func(t *testing.T) {
				resp, err := cliQueryFrontend.RunRangeQuery(context.Background(), tc.query)
				require.NoError(t, err)
				assert.Equal(t, "streams", resp.Data.ResultType)

				var lines []string
				var streams []string
				for _, stream := range resp.Data.Stream {
					streams = append(streams, labels.FromMap(stream.Stream).String())
					for _, val := range stream.Values {
						lines = append(lines, val[1])
					}
				}

				assert.ElementsMatch(t, tc.expectedLines, lines)
				assert.ElementsMatch(t, tc.expectedStreams, streams)
			})
		})
	}
}
