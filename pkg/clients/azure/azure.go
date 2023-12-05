package azure

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/kubelogin/pkg/token"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

// Credentials Secret content is a json whose keys are below.
const (
	CredentialsKeyClientID       = "clientId"
	CredentialsKeyClientSecret   = "clientSecret"
	CredentialsKeyTenantID       = "tenantId"
	CredentialsKeyClientCert     = "clientCertificate"
	CredentialsKeyClientCertPass = "clientCertificatePassword"
)

func WrapRESTConfig(_ context.Context, rc *rest.Config, credentials []byte, _ ...string) error {
	m := map[string]string{}
	if err := json.Unmarshal(credentials, &m); err != nil {
		return err
	}

	fs := pflag.NewFlagSet("kubelogin", pflag.ContinueOnError)
	opts := token.NewOptions()
	opts.AddFlags(fs)
	// opts are filled with provided args
	err := fs.Parse(rc.ExecProvider.Args)
	if err != nil {
		return errors.Wrap(err, "could not parse execProvider arguments in kubeconfig")
	}
	rc.ExecProvider = nil
	// TODO: support other login methods like MSI, Workload Identity in the future
	opts.LoginMethod = token.ServicePrincipalLogin
	opts.ClientID = m[CredentialsKeyClientID]
	opts.ClientSecret = m[CredentialsKeyClientSecret]
	opts.TenantID = m[CredentialsKeyTenantID]
	if cert, ok := m[CredentialsKeyClientCert]; ok {
		opts.ClientCert = cert
		if certpass, ok2 := m[CredentialsKeyClientCertPass]; ok2 {
			opts.ClientCertPassword = certpass
		}
	}
	// ServerID is extracted from the execProvider section of unconverted kubeconfig
	// it is constant for Azure AKS
	// opts.ServerID = "6dae42f8-4368-4678-94ff-3960e28e3630"

	p, err := token.NewTokenProvider(&opts)
	if err != nil {
		return errors.New("cannot build azure token provider")
	}

	rc.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &tokenTransport{Provider: p, Base: rt}
	})

	return nil
}
