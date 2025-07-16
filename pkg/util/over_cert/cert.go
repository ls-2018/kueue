package over_cert

import (
	"fmt"

	"github.com/go-logr/logr"
	cert "github.com/open-policy-agent/cert-controller/pkg/rotator"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	config "sigs.k8s.io/kueue/apis/config/v1beta1"
)

const (
	vwcName        = "kueue-validating-webhook-configuration"
	mwcName        = "kueue-mutating-webhook-configuration"
	caName         = "kueue-ca"
	caOrganization = "kueue"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;update

// ManageCerts creates all certs for webhooks. This function is called from main.go.
func ManageCerts(mgr ctrl.Manager, cfg config.Configuration, setupFinished chan struct{}) error {
	// DNSName is <service name>.<namespace>.svc
	var dnsName = fmt.Sprintf("%s.%s.svc", *cfg.InternalCertManagement.WebhookServiceName, *cfg.Namespace)

	return cert.AddRotator(mgr, &cert.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: *cfg.Namespace,
			Name:      *cfg.InternalCertManagement.WebhookSecretName,
		},
		CertDir:        cfg.Webhook.CertDir,
		CAName:         caName,
		CAOrganization: caOrganization,
		DNSName:        dnsName,
		IsReady:        setupFinished,
		Webhooks: []cert.WebhookInfo{{
			Type: cert.Validating,
			Name: vwcName,
		}, {
			Type: cert.Mutating,
			Name: mwcName,
		}},
		// When kueue is running in the leader election mode,
		// we expect webhook server will run in primary and secondary instance
		RequireLeaderElection: false,
	})
}

func WaitForCertsReady(log logr.Logger, certsReady chan struct{}) {
	log.Info("Waiting for certificate generation to complete")
	<-certsReady
	log.Info("Certs ready")
}
