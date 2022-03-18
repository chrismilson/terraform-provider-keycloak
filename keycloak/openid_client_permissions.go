package keycloak

import (
	"context"
	"fmt"
)

type OpenidClientPermissionsInput struct {
	Enabled bool `json:"enabled"`
}

type OpenidClientPermissions struct {
	RealmId          string            `json:"-"`
	ClientId         string            `json:"-"`
	Enabled          bool              `json:"enabled"`
	Resource         string            `json:"resource"`
	ScopePermissions map[string]string `json:"scopePermissions"`
}

func (keycloakClient *KeycloakClient) EnableOpenidClientPermissions(ctx context.Context, realmId, clientId string) error {
	return keycloakClient.put(ctx, fmt.Sprintf("/realms/%s/clients/%s/management/permissions", realmId, clientId), OpenidClientPermissionsInput{Enabled: true})
}

func (keycloakClient *KeycloakClient) DisableOpenidClientPermissions(ctx context.Context, realmId, clientId string) error {
	return keycloakClient.put(ctx, fmt.Sprintf("/realms/%s/clients/%s/management/permissions", realmId, clientId), OpenidClientPermissionsInput{Enabled: false})
}

func (keycloakClient *KeycloakClient) GetOpenidClientPermissions(ctx context.Context, realmId, clientId string) (*OpenidClientPermissions, error) {
	var openidClientPermissions OpenidClientPermissions
	openidClientPermissions.RealmId = realmId
	openidClientPermissions.ClientId = clientId

	err := keycloakClient.get(ctx, fmt.Sprintf("/realms/%s/clients/%s/management/permissions", realmId, clientId), &openidClientPermissions, nil)
	if err != nil {
		return nil, err
	}

	return &openidClientPermissions, nil
}
