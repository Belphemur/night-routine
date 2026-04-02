package fairness

// CaregiverType represents who handled a night routine assignment.
type CaregiverType string

const (
	// CaregiverTypeParent marks a parent assignment.
	CaregiverTypeParent CaregiverType = "parent"
	// CaregiverTypeBabysitter marks a babysitter assignment.
	CaregiverTypeBabysitter CaregiverType = "babysitter"
)

// String returns the string representation of the caregiver type.
func (c CaregiverType) String() string {
	return string(c)
}
