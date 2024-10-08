# Terraform Provider for Power BI

[![Test status](https://github.com/MWS-TAI/terraform-provider-powerbi/workflows/tests/badge.svg?branch=master)](https://github.com/MWS-TAI/terraform-provider-powerbi/actions?query=workflow%3Atests+branch%3Amaster)

A Terraform provider that allows the creation and updating of Power BI resources

The Power BI Provider supports Terraform 0.12.x. It may still function on earlier versions but has only been tested on 0.12.x and later versions

* [Terraform Website](https://www.terraform.io)

## Traffyk Building
GOOS=darwin GOARCH=arm64 go build -o terraform-provider-powerbi_v1.0.0_darwin_arm64
GOOS=linux GOARCH=amd64 go build -o terraform-provider-powerbi_v1.0.0_linux_amd64
GOOS=windows GOARCH=amd64 go build -o terraform-provider-powerbi_v1.0.0_windows_amd64

## Installation

### Registry
If you use Terraform 0.13 or greater, the provider can be installed from the [terraform provider registry](https://registry.terraform.io/providers/codecutout/powerbi/latest) by specifying the following item in your terraform

```terraform
terraform {
  required_providers {
    powerbi = {
      source = "codecutout/powerbi"
      version = "~>1.3"
    }
  }
}
```

### Local
If using terraform 0.12 plugins must be installed locally

1. From the [releases](/releases) section download the zip file for your desired version, operating system and architecture
2. Extract the zip file into `%APPDATA%\terraform.d\plugins` for windows, or `~/.terraform.d/plugins` for other systems
3. `terraform init` should now detect usage of the provider and apply the plugin

Further details about installing terraform plugs can be found at https://www.terraform.io/docs/plugins/basics.html#installing-plugins

## Usage Example

```terraform
# Configure the Power BI Provider
provider "powerbi" {
  tenant_id       = "..."
  client_id       = "..."
  client_secret   = "..."
  username        = "..."
  password        = "..."
}

# Create a workspace
resource "powerbi_workspace" "example" {
  name     = "Example Workspace"
}

# Create a pbix within the workspace
resource "powerbi_pbix" "example" {
	workspace_id = "${powerbi_workspace.example.id}"
	name = "My PBIX"
	source = "./my-pbix.pbix"
	source_hash = "${filemd5(".my-pbix.pbix")}"
	datasource {
		type = "OData"
		url = "https://services.odata.org/V3/(S(kbiqo1qkby04vnobw0li0fcp))/OData/OData.svc"
		original_url = "https://services.odata.org/V3/OData/OData.svc"
	}
}
```

## Documentation
Provider and resources properties and example usages can be found in this repositories [docs](docs) folder

## Developer Requirements

* [Terraform](https://www.terraform.io/downloads.html) version 0.12.x +
* [Go](https://golang.org/doc/install) version 1.13.x (to build the provider plugin)

If you're on Windows you'll also need:
* [Git Bash for Windows](https://git-scm.com/download/win)

For *Git Bash for Windows*, at the step of "Adjusting your PATH environment", please choose "Use Git and optional Unix tools from Windows Command Prompt".*

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.13+ is **required**). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

### Build
```sh
$ go build
```

### Documentation generation
Documentation markdown files are partly generated from terraform schema definitions. To regenreate the documentation from updated schema run
``` sh
$ go run internal/docgen/cmd/main.go
```

### Testing
```sh
$ go test -v ./...
```

The majority of tests in the provider are Acceptance Tests - which provisions real resources in power BI. It's possible to run the acceptance tests with the above command by setting the following enviornment variables: 
- `TF_ACC=1`
- `POWERBI_TENANT_ID`
- `POWERBI_CLIENT_ID`
- `POWERBI_CLIENT_SECRET`
- `POWERBI_USERNAME`
- `POWERBI_PASSWORD`

### Running with Terraform on Windows
- Run `go build` - This will build and deploy `terraform-provider-powerbi.exe`
- Run `mkdir %APPDATA%\terraform.d\plugins\local.dev\codecutout\powerbi\0.1\windows_amd64` to provison a [locally available provider namespace](https://www.terraform.io/docs/language/providers/requirements.html#in-house-providers)
- Move the binary to the local namespace: `move terraform-provider-powerbi.exe %APPDATA%\terraform.d\plugins\local.dev\codecutout\powerbi\0.1\windows_amd64`
- Use the custom build provider from Terraform:
```terraform
terraform {
  required_providers {
    powerbi = {
      source  = "local.dev/codecutout/powerbi"
    }
  }
}
```

