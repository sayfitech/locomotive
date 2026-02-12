package main

import (
	"context"
	"fmt"
	"strings"
	"log/slog"
	"sync/atomic"

	"github.com/brody192/locomotive/internal/logger"
	"github.com/brody192/locomotive/internal/config"
	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
	"github.com/brody192/locomotive/internal/webhook"
)

func handleDeployLogsAsync(ctx context.Context, deployLogsProcessed *atomic.Int64, serviceLogTrack chan []environment_logs.EnvironmentLogWithMetadata, min_severity config.SeverityLevel) {
	go func() {
		fmt.Printf("level: %s\n", min_severity)
		for {
			select {
			case <-ctx.Done():
				return
			case logs := <-serviceLogTrack:
				filteredLogs := make([]environment_logs.EnvironmentLogWithMetadata, 0, len(logs))

				for _, logEntry := range logs {
					logSeverity := config.SeverityLevel(
						strings.ToLower(strings.TrimSpace(logEntry.Log.Severity)),
					)

					// Compare ranks
					if logSeverity.Rank() >= min_severity.Rank() {
						filteredLogs = append(filteredLogs, logEntry)
					}
				}

				if len(filteredLogs) == 0 {
					continue
				}

				if serializedLogs, err := webhook.SendDeployLogsWebhook(filteredLogs); err != nil {
					attrs := []any{logger.ErrAttr(err)}

					if serializedLogs != nil {
						attrs = append(attrs, slog.String("serialized_logs", string(serializedLogs)))
					}

					logger.Stderr.Error("error sending deploy logs webhook(s)", attrs...)

					continue
				}

				deployLogsProcessed.Add(int64(len(logs)))
			}
		}
	}()
}

func handleHttpLogsAsync(ctx context.Context, httpLogsProcessed *atomic.Int64, httpLogTrack chan []http_logs.DeploymentHttpLogWithMetadata) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case logs := <-httpLogTrack:
				if serializedLogs, err := webhook.SendHttpLogsWebhook(logs); err != nil {
					attrs := []any{logger.ErrAttr(err)}

					if serializedLogs != nil {
						attrs = append(attrs, slog.String("serialized_logs", string(serializedLogs)))
					}

					logger.Stderr.Error("error sending http logs webhook(s)", attrs...)

					continue
				}

				httpLogsProcessed.Add(int64(len(logs)))
			}
		}
	}()
}
