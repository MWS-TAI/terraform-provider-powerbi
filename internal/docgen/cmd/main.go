package main

import (
	"github.com/MWS-TAI/terraform-provider-powerbi/internal/docgen"
	"github.com/MWS-TAI/terraform-provider-powerbi/internal/powerbi"
)

func main() {
	docgen.PopulateTerraformDocs("./docs", "powerbi", powerbi.Provider())
}
