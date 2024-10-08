package powerbi

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MWS-TAI/terraform-provider-powerbi/internal/powerbiapi"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// ResourcePBIX represents a Power BI PBIX file
func ResourcePBIX() *schema.Resource {
	return &schema.Resource{
		Create: createPBIX,
		Read:   readPBIX,
		Update: updatePBIX,
		Delete: deletePBIX,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"workspace_id": {
				Type:        schema.TypeString,
				Description: "Workspace ID in which the PBIX will be added.",
				Required:    true,
				ForceNew:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the PBIX. This will be used as the name for the report and dataset.",
				Required:    true,
				ForceNew:    true,
			},
			"source": {
				Type:        schema.TypeString,
				Description: "An absolute path to a PBIX file on the local system.",
				Required:    true,
			},
			"source_hash": {
				Type:        schema.TypeString,
				Description: "Used to trigger updates. The only meaningful value is `${filemd5(\"path/to/file\")}`.",
				Optional:    true,
			},
			"skip_report": {
				Type:        schema.TypeBool,
				Description: "If true, only the PBIX dataset is deployed.",
				Optional:    true,
				Default:     false,
			},
			"report_id": {
				Type:        schema.TypeString,
				Description: "The ID for the report that was deployed as part of the PBIX.",
				Computed:    true,
			},
			"dataset_id": {
				Type:        schema.TypeString,
				Description: "The ID for the dataset that was deployed as part of the PBIX.",
				Computed:    true,
			},
			"rebind_dataset_id": {
				Type:          schema.TypeString,
				Description:   "If set, will rebind the report to the the specified dataset ID.",
				Optional:      true,
				ConflictsWith: []string{"parameter", "datasource"},
			},
			"report_original_dataset_id": {
				Type:        schema.TypeString,
				Description: "The dataset to which the report that was deployed is pointing. This is primarily used to allow reverting rebinded datasets back to the original source.",
				Computed:    true,
			},
			"parameter": {
				Type:        schema.TypeSet,
				Description: "Parameters to be configured on the PBIX dataset. These can be updated without requiring reuploading the PBIX. Any parameters not mentioned will not be tracked or updated",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "The parameter name",
							Required:    true,
						},
						"value": {
							Type:        schema.TypeString,
							Description: "The parameter value",
							Required:    true,
						},
					},
				},
			},
			"datasource": {
				Type:        schema.TypeSet,
				Description: "Datasources to be reconfigured after deploying the PBIX dataset. Changing this value will require reuploading the PBIX. Any datasource updated will not be tracked",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Description: "The type of datasource. For example web, sql",
							Optional:    true,
						},
						"database": {
							Type:        schema.TypeString,
							Description: "The database name, if applicable for the type of datasource",
							Optional:    true,
						},
						"server": {
							Type:        schema.TypeString,
							Description: "The server name, if applicable for the type of datasource",
							Optional:    true,
						},
						"url": {
							Type:        schema.TypeString,
							Description: "The service URL, if applicable for the type of datasource",
							Optional:    true,
						},
						"original_database": {
							Type:        schema.TypeString,
							Description: "The database name as configured in the PBIX, if applicable for the type of datasource This will be the value replaced with the value in the 'databsase' field",
							Optional:    true,
						},
						"original_server": {
							Type:        schema.TypeString,
							Description: "The server name as configured in the PBIX, if applicable for the type of datasource. This will be the value replaced with the value in the 'server' field",
							Optional:    true,
						},
						"original_url": {
							Type:        schema.TypeString,
							Description: "The service URL as configured in the PBIX, if applicable for the type of datasource. This will be the value replaced with the value in the 'url' field",
							Optional:    true,
						},
					},
				},
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(5 * time.Minute),
		},
	}
}

func openContentReader(d *schema.ResourceData) (io.Reader, error) {
	filepath := d.Get("source").(string)
	return os.Open(filepath)
}

