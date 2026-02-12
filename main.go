package main

import (
	"context"
	"log/slog"
	"os"
	"fmt"
	"sync/atomic"

	"github.com/brody192/locomotive/internal/config"
	"github.com/brody192/locomotive/internal/errgroup"
	"github.com/brody192/locomotive/internal/logger"
	"github.com/brody192/locomotive/internal/railway"
	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
)

func main() {
	fmt.Println("🚂 Locomotive environment variables:")
	envVars := []string{
		"LOCOMOTIVE_ENVIRONMENT_ID",
		"LOCOMOTIVE_SERVICE_IDS",
		"LOCOMOTIVE_MIN_SEVERITY",
		"LOCOMOTIVE_ENABLE_DEPLOY_LOGS",
		"LOCOMOTIVE_ENABLE_HTTP_LOGS",
		"LOCOMOTIVE_WEBHOOK_MODE",
		"RAILWAY_API_KEY",
		"SENTRY_DSN",
	}
	for _, key := range envVars {
		val := os.Getenv(key)
		if val == "" {
			fmt.Printf("%s = <not set>\n", key)
		} else {
			fmt.Printf("%s = %s\n", key, val)
		}
	}
	logger.Stdout.Info("Preparing the locomotive for departure...")

	gqlClient, err := railway.NewClient(&railway.GraphQLClient{
		AuthToken:           config.Global.RailwayApiKey,
		BaseURL:             "https://backboard.railway.app/graphql/v2",
		BaseSubscriptionURL: "wss://backboard.railway.app/graphql/internal",
	})
	if err != nil {
		logger.Stderr.Error("error creating graphql client", logger.ErrAttr(err))
		os.Exit(1)
	}

	allServicesExist, foundServices, missingServices, err := railway.VerifyAllServicesExistWithinEnvironment(gqlClient, config.Global.ServiceIds, config.Global.EnvironmentId)
	if err != nil {
		logger.Stderr.Error("error verifying if services exist within the environment", logger.ErrAttr(err))
		os.Exit(1)
	}

	if !allServicesExist {
		logger.Stderr.Error("all services must exist within the environment set by the LOCOMOTIVE_ENVIRONMENT_ID variable",
			slog.Any("missing_service_ids", missingServices),
			slog.Any("configured_service_ids", config.Global.ServiceIds),
			slog.Any("found_service_ids", foundServices),
			slog.Any("environment_id", config.Global.EnvironmentId),
		)

		os.Exit(1)
	}
	if !config.Global.MinSeverity.IsValid() {
		logger.Stderr.Error("invalid MIN_SEVERITY value",
			slog.String("min_severity", string(config.Global.MinSeverity)),
		)
		os.Exit(1)
	}

	logger.Stdout.Info("The locomotive is ready to depart...",
		slog.String("webhook_url_host", config.Global.WebhookUrl.Host),
		slog.Any("service_ids", config.Global.ServiceIds),
		slog.Any("environment_id", config.Global.EnvironmentId),
		slog.Any("webhook_mode", config.Global.WebhookMode),
		slog.Bool("enable_http_logs", config.Global.EnableHttpLogs),
		slog.Bool("enable_deploy_logs", config.Global.EnableDeployLogs),
		slog.String("min_severity", string(config.Global.MinSeverity)),
	)
	fmt.Printf("severity level: %s\n", config.Global.MinSeverity)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serviceLogTrack := make(chan []environment_logs.EnvironmentLogWithMetadata)
	httpLogTrack := make(chan []http_logs.DeploymentHttpLogWithMetadata)

	deployLogsProcessed := atomic.Int64{}
	httpLogsProcessed := atomic.Int64{}

	reportStatusAsync(&deployLogsProcessed, &httpLogsProcessed)

	handleDeployLogsAsync(ctx, &deployLogsProcessed, serviceLogTrack, config.Global.MinSeverity)
	handleHttpLogsAsync(ctx, &httpLogsProcessed, httpLogTrack)

	errGroup := errgroup.NewErrGroup()

	errGroup.Go(func() error {
		if !config.Global.EnableDeployLogs {
			logger.Stdout.Info("Deploy log transport is disabled. To enable it, set LOCOMOTIVE_ENABLE_DEPLOY_LOGS=true")
			return nil
		}

		return startStreamingDeployLogs(ctx, gqlClient, serviceLogTrack, config.Global.EnvironmentId, config.Global.ServiceIds)
	})

	errGroup.Go(func() error {
		if !config.Global.EnableHttpLogs {
			logger.Stdout.Info("HTTP log transport is disabled. To enable it, set LOCOMOTIVE_ENABLE_HTTP_LOGS=true")
			return nil
		}

		return startStreamingHttpLogs(ctx, gqlClient, httpLogTrack, config.Global.EnvironmentId, config.Global.ServiceIds)
	})

	logger.Stdout.Info("The locomotive is waiting for cargo...")

	if err := errGroup.Wait(); err != nil {
		logger.Stderr.Error("error returned from subscription(s)", logger.ErrAttr(err))
		os.Exit(1)
	}
}
