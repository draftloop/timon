package migrations

var M001Init = Migration{
	Version: 1,
	Up: `
-------------------------------------------------- table contracts --------------------------------------------------
CREATE TABLE contracts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT,
    type TEXT NOT NULL CHECK (
        type IN ('probe', 'job')
    ),
	is_stale BOOLEAN NOT NULL,

    last_report_id INTEGER,

	rule_stale INTEGER,
	rule_stale_at DATETIME,
	rule_stale_incident INTEGER,
	rule_stale_incident_at DATETIME,

	FOREIGN KEY (last_report_id) REFERENCES reports(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX ux_contracts_code ON contracts(code);
CREATE INDEX idx_contracts_type ON contracts(type);
CREATE INDEX idx_contracts_rule_stale_at ON contracts(rule_stale_at);
CREATE INDEX idx_contracts_rule_stale_incident_at ON contracts(rule_stale_incident_at);

CREATE INDEX idx_contracts_last_report_id ON contracts(last_report_id);


-------------------------------------------------- table reports --------------------------------------------------
CREATE TABLE reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contract_id INTEGER NOT NULL,
    health TEXT NOT NULL CHECK (
        health IN ('healthy', 'warning', 'critical', 'unknown')
    ),

    probe_pushed_at DATETIME,
    probe_comment TEXT,

    uid TEXT,
    job_started_at DATETIME,
    job_start_comment TEXT,
    job_ended_at DATETIME,
    job_end_comment TEXT,

	rule_stale INTEGER,
	rule_stale_incident INTEGER,
	rule_job_overtime_incident INTEGER,
	rule_job_overtime_incident_at DATETIME,
	rule_job_overlap_incident BOOLEAN,

    FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE CASCADE
);

CREATE INDEX idx_reports_contract_id ON reports(contract_id);
CREATE INDEX idx_reports_uid ON reports(uid);
CREATE INDEX idx_reports_health ON reports(health);
CREATE INDEX idx_reports_rule_job_overtime_incident_at ON reports(rule_job_overtime_incident_at);


-------------------------------------------------- table report_job_steps --------------------------------------------------
CREATE TABLE report_job_steps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    report_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    health TEXT NOT NULL CHECK (
        health IN ('healthy', 'warning', 'critical')
    ),
    at DATETIME NOT NULL,

    FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE
);

CREATE INDEX idx_report_job_steps_report_id ON report_job_steps(report_id);
CREATE INDEX idx_report_job_steps_health ON report_job_steps(health);


-------------------------------------------------- table incidents --------------------------------------------------
CREATE TABLE incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    state TEXT NOT NULL CHECK (
        state IN ('open', 'recovered', 'relapsed', 'resolved')
    ),
    trigger_type TEXT NOT NULL CHECK (
        trigger_type IN ('manual', 'critical', 'job_overlap', 'stale', 'job_overtime')
    ),
    title TEXT NOT NULL,
    description TEXT,

    contract_id INTEGER,

    opened_at DATETIME NOT NULL,
    recovered_at DATETIME,
    relapsed_at DATETIME,
    resolved_at DATETIME,

    FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE CASCADE,

    CHECK (
        (state = 'open' AND recovered_at IS NULL AND relapsed_at IS NULL AND resolved_at IS NULL)
        OR
        (state = 'recovered' AND recovered_at IS NOT NULL AND relapsed_at IS NULL AND resolved_at IS NULL)
        OR
        (state = 'relapsed' AND recovered_at IS NOT NULL AND relapsed_at IS NOT NULL AND resolved_at IS NULL)
        OR
        (state = 'resolved' AND resolved_at IS NOT NULL)
    )
);

CREATE INDEX idx_incidents_state ON incidents(state);

CREATE INDEX idx_incidents_contract_id ON incidents(contract_id);


-------------------------------------------------- table incident_events --------------------------------------------------
CREATE TABLE incident_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK (
        type IN ('incident_opened', 'incident_recovered', 'incident_relapsed', 'incident_resolved', 'annotation')
    ),

    incident_id INTEGER NOT NULL,
    report_id INTEGER,

    note TEXT NOT NULL,
    is_system BOOLEAN NOT NULL,
    at DATETIME NOT NULL,

    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
    FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE SET NULL
);

CREATE INDEX idx_incident_events_incident_id ON incident_events(incident_id);
CREATE INDEX idx_incident_events_report_id ON incident_events(report_id);


-------------------------------------------------- table webhook_calls --------------------------------------------------
CREATE TABLE webhook_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    headers TEXT,
    cert TEXT,
    body TEXT NOT NULL,
	attempts_left INTEGER,
	attempt_timeout INTEGER NOT NULL,
	retry_delay INTEGER NOT NULL,
    next_try_at DATETIME NOT NULL
);

CREATE INDEX idx_webhook_calls_next_try_at ON webhook_calls(next_try_at);


PRAGMA foreign_keys = ON;
`,
}