func createPBIX(d *schema.ResourceData, meta interface{}) error {

	d.Partial(true)

	err := createImport(d, meta)
	if err != nil {
		return err
	}

	err = readImport(d, meta, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return err
	}

	err = setPBIXParameters(d, meta)
	if err != nil {
		return err
	}

	err = setPBIXDatasources(d, meta)
	if err != nil {
		return err
	}

	if _, ok := d.GetOk("rebind_dataset_id"); ok {
		err = rebindPBIXDataset(d, meta)
		if err != nil {
			return err
		}
	}

	d.Partial(false)

	return nil

}

func readPBIX(d *schema.ResourceData, meta interface{}) error {

	err := readImport(d, meta, d.Timeout(schema.TimeoutRead))
	if isHTTP404Error(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}

	err = readPBIXParameters(d, meta)
	if err != nil {
		return err
	}

	err = readPBIXDatasources(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func updatePBIX(d *schema.ResourceData, meta interface{}) error {

	// Take over the dataset / report to ensure we can update it
	err := takeOverPBIXReport(d, meta)
	if err != nil {
		return err
	}

	if d.HasChange("source") || d.HasChange("source_hash") || d.HasChange("datasource") {

		d.Partial(true)

		// Imports do not update rebinded datasets, so we unbind before doing the import
		err := unbindPBIXDataset(d, meta)
		if err != nil {
			return err
		}

		err = createImport(d, meta)
		if err != nil {
			return err
		}

		err = readImport(d, meta, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return err
		}

		err = setPBIXParameters(d, meta)
		if err != nil {
			return err
		}

		err = setPBIXDatasources(d, meta)
		if err != nil {
			return err
		}

		err = rebindPBIXDataset(d, meta)
		if err != nil {
			return err
		}

		d.Partial(false)

		return nil
	}

	if d.HasChange("rebind_dataset_id") {
		err := unbindPBIXDataset(d, meta)
		if err != nil {
			return err
		}

		err = rebindPBIXDataset(d, meta)
		if err != nil {
			return err
		}
	}

	if d.HasChange("parameter") {
		err := setPBIXParameters(d, meta)
		if err != nil {
			return err
		}
	}

	return nil
}

func deletePBIX(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*powerbiapi.Client)

	groupID := d.Get("workspace_id").(string)

	// Take over the dataset / report to ensure we can delete it
	err := takeOverPBIXReport(d, meta)
	if err != nil {
		return err
	}

	if reportID, reportIDOk := d.GetOk("report_id"); reportIDOk {
		err := client.DeleteReportInGroup(groupID, reportID.(string))
		if err != nil {
			return err
		}
	}

	if datasetID, datasetIDOk := d.GetOk("dataset_id"); datasetIDOk {
		err := client.DeleteDatasetInGroup(groupID, datasetID.(string))
		if err != nil {
			return err
		}
	}

	return nil
}

func createImport(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*powerbiapi.Client)

	reader, err := openContentReader(d)
	if err != nil {
		return err
	}

	resp, err := client.PostImportInGroup(
		d.Get("workspace_id").(string),
		d.Get("name").(string),
		"CreateOrOverwrite",
		d.Get("skip_report").(bool),
		reader,
	)
	if err != nil {
		return err
	}

	d.SetId(resp.ID)
	d.SetPartial("workspace_id")
	d.SetPartial("source")
	d.SetPartial("source_hash")

	return nil
}

func readImport(d *schema.ResourceData, meta interface{}, timeoutForSuccessfulImport time.Duration) error {
	client := meta.(*powerbiapi.Client)
	id := d.Id()
	groupID := d.Get("workspace_id").(string)

	im, err := client.WaitForImportInGroupToSucceed(groupID, id, timeoutForSuccessfulImport)
	if err != nil {
		return err
	}

	d.SetPartial("name")
	d.Set("name", im.Name)

	// powerbi imports can be modified by some operations (such as rebind)
	// in order to keep reference to the original report and original dataset
	// we will only look them up once after creation
	if d.IsNewResource() {
		if len(im.Reports) >= 1 {
			d.SetPartial("report_id")
			d.Set("report_id", im.Reports[0].ID)

			report, err := client.GetReportInGroup(groupID, im.Reports[0].ID)
			if err != nil {
				return err
			}
			d.SetPartial("report_original_dataset_id")
			d.Set("report_original_dataset_id", report.DatasetID)
		}

		if len(im.Datasets) >= 1 {
			d.SetPartial("dataset_id")
			d.Set("dataset_id", im.Datasets[0].ID)
		}
	}
	return nil
}

