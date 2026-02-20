package reconstruct_sentry

import (
	"bytes"
	// "cmp"
	// "fmt"
	// "strconv"
	"time"

	"github.com/tidwall/sjson"

	// "github.com/brody192/locomotive/internal/logline/reconstructor"
	// "github.com/brody192/locomotive/internal/logline/reconstructor/reconstruct_sentry/sentry_attribute"
	"github.com/brody192/locomotive/internal/railway/subscribe/environment_logs"
	"github.com/brody192/locomotive/internal/util"
)

func EnvironmentLogsEnvelope(logs []environment_logs.EnvironmentLogWithMetadata) ([]byte, error) {
	jsonObject := bytes.Buffer{}

	// len(logs) == 1
	log := logs[0]
	eventID := generateRandomHexString()
	timestamp := log.Log.Timestamp.Format(time.RFC3339Nano)

	// --- derive server_name ---
	serverName := log.Metadata["service_name"]
	if serverName == "" {
		serverName = log.Metadata["service_id"]
	}

	// --- derive environment ---
	environmentName := log.Metadata["environment_name"]
	if environmentName == "" {
		environmentName = "production"
	}

	// --- Line One ---
	firstLineData := LineOne
	firstLineData, _ = sjson.Set(firstLineData, "event_id", eventID)
	firstLineData, _ = sjson.Set(firstLineData, "sent_at", timestamp)
	jsonObject.WriteString(firstLineData)
	jsonObject.WriteByte('\n')

	// --- Line Two ---
	secondLineData := LineTwo
	jsonObject.WriteString(secondLineData)
	jsonObject.WriteByte('\n')

	// --- Line Three (Event) ---
	thirdLineData := LineThree
	thirdLineData, _ = sjson.Set(thirdLineData, "event_id", eventID)
	thirdLineData, _ = sjson.Set(thirdLineData, "timestamp", timestamp)
	thirdLineData, _ = sjson.Set(thirdLineData, "platform", "go")
	thirdLineData, _ = sjson.Set(thirdLineData, "level", normalizeLevel(log.Log.Severity))
	thirdLineData, _ = sjson.Set(thirdLineData, "message", util.StripAnsi(log.Log.Message))
	thirdLineData, _ = sjson.Set(thirdLineData, "server_name", serverName)
	thirdLineData, _ = sjson.Set(thirdLineData, "environment", environmentName)

	// --- Relevant tags (IDs only) ---
	if v := log.Metadata["project_id"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.project_id", v)
	}
	if v := log.Metadata["environment_id"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.environment_id", v)
	}
	if v := log.Metadata["service_id"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.service_id", v)

		// Add service_id to fingerprint for grouping isolation
		thirdLineData, _ = sjson.Set(thirdLineData, "fingerprint.0", "{{ default }}")
		thirdLineData, _ = sjson.Set(thirdLineData, "fingerprint.1", v)
	}
	if v := log.Metadata["deployment_id"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.deployment_id", v)
	}
	if v := log.Metadata["deployment_instance_id"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.deployment_instance_id", v)
	}
	if v := log.Metadata["log_type"]; v != "" {
		thirdLineData, _ = sjson.Set(thirdLineData, "tags.log_type", v)
	}


	jsonObject.WriteString(thirdLineData)
	jsonObject.WriteByte('\n')

	return jsonObject.Bytes(), nil
}
