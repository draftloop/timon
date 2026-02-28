package enums

type IncidentState string

const (
	IncidentStateOpen      IncidentState = "open"
	IncidentStateRecovered IncidentState = "recovered"
	IncidentStateRelapsed  IncidentState = "relapsed"
	IncidentStateResolved  IncidentState = "resolved"
)

type IncidentTriggerType string

const (
	IncidentTriggerTypeManual      IncidentTriggerType = "manual"
	IncidentTriggerTypeCritical    IncidentTriggerType = "critical"
	IncidentTriggerTypeJobOverlap  IncidentTriggerType = "job_overlap"
	IncidentTriggerTypeStale       IncidentTriggerType = "stale"
	IncidentTriggerTypeJobOvertime IncidentTriggerType = "job_overtime"
)

type IncidentEventType string

const (
	IncidentEventTypeIncidentOpened    IncidentEventType = "incident_opened"
	IncidentEventTypeIncidentRecovered IncidentEventType = "incident_recovered"
	IncidentEventTypeIncidentRelapsed  IncidentEventType = "incident_relapsed"
	IncidentEventTypeIncidentResolved  IncidentEventType = "incident_resolved"
	IncidentEventTypeAnnotation        IncidentEventType = "annotation"
)
