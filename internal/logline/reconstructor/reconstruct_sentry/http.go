package reconstruct_sentry

import (
	"bytes"
	"fmt"
	"time"

	"github.com/brody192/locomotive/internal/logline/reconstructor/reconstruct_sentry/sentry_attribute"
	"github.com/brody192/locomotive/internal/railway/subscribe/http_logs"
	"github.com/tidwall/sjson"
)
func HttpLogsEnvelope(logs []http_logs.DeploymentHttpLogWithMetadata) ([]byte, error) {
	jsonObject := bytes.Buffer{}

	for _, log := range logs {
		eventID := generateRandomHexString()
		timestamp := time.Now().Format(time.RFC3339Nano)

		// --- Line One ---
		firstLineData := LineOne
		firstLineData, _ = sjson.Set(firstLineData, "event_id", eventID)
		firstLineData, _ = sjson.Set(firstLineData, "sent_at", timestamp)
		jsonObject.WriteString(firstLineData)
		jsonObject.WriteByte('\n')

		// --- Line Two ---
		secondLineData := LineTwo
		secondLineData, _ = sjson.Set(secondLineData, "item_count", 1)
		jsonObject.WriteString(secondLineData)
		jsonObject.WriteByte('\n')

		// --- Line Three ---
		thirdLineData := LineThree
		thirdLineData, _ = sjson.Set(thirdLineData, "event_id", eventID)
		thirdLineData, _ = sjson.Set(thirdLineData, "timestamp", timestamp)

		item := Item
		item, _ = sjson.Set(item, "timestamp", log.Timestamp.Format(time.RFC3339Nano))
		item, _ = sjson.Set(item, "trace_id", generateRandomHexString())

		level, severityNumber := getLevelFromStatusCode(log.StatusCode)
		item, _ = sjson.Set(item, "level", level)
		item, _ = sjson.Set(item, "severity_number", severityNumber)
		item, _ = sjson.Set(item, "body", log.Path)
		item, _ = sjson.Set(item, "attributes.level", sentry_attribute.StringValue(level))

		for key, value := range jsonBytesToSentryAttributes(log.Log) {
			item, _ = sjson.Set(item, fmt.Sprintf("attributes.%s", key), value)
		}

		for key, value := range log.Metadata {
			item, _ = sjson.Set(item, fmt.Sprintf("attributes._metadata__%s", key), sentry_attribute.StringValue(value))
		}

		thirdLineData, _ = sjson.SetRaw(thirdLineData, "items.0", item)

		jsonObject.WriteString(thirdLineData)
		jsonObject.WriteByte('\n')
	}

	return jsonObject.Bytes(), nil
}