CREATE TABLE reminders (
    id           BIGSERIAL PRIMARY KEY,
    chat_id      BIGINT NOT NULL,
    task         TEXT NOT NULL,
    remind_at    TIMESTAMPTZ NOT NULL,
    done         BOOLEAN DEFAULT FALSE,
    done_at      TIMESTAMPTZ,
    cancelled    BOOLEAN DEFAULT FALSE,
    cancelled_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reminders_remind_at
    ON reminders(remind_at)
    WHERE done = FALSE AND cancelled = FALSE;
