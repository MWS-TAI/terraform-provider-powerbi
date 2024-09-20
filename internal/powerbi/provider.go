package powerbi

import (
	"github.com/MWS-TAI/terraform-provider-powerbi/internal/powerbiapi"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"	
)

// Provider represents the powerbi terraform provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_TENANT_ID", ""),
				Description: "The Tenant ID for the tenant which contains the Azure Active Directory App Registration to use for performing Power BI REST API operations. This can also be sourced from the `POWERBI_TENANT_ID` Environment Variable",
			},
			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_CLIENT_ID", ""),
				Description: "Also called Application ID. The Client ID for the Azure Active Directory App Registration to use for performing Power BI REST API operations. This can also be sourced from the `POWERBI_CLIENT_ID` Environment Variable",
			},
			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_CLIENT_SECRET", ""),
				Description: "Also called Application Secret. The Client Secret for the Azure Active Directory App Registration to use for performing Power BI REST API operations. This can also be sourced from the `POWERBI_CLIENT_SECRET` Environment Variable",
			},
			"access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_ACCESS_TOKEN", ""),
				Description: "The access token for the a Power BI user to use for performing Power BI REST API operations. If provided will use access token flow with delegate permissions. This can also be sourced from the `POWERBI_ACCESS_TOKEN` Environment Variable",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_USERNAME", ""),
				Description: "The username for the a Power BI user to use for performing Power BI REST API operations. If provided will use resource owner password credentials flow with delegate permissions. This can also be sourced from the `POWERBI_USERNAME` Environment Variable",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("POWERBI_PASSWORD", ""),
				Description: "The password for the a Power BI user to use for performing Power BI REST API operations. If provided will use resource owner password credentials flow with delegate permissions. This can also be sourced from the `POWERBI_PASSWORD` Environment Variable",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"powerbi_workspace":        ResourceWorkspace(),
			"powerbi_pbix":             ResourcePBIX(),
			"powerbi_refresh_schedule": ResourceRefreshSchedule(),
			"powerbi_workspace_access": ResourceGroupUsers(),
			"powerbi_dataset":          ResourceDataset(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"powerbi_workspace": DataSourceWorkspace(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	// Check if the access token is provided
	accessToken, accessTokenOk := d.GetOk("access_token")
	if accessTokenOk {
		return powerbiapi.NewClientWithAccessToken(accessToken.(string))
	}

	// Check if the username and password are provided
	username, usernameOk := d.GetOk("username")
	password, passwordOk := d.GetOk("password")

	if usernameOk && passwordOk {
		return powerbiapi.NewClientWithPasswordAuth(
			d.Get("tenant_id").(string),
			d.Get("client_id").(string),
			d.Get("client_secret").(string),
			username.(string),
			password.(string),
		)
	}

	// Check if the client_id and client_secret are provided
	tenant_id, tenant_idOk := d.GetOk("tenant_id")
	client_id, client_idOk := d.GetOk("client_id")
	client_secret, client_secretOk := d.GetOk("client_secret")
	if tenant_idOk && client_idOk && client_secretOk {
		return powerbiapi.NewClientWithClientCredentialAuth(
			tenant_id.(string),
			client_id.(string),
			client_secret.(string),
		)
	}

	// Default to Azure CLI authentication
	return powerbiapi.NewClientWithAzureCLIAuth()
}
