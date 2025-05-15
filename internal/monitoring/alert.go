package monitoring

import (
	"github.com/rs/zerolog/log"
)

// MockAlert sends a mock alert (logs for now)
func MockAlert(message string, labels map[string]string) {
	log.Error().
		Str("alert", message).
		Fields(labels).
		Msg("ALERT: Provisioning issue detected")
}
