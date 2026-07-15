package reporter

import (
	"encoding/json"
	"fmt"

	"github.com/lihongjie0209/redis-analyze/internal/models"
)

// RenderJSON renders the report as pretty-printed JSON.
func RenderJSON(report models.Report) string {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}
