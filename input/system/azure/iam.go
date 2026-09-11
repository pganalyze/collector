package azure

import (
	"context"
	"fmt"

	"github.com/pganalyze/collector/config"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// dbAuthScope is the OAuth scope for Azure Database for PostgreSQL (single
// server and flexible server) Entra ID (AAD) authentication.
//
// See https://learn.microsoft.com/en-us/azure/postgresql/flexible-server/how-to-configure-sign-in-azure-ad-authentication
const dbAuthScope = "https://ossrdbms-aad.database.windows.net/.default"

// GetDbAuthToken fetches an Entra ID (AAD) access token to use as the
// password when connecting to Azure Database for PostgreSQL with IAM auth
// (i.e. db_use_iam_auth). This supports the same credential sources as our
// Event Hub / ARM integrations: client secret, client certificate, workload
// identity, or (the common case for PoC/testing) VM/AKS managed identity.
func GetDbAuthToken(ctx context.Context, config config.ServerConfig) (string, error) {
	credential, err := getAzureCredential(config)
	if err != nil {
		return "", err
	}

	token, err := credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{dbAuthScope},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get Azure AD token for database authentication: %s", err)
	}

	return token.Token, nil
}
