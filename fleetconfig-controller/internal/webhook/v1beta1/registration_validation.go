package v1beta1

import (
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

func validateHubRegistrationAuth(h *v1beta1.Hub) field.ErrorList {
	var errs field.ErrorList
	base := field.NewPath("spec").Child("registrationAuth")
	ra := h.Spec.RegistrationAuth

	if ra.Driver == v1beta1.AWSIRSARegistrationDriver && ra.GRPC != nil {
		errs = append(errs, field.Invalid(base.Child("grpc"), ra.GRPC, "grpc must be unset when registrationAuth.driver is awsirsa"))
	}

	if ra.GRPC != nil && ra.GRPC.Join != nil && ra.Driver != v1beta1.GRPCRegistrationDriver {
		errs = append(errs, field.Invalid(base.Child("grpc").Child("join"), ra.GRPC.Join, "grpc.join is only allowed when registrationAuth.driver is grpc"))
	}

	if ra.GRPC != nil && ra.GRPC.Init != nil {
		et := ra.GRPC.Init.EndpointType
		if et != "" && !strings.EqualFold(et, v1beta1.GRPCEndpointTypeHostname) && !strings.EqualFold(et, v1beta1.GRPCEndpointTypeLoadBalancer) {
			errs = append(errs, field.NotSupported(base.Child("grpc").Child("init").Child("endpointType"), et, []string{v1beta1.GRPCEndpointTypeHostname, v1beta1.GRPCEndpointTypeLoadBalancer}))
		}
	}

	if ra.Driver == v1beta1.GRPCRegistrationDriver {
		if ra.GRPC == nil || ra.GRPC.Join == nil {
			errs = append(errs, field.Required(base.Child("grpc").Child("join"), "grpc.join is required when registrationAuth.driver is grpc"))
		} else {
			if ra.GRPC.Join.JoinServer == "" {
				errs = append(errs, field.Required(base.Child("grpc").Child("join").Child("joinServer"), "joinServer is required when registrationAuth.driver is grpc"))
			}
			if ra.GRPC.Join.CertificateAuthority == "" {
				errs = append(errs, field.Required(base.Child("grpc").Child("join").Child("certificateAuthority"), "certificateAuthority is required when registrationAuth.driver is grpc"))
			}
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

func validateSpokeRegistrationWithHub(spoke *v1beta1.Spoke, hub *v1beta1.Hub) field.ErrorList {
	var errs field.ErrorList
	if hub.Spec.RegistrationAuth.Driver != v1beta1.GRPCRegistrationDriver {
		return errs
	}
	if spoke.Spec.Klusterlet.Values != nil && spoke.Spec.Klusterlet.Values.GRPCConfig != "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("klusterlet").Child("values").Child("grpcConfig"), spoke.Spec.Klusterlet.Values.GRPCConfig,
			"grpcConfig must be empty when hub registrationAuth.driver is grpc; configure gRPC join via hub.spec.registrationAuth.grpc.join"))
	}
	return errs
}
