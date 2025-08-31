CREATE TYPE log_level AS ENUM ('INFO', 'WARN', 'ERROR');


CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    service TEXT NOT NULL,
    level log_level NOT NULL,
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

COMMENT ON TABLE logs IS 'Таблица с логами';

COMMENT ON COLUMN logs.id IS 'Идентификатор лога';

COMMENT ON COLUMN logs.service IS 'Сервис, который прислал лог';

COMMENT ON COLUMN logs.level IS 'Уровень лога';

COMMENT ON COLUMN logs.message IS 'Сообщение к логу';

COMMENT ON COLUMN logs.created_at IS 'Дата создания лога';
