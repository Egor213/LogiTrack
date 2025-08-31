-- +migrate StatementBeginTx false
CREATE INDEX idx_logs_service ON logs(service);
CREATE INDEX idx_logs_level ON logs(level);
CREATE INDEX idx_logs_created_at ON logs(created_at);
CREATE INDEX idx_logs_service_level_created_at ON logs(service, level, created_at);
