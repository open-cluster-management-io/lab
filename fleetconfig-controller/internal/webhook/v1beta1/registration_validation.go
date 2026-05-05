package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/klusterletvalues"
)

func validateHubRegistrationAuth(h *v1beta1.Hub) field.ErrorList {
	var errs field.ErrorList
	base := field.NewPath("spec").Child("registrationAuth")
	ra := h.Spec.RegistrationAuth

	if ra.Driver == v1beta1.AWSIRSARegistrationDriver && ra.GRPC != nil {
		errs = append(errs, field.Invalid(base.Child("grpc"), ra.GRPC, "grpc must be unset when registrationAuth.driver is awsirsa"))
	}

	if ra.GRPC != nil && ra.GRPC.EndpointType != "" {
		et := ra.GRPC.EndpointType
		if !strings.EqualFold(et, v1beta1.GRPCEndpointTypeHostname) && !strings.EqualFold(et, v1beta1.GRPCEndpointTypeLoadBalancer) {
			errs = append(errs, field.NotSupported(base.Child("grpc").Child("endpointType"), et, []string{v1beta1.GRPCEndpointTypeHostname, v1beta1.GRPCEndpointTypeLoadBalancer}))
		}
	}

	return errs
}

func validateSpokeKlusterletAddon(spoke *v1beta1.Spoke) field.ErrorList {
	var errs field.ErrorList
	auth := strings.ToLower(spoke.Spec.Klusterlet.AddonKubeClientRegistrationAuth)
	if auth == "" {
		return errs
	}
	if !slices.Contains([]string{v1beta1.AddonKubeClientRegistrationAuthCSR, v1beta1.AddonKubeClientRegistrationAuthToken}, auth) {
		errs = append(errs, field.NotSupported(field.NewPath("spec").Child("klusterlet").Child("addonKubeClientRegistrationAuth"), spoke.Spec.Klusterlet.AddonKubeClientRegistrationAuth,
			[]string{v1beta1.AddonKubeClientRegistrationAuthCSR, v1beta1.AddonKubeClientRegistrationAuthToken}))
	}
	return errs
}

// validateSpokeRegistrationWithHub rejects klusterlet chart grpcConfig when the hub uses gRPC registration.
// It evaluates the same merge as the reconciler (spec.klusterlet.values over spec.klusterlet.valuesFrom).
// When valuesFrom is set, the referenced ConfigMap and key must exist so the merged chart values can be checked.
func validateSpokeRegistrationWithHub(ctx context.Context, c client.Reader, spoke *v1beta1.Spoke, hub *v1beta1.Hub) field.ErrorList {
	var errs field.ErrorList
	if hub.Spec.RegistrationAuth.Driver != v1beta1.GRPCRegistrationDriver {
		return errs
	}

	merged, err := klusterletvalues.Merge(ctx, c, spoke.Namespace, spoke.Spec.Klusterlet, klusterletvalues.MergeOptions{
		StrictValuesFrom: true,
	})
	if err != nil {
		vf := field.NewPath("spec").Child("klusterlet").Child("valuesFrom")
		switch {
		case kerrs.IsNotFound(err):
			errs = append(errs, field.Invalid(vf, spoke.Spec.Klusterlet.ValuesFrom,
				fmt.Sprintf("ConfigMap must exist when the hub uses gRPC registration authentication so merged klusterlet values can be validated (grpcConfig must not be set): %v", err)))
		case errors.Is(err, klusterletvalues.ErrValuesFromKeyMissing):
			errs = append(errs, field.Invalid(vf, spoke.Spec.Klusterlet.ValuesFrom,
				fmt.Sprintf("referenced data key must exist when the hub uses gRPC registration authentication: %v", err)))
		default:
			errs = append(errs, field.InternalError(field.NewPath("spec").Child("klusterlet"), err))
		}
		return errs
	}

	if merged != nil && merged.GRPCConfig != "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("klusterlet"), spoke.Spec.Klusterlet,
			"effective klusterlet chart values (spec.klusterlet.values merged over spec.klusterlet.valuesFrom) must not set grpcConfig when hub registrationAuth.driver is grpc; configure gRPC via hub.spec.registrationAuth.grpc (server when endpointType is not loadBalancer, or hub status when loadBalancer) and hub status grpcServerCA"))
	}
	return errs
}
