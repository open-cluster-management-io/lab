package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFleetConfig_IsUnjoined(t *testing.T) {
	// Create base time for testing
	baseTime := time.Now()
	joinedTime := metav1.NewTime(baseTime)
	unjoinedTime := metav1.NewTime(baseTime.Add(time.Hour)) // 1 hour later

	tests := []struct {
		name        string
		fleetConfig *FleetConfig
		spoke       Spoke
		joinedSpoke JoinedSpoke
		want        bool
	}{
		{
			name: "successfully joined and unjoined - should return true",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime,
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: unjoinedTime,
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        true,
		},
		{
			name: "joined but not unjoined - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime,
							},
						},
						// No unjoin condition
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        false,
		},
		{
			name: "both conditions exist but unjoin is False - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime,
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionFalse,
								LastTransitionTime: unjoinedTime,
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        false,
		},
		{
			name: "both conditions exist but join is False - should return true",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionFalse,
								LastTransitionTime: metav1.NewTime(unjoinedTime.Add(time.Hour)), // 1 hour after unjoin, failed re-join
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: unjoinedTime,
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        true,
		},
		{
			name: "both conditions exist but unjoin time is before join time - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime,
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(baseTime.Add(-time.Hour)), // 1 hour earlier
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        false,
		},
		{
			name: "both conditions exist with same time - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime,
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: joinedTime, // Same time
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        false,
		},
		{
			name: "no conditions exist - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        false,
		},
		{
			name: "conditions with Unknown status - should return false",
			fleetConfig: &FleetConfig{
				Status: FleetConfigStatus{
					Conditions: []Condition{
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-joined",
								Status:             metav1.ConditionUnknown,
								LastTransitionTime: joinedTime,
							},
						},
						{
							Condition: metav1.Condition{
								Type:               "spoke-cluster-testcluster-unjoined",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: unjoinedTime,
							},
						},
					},
				},
			},
			spoke:       Spoke{Name: "testcluster"},
			joinedSpoke: JoinedSpoke{Name: "testcluster"},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fleetConfig.IsUnjoined(tt.spoke, tt.joinedSpoke)
			if got != tt.want {
				t.Errorf("FleetConfig.IsUnjoined() = %v, want %v", got, tt.want)
			}
		})
	}
}
