-- Replace 'Consecutive Avoidance' decision reason with 'Total Count' since
-- ConsecutiveAvoidance was removed from the fairness algorithm.
UPDATE assignments
SET decision_reason = 'Total Count'
WHERE decision_reason = 'Consecutive Avoidance';
