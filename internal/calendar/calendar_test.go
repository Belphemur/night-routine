package calendar

import (
	"testing"

	"github.com/belphemur/night-routine/internal/constants"
	"github.com/belphemur/night-routine/internal/fairness"
	"github.com/belphemur/night-routine/internal/fairness/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		want       string
	}{
		{
			name: "parent assignment uses Parent field",
			assignment: &scheduler.Assignment{
				Parent:        "Alice",
				CaregiverType: fairness.CaregiverTypeParent,
			},
			want: "Alice",
		},
		{
			name: "babysitter assignment uses Parent name",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "Dawn",
		},
		{
			name: "babysitter with parent name",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "Dawn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, displayName(tt.assignment))
		})
	}
}

func TestFormatEventSummary(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		want       string
	}{
		{
			name: "parent assignment",
			assignment: &scheduler.Assignment{
				Parent:        "Alice",
				CaregiverType: fairness.CaregiverTypeParent,
			},
			want: "[Alice] \U0001f303\U0001f476Routine",
		},
		{
			name: "babysitter assignment",
			assignment: &scheduler.Assignment{
				Parent:        "Dawn",
				CaregiverType: fairness.CaregiverTypeBabysitter,
			},
			want: "[Dawn] \U0001f303\U0001f476Routine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatEventSummary(tt.assignment))
		})
	}
}

func TestFormatEventDescription(t *testing.T) {
	tests := []struct {
		name       string
		assignment *scheduler.Assignment
		wantPrefix string
		wantSuffix string
	}{
		{
			name: "parent assignment says assigned to",
			assignment: &scheduler.Assignment{
				Parent:         "Alice",
				CaregiverType:  fairness.CaregiverTypeParent,
				DecisionReason: fairness.DecisionReasonTotalCount,
			},
			wantPrefix: "Night routine duty assigned to Alice",
			wantSuffix: "[" + constants.NightRoutineIdentifier + "]",
		},
		{
			name: "babysitter assignment says handled by babysitter",
			assignment: &scheduler.Assignment{
				Parent:         "Dawn",
				CaregiverType:  fairness.CaregiverTypeBabysitter,
				DecisionReason: fairness.DecisionReasonOverride,
			},
			wantPrefix: "Night routine handled by babysitter Dawn",
			wantSuffix: "[" + constants.NightRoutineIdentifier + "]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := formatEventDescription(tt.assignment)
			assert.Contains(t, desc, tt.wantPrefix)
			assert.Contains(t, desc, tt.wantSuffix)
		})
	}
}
