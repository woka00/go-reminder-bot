ALTER TABLE reminders ADD COLUMN recurrence      VARCHAR(50);
ALTER TABLE reminders ADD COLUMN recurrence_day  VARCHAR(20);
ALTER TABLE reminders ADD COLUMN parent_id       BIGINT REFERENCES reminders(id);
