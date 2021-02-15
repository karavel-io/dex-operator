package dex

import (
	"context"
	"fmt"
	"github.com/dexidp/dex/api/v2"
	"github.com/go-logr/logr"
	dexv1alpha1 "github.com/mikamai/dex-operator/api/v1alpha1"
	"github.com/mikamai/dex-operator/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Secret(dc *dexv1alpha1.DexClient) (v1.Secret, string, error) {
	secret, err := utils.GenerateRandomString(15)
	if err != nil {
		return v1.Secret{}, "", err
	}

	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("dex-%s-credentials", dc.Name),
			Namespace: dc.Namespace,
		},
		StringData: map[string]string{
			"clientId":     dc.ClientID(),
			"clientSecret": secret,
		},
	}, secret, nil
}

func AssertDexClient(ctx context.Context, log logr.Logger, svc *v1.Service, client *dexv1alpha1.DexClient, secret string) error {
	a, err := buildDexApi(svc, "")
	if err != nil {
		return err
	}

	id := client.ClientID()
	name := client.Spec.Name
	uris := client.Spec.RedirectUris
	public := client.Spec.Public
	log.Info("Asserting DexClient", "id", id, "name", name, "redirectUris", uris, "public", public)

	if public && secret != "" {
		return errors.New("a public client cannot have a secret")
	}

	if !public && secret == "" {
		return errors.New("a confidential client must have a secret")
	}

	creq := &api.CreateClientReq{
		Client: &api.Client{
			Id:           id,
			Name:         name,
			RedirectUris: uris,
			Public:       public,
			Secret:       secret,
		},
	}

	cres, err := a.CreateClient(ctx, creq)
	if err != nil {
		return err
	}

	if !cres.AlreadyExists {
		log.Info("Created DexClient", "id", id, "name", name, "redirectUris", uris, "public", public)
		return nil
	}

	ureq := &api.UpdateClientReq{
		Id:           id,
		Name:         name,
		RedirectUris: uris,
	}

	if _, err := a.UpdateClient(ctx, ureq); err != nil {
		return err
	}

	log.Info("Updated DexClient", "id", id, "name", name, "redirectUris", uris, "public", public)
	return nil
}

func DeleteDexClient(ctx context.Context, log logr.Logger, svc *v1.Service, client *dexv1alpha1.DexClient) error {
	a, err := buildDexApi(svc, "")
	if err != nil {
		return err
	}

	id := client.ClientID()
	name := client.Spec.Name
	log.Info("Deleting DexClient", "id", id, "name", name)
	req := &api.DeleteClientReq{
		Id: id,
	}
	if _, err := a.DeleteClient(ctx, req); err != nil {
		return err
	}

	log.Info("Deleted DexClient", "id", id, "name", name)
	return nil
}

func buildDexApi(svc *v1.Service, caPath string) (api.DexClient, error) {
	var port *int32
	for _, p := range svc.Spec.Ports {
		if p.Name == "grpc" {
			port = &p.Port
			break
		}
	}

	if port == nil {
		return nil, errors.Errorf("no port named 'grpc' found in service %s/%s", svc.Namespace, svc.Name)
	}

	host := fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, *port)
	opts := make([]grpc.DialOption, 0)
	if caPath != "" {
		creds, err := credentials.NewClientTLSFromFile(caPath, "")
		if err != nil {
			return nil, fmt.Errorf("load svc cert: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial: %v", err)
	}

	return api.NewDexClient(conn), nil
}
