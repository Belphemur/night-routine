-- Update all overridden assignments to have the "Override" decision reason
-- regardless of their current decision_reason value
UPDATE assignments
SET decision_reason = 'Override'
WHERE override = 1;