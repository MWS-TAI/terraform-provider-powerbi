package main

import (
	"github.com/MWS-TAI/terraform-provider-powerbi/internal/powerbi"
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return powerbi.Provider()
		},
	})
}