func setPBIXParameters(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*powerbiapi.Client)
	parameter := d.Get("parameter").(*schema.Set)
	datasetID, datasetOk := d.GetOk("dataset_id")
	groupID := d.Get("workspace_id").(string)
	if parameter != nil {
		parameterList := parameter.List()
		if len(parameterList) > 0 {

			if !datasetOk {
				return fmt.Errorf("Unable to update parameters on a PBIX file that does not contain a dataset")
			}

			updateParameterRequest := powerbiapi.UpdateParametersInGroupRequest{}
			for _, parameterObj := range parameterList {
				parameterObj := parameterObj.(map[string]interface{})
				updateParameterRequest.UpdateDetails = append(updateParameterRequest.UpdateDetails, powerbiapi.UpdateParametersInGroupRequestItem{
					Name:     parameterObj["name"].(string),
					NewValue: parameterObj["value"].(string),
				})
			}

			err := client.UpdateParametersInGroup(groupID, datasetID.(string), updateParameterRequest)
			if err != nil {
				return err
			}

			d.SetPartial("parameter")
			d.Set("parameter", parameter)
		}
	}
	return nil
}

func readPBIXParameters(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*powerbiapi.Client)

	groupID := d.Get("workspace_id").(string)
	datasetID, datasetOK := d.GetOk("dataset_id")
	stateParameters := d.Get("parameter").(*schema.Set)

	// some pbix do not have datasets, and therefore not all have parameters
	if !datasetOK {
		return nil
	}

	apiParameters, err := client.GetParametersInGroup(groupID, datasetID.(string))
	if err != nil {
		return err
	}

	for _, stateParameter := range stateParameters.List() {
		for _, apiParameter := range apiParameters.Value {
			stateParameterObj := stateParameter.(map[string]interface{})
			if stateParameterObj["name"] == apiParameter.Name {
				stateParameterObj["value"] = apiParameter.CurrentValue
			}
		}
	}

	d.SetPartial("parameter")
	d.Set("parameter", stateParameters)
	return nil
}

func setPBIXDatasources(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*powerbiapi.Client)
	datasources := d.Get("datasource").(*schema.Set)
	datasetID, datasetOk := d.GetOk("dataset_id")
	groupID := d.Get("workspace_id").(string)

	if datasources != nil {
		datasourceList := datasources.List()
		if len(datasourceList) > 0 {

			if !datasetOk {
				return fmt.Errorf("Unable to update datasources on a PBIX file that does not contain a dataset")
			}

			updateDatasourcesRequest := powerbiapi.UpdateDatasourcesInGroupRequest{}
			for _, datasourceObj := range datasourceList {
				datasourceObj := datasourceObj.(map[string]interface{})
				updateDatasourcesRequest.UpdateDetails = append(updateDatasourcesRequest.UpdateDetails, powerbiapi.UpdateDatasourcesInGroupRequestItem{
					ConnectionDetails: powerbiapi.UpdateDatasourcesInGroupRequestItemConnectionDetails{
						URL:      emptyStringToNil(datasourceObj["url"].(string)),
						Database: emptyStringToNil(datasourceObj["database"].(string)),
						Server:   emptyStringToNil(datasourceObj["server"].(string)),
					},
					DatasourceSelector: powerbiapi.UpdateDatasourcesInGroupRequestItemDatasourceSelector{
						DatasourceType: datasourceObj["type"].(string),
						ConnectionDetails: powerbiapi.UpdateDatasourcesInGroupRequestItemConnectionDetails{
							URL:      emptyStringToNil(datasourceObj["original_url"].(string)),
							Database: emptyStringToNil(datasourceObj["original_database"].(string)),
							Server:   emptyStringToNil(datasourceObj["original_server"].(string)),
						},
					},
				})
			}

			err := client.UpdateDatasourcesInGroup(groupID, datasetID.(string), updateDatasourcesRequest)
			if err != nil {
				return err
			}

			d.SetPartial("datasource")
			d.Set("datasource", datasources)
		}
	}
	return nil
}

func readPBIXDatasources(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*powerbiapi.Client)

	datasetID, datasetOk := d.GetOk("dataset_id")
	groupID := d.Get("workspace_id").(string)
	stateDatasources := d.Get("datasource").(*schema.Set)

	// some pbix do not have datasets, and therefore not all have datasources
	if !datasetOk {
		return nil
	}

	apiDatasources, err := client.GetDatasourcesInGroup(groupID, datasetID.(string))
	if err != nil {
		return err
	}

	// Because datasource updates work in "find and replace" kind of semantic, it is
	// impossible to track the values of individual datasources. However we can
	// determine if there are no datasource that match our original replacement
	for _, stateDatasource := range stateDatasources.List() {
		stateDatasourceObj := stateDatasource.(map[string]interface{})
		anyAPIMatchesState := false
		for _, apiDatasource := range apiDatasources.Value {
			apiMatchesState := (stateDatasourceObj["url"] == "" || stateDatasourceObj["url"] == *apiDatasource.ConnectionDetails.URL) &&
				(stateDatasourceObj["server"] == "" || stateDatasourceObj["server"] == *apiDatasource.ConnectionDetails.Server) &&
				(stateDatasourceObj["database"] == "" || stateDatasourceObj["database"] == *apiDatasource.ConnectionDetails.Database)
			anyAPIMatchesState = anyAPIMatchesState || apiMatchesState
		}

		if !anyAPIMatchesState {
			if stateDatasourceObj["url"] != "" {
				stateDatasourceObj["url"] = "???"
			}
			if stateDatasourceObj["server"] != "" {
				stateDatasourceObj["server"] = "???"
			}
			if stateDatasourceObj["database"] != "" {
				stateDatasourceObj["database"] = "???"
			}
		}
	}

	d.SetPartial("datasource")
	d.Set("datasource", stateDatasources)
	return nil
}

func rebindPBIXDataset(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*powerbiapi.Client)

	groupID := d.Get("workspace_id").(string)
	rebindDatasetID, rebindDatasetOk := d.GetOk("rebind_dataset_id")
	reportID, reportOk := d.GetOk("report_id")

	// only run this if rebind_dataset_id and report_id are available
	if !rebindDatasetOk || !reportOk {
		return nil
	}

	return client.RebindReportInGroup(groupID, reportID.(string), powerbiapi.RebindReportInGroupRequest{
		DatasetID: rebindDatasetID.(string),
	})
}

func unbindPBIXDataset(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*powerbiapi.Client)

	groupID := d.Get("workspace_id").(string)
	originalDatasetID, datasetOk := d.GetOk("report_original_dataset_id")
	reportID, reportOk := d.GetOk("report_id")
	_, rebindDatasetOk := d.GetOk("rebind_dataset_id")

	// Only unbind if we either know we are rebinded, or rebinding has changed and we might be rebinded
	isRebinded := rebindDatasetOk || d.HasChange("rebind_dataset_id")

	if !isRebinded || !datasetOk || !reportOk {
		return nil
	}

	return client.RebindReportInGroup(groupID, reportID.(string), powerbiapi.RebindReportInGroupRequest{
		DatasetID: originalDatasetID.(string),
	})
}

func takeOverPBIXReport(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*powerbiapi.Client)

	groupID := d.Get("workspace_id").(string)
	datasetID, datasetOk := d.GetOk("dataset_id")
	reportID, reportOk := d.GetOk("report_id")

	var err error

	if datasetOk {
		err = client.TakeOverDatasetInGroup(groupID, datasetID.(string))
	} else if reportOk {
		err = client.TakeOverReportInGroup(groupID, reportID.(string))
	}

	if err != nil {
		return err
	}

	return nil
}
