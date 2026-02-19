package provider

import (
	"errors"
	"strings"
	"time"

	"github.com/tpretz/go-zabbix-api"
)

// retryZabbixTransient retries operations that are known to occasionally fail due to
// transient Zabbix DB/API issues (e.g. DB deadlocks surfaced as DBEXECUTE_ERROR).
func retryZabbixTransient(op func() error) error {
	// Conservative: small number of attempts with exponential-ish backoff.
	backoffs := []time.Duration{0, 500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 4 * time.Second}

	var last error
	for i, d := range backoffs {
		if d > 0 {
			time.Sleep(d)
		}
		err := op()
		if err == nil {
			return nil
		}
		last = err

		if !isTransientZabbixError(err) {
			return err
		}

		// If this was the last attempt, break and return the error.
		if i == len(backoffs)-1 {
			break
		}
	}
	return last
}

func isTransientZabbixError(err error) bool {
	var ze *zabbix.Error
	if errors.As(err, &ze) {
		// Zabbix often reports transient DB issues as code=-32603 with data like DBEXECUTE_ERROR.
		if ze.Code == -32603 {
			if strings.Contains(ze.Data, "DBEXECUTE_ERROR") {
				return true
			}
			// Be slightly broader: sometimes deadlocks/lock timeouts surface with other data strings.
			if strings.Contains(strings.ToLower(ze.Data), "deadlock") || strings.Contains(strings.ToLower(ze.Data), "lock") {
				return true
			}
		}
	}

	// Fallback: string match (covers wrapped errors)
	msg := err.Error()
	if strings.Contains(msg, "DBEXECUTE_ERROR") {
		return true
	}
	if strings.Contains(strings.ToLower(msg), "deadlock") {
		return true
	}
	return false
}
