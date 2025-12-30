-- Create assignment_details table to store fairness algorithm calculations
CREATE TABLE IF NOT EXISTS assignment_details (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    assignment_id INTEGER NOT NULL UNIQUE,
    calculation_date TEXT NOT NULL,
    parent_a_name TEXT NOT NULL,
    parent_a_total_count INTEGER NOT NULL,
    parent_a_last_30_days INTEGER NOT NULL,
    parent_b_name TEXT NOT NULL,
    parent_b_total_count INTEGER NOT NULL,
    parent_b_last_30_days INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE
);

-- Create index on assignment_id for faster lookups (unique constraint already provides this)
CREATE INDEX IF NOT EXISTS idx_assignment_details_assignment_id ON assignment_details(assignment_id);
