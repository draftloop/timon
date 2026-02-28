package models

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/draftloop/elm"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
	configdaemon "timon/internal/config/daemon"
	database "timon/internal/daemon/db"
	"timon/internal/log"
	"timon/internal/utils"
)

type WebhookCall struct {
	ID             int64
	URL            string
	Headers        *string
	Cert           *string
	Body           string
	AttemptsLeft   *int
	AttemptTimeout time.Duration
	RetryDelay     time.Duration
	NextTryAt      time.Time
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func makeJsonFromIncident(incident Incident) map[string]interface{} {
	return map[string]interface{}{
		"incident": map[string]interface{}{
			"id":           incident.ID,
			"state":        incident.State,
			"trigger_type": incident.TriggerType,
			"title":        incident.Title,
			"description":  incident.Description,
			"opened_at":    formatTime(&incident.OpenedAt),
			"recovered_at": formatTime(incident.RecoveredAt),
			"relapsed_at":  formatTime(incident.RelapsedAt),
			"resolved_at":  formatTime(incident.ResolvedAt),
		},
	}
}

func FireWebhookEventIncidentOpened(incident Incident, note *string) {
	var datas []map[string]interface{}
	datas = append(datas, makeJsonFromIncident(incident))
	if note != nil {
		datas = append(datas, map[string]interface{}{
			"note": *note,
		})
	}
	fireWebhookEvent("incident.open", datas...)
}

func FireWebhookEventIncidentRelapsed(incident Incident, note string) {
	fireWebhookEvent("incident.relapsed", makeJsonFromIncident(incident), map[string]interface{}{"note": note})
}

func FireWebhookEventIncidentRecovered(incident Incident, note string) {
	fireWebhookEvent("incident.recovered", makeJsonFromIncident(incident), map[string]interface{}{"note": note})
}

func FireWebhookEventIncidentResolved(incident Incident, note *string) {
	var datas []map[string]interface{}
	datas = append(datas, makeJsonFromIncident(incident))
	if note != nil {
		datas = append(datas, map[string]interface{}{
			"note": *note,
		})
	}
	fireWebhookEvent("incident.resolved", datas...)
}

func FireWebhookEventIncidentAnnotated(incident Incident, note string) {
	fireWebhookEvent("incident.annotated", makeJsonFromIncident(incident), map[string]interface{}{"note": note})
}

func FireWebhookEventTimonStarted() {
	fireWebhookEvent("timon.started", map[string]interface{}{})
}

func FireWebhookEventTimonPing() {
	fireWebhookEvent("timon.ping", map[string]interface{}{})
}

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func fireWebhookEvent(event string, datas ...map[string]interface{}) {
	cfg := configdaemon.GetConfig()
	if cfg == nil || len(cfg.Webhooks) == 0 {
		return
	}

	for _, wh := range cfg.Webhooks {
		eventRequested := false
		eventParts := strings.Split(event, ".")
		for _, onEvent := range wh.On {
			if onEvent == event {
				eventRequested = true
				break
			}
			onEventParts := strings.Split(onEvent, ".")
			for i := 0; i < max(len(eventParts), len(onEventParts)); i++ {
				if i < len(onEventParts) && onEventParts[i] == "*" {
					eventRequested = true
					break
				} else if !(i < len(eventParts) && i < len(onEventParts) && eventParts[i] == onEventParts[i]) {
					break
				}
			}
		}
		if !eventRequested {
			continue
		}

		payload := map[string]interface{}{
			"_event":     event,
			"_hostname":  cfg.Daemon.Hostname,
			"_timestamp": formatTime(utils.Ptr(time.Now())),
		}
		for iData := range datas {
			for k, v := range datas[iData] {
				payload[k] = v
			}
		}

		var body string
		if wh.Body == "" {
			j, err := json.Marshal(payload)
			if err != nil {
				_ = log.Daemon.Errorf("headers marshalling failed: %v", err)
				continue
			}
			body = string(j)
		} else {
			tmpl, err := template.New("").Funcs(template.FuncMap{
				"json":      toJSON,
				"urlencode": url.QueryEscape,
			}).Parse(wh.Body)
			if err != nil {
				_ = log.Daemon.Errorf("body template parsing failed: %v", err)
				continue
			}
			var buf bytes.Buffer
			err = tmpl.Execute(&buf, payload)
			if err != nil {
				_ = log.Daemon.Errorf("body template execution failed: %v", err)
				continue
			}
			body = buf.String()
		}

		var headersJSON *string
		if len(wh.Headers) > 0 {
			h, err := json.Marshal(wh.Headers)
			if err != nil {
				_ = log.Daemon.Errorf("headers marshalling failed: %v", err)
				continue
			}
			headersJSON = utils.Ptr(string(h))
		}

		call := WebhookCall{
			URL:            wh.URL,
			Headers:        headersJSON,
			Body:           body,
			AttemptsLeft:   wh.Retry.Attempts,
			AttemptTimeout: wh.Retry.TimeoutDuration,
			RetryDelay:     wh.Retry.DelayDuration,
			NextTryAt:      time.Now(),
		}
		if wh.Cert != "" {
			call.Cert = &wh.Cert
		}

		err := call.Send()
		if err != nil {
			_ = log.Daemon.Errorf("webhook send failed, will retry: %v", err)

			if err := database.GetDB().Save(&call); err != nil {
				_ = log.Daemon.Errorf("webhook queue save error: %v", err)
				continue
			}
		}
	}
}

func (wh *WebhookCall) Send() error {
	db := database.GetDB()

	sendErr := func() error {
		var transport *http.Transport
		if wh.Cert != nil {
			caCert, err := os.ReadFile(*wh.Cert)
			if err != nil {
				return fmt.Errorf("read cert: %w", err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)
			transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}
		} else {
			transport = &http.Transport{}
		}

		client := &http.Client{Transport: transport, Timeout: wh.AttemptTimeout}

		req, err := http.NewRequest(http.MethodPost, wh.URL, strings.NewReader(wh.Body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if wh.Headers != nil {
			var headers map[string]string
			if err := json.Unmarshal([]byte(*wh.Headers), &headers); err != nil {
				return fmt.Errorf("parse headers: %w", err)
			}
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		return nil
	}()
	if sendErr != nil {
		if wh.AttemptsLeft != nil {
			*wh.AttemptsLeft--
		}

		if wh.AttemptsLeft != nil && *wh.AttemptsLeft <= 0 {
			if wh.ID != 0 {
				err := db.Delete(wh)
				if err != nil {
					return fmt.Errorf("delete webhook call: %w; send error: %w", err, sendErr)
				}
			}
		} else {
			err := db.Model(WebhookCall{}).
				Set("attempts_left", wh.AttemptsLeft).
				Set("next_try_at", time.Now().Add(wh.RetryDelay)).
				Where(elm.Eq("id", wh.ID)).
				Update()
			if err != nil {
				return fmt.Errorf("update webhook call: %w; send error: %w", err, sendErr)
			}
		}
		return sendErr
	} else {
		if wh.ID != 0 {
			err := db.Delete(wh)
			if err != nil {
				return fmt.Errorf("delete webhook call: %w", err)
			}
		}
	}

	return nil
}
