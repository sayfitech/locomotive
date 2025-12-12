package webhook

import (
	"fmt"

	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
	"github.com/brody192/locomotive/internal/webhook/generic"
)

func SendDeployLogsWebhook(logs []environment_logs.EnvironmentLogWithMetadata) (serializedLogs []byte, err error) {
	if serializedLogs, err := generic.SendWebhookForDeployLogs(logs, client); err != nil {
		return serializedLogs, fmt.Errorf("failed to send webhook for deploy logs: %w", err)
	}

	return nil, nil
}

func SendHttpLogsWebhook(logs []http_logs.DeploymentHttpLogWithMetadata) (serializedLogs []byte, err error) {
	if serializedLogs, err := generic.SendWebhookForHttpLogs(logs, client); err != nil {
		return serializedLogs, fmt.Errorf("failed to send webhook for http logs: %w", err)
	}

	return nil, nil
}
