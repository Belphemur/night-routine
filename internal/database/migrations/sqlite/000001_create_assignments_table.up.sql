CREATE TABLE IF NOT EXISTS assignments (
id INTEGER PRIMARY KEY AUTOINCREMENT,
parent_name TEXT NOT NULL,
assignment_date TEXT NOT NULL,
created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
