-- Создание таблицы для хранения метрик

CREATE TABLE metrics (
    id TEXT PRIMARY KEY,
    type VARCHAR(10) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT valid_type
        CHECK (type IN ('gauge', 'counter')),

    CONSTRAINT valid_metric_value
        CHECK (
            (type = 'gauge' AND value IS NOT NULL AND delta IS NULL)
            OR
            (type = 'counter' AND delta IS NOT NULL AND value IS NULL)
        )
);
