CREATE TABLE IF NOT EXISTS notification_channels (
    id TEXT PRIMARY KEY,
    resource_id TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    expiration TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create trigger to update the updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_notification_channels_updated_at
AFTER UPDATE ON notification_channels
FOR EACH ROW
BEGIN
    UPDATE notification_channels SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;