package v1alpha1

import (
	"testing"
)

func TestRegistrationAuth_GetDriver(t *testing.T) {
	tests := []struct {
		name string
		ra   *RegistrationAuth
		want string
	}{
		{
			name: "nil receiver",
			ra:   nil,
			want: CSRRegistrationDriver,
		},
		{
			name: "explicit CSR driver",
			ra:   &RegistrationAuth{Driver: CSRRegistrationDriver},
			want: CSRRegistrationDriver,
		},
		{
			name: "explicit AWS IRSA driver",
			ra:   &RegistrationAuth{Driver: AWSIRSARegistrationDriver},
			want: AWSIRSARegistrationDriver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ra.GetDriver()
			if got != tt.want {
				t.Errorf("GetDriver() = %q, want %q", got, tt.want)
			}
		})
	}
}
