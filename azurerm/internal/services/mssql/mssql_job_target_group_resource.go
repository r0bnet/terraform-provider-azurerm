package mssql

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/v3.0/sql"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/tf"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/mssql/parse"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/mssql/validate"
	azSchema "github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/tf/schema"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceMsSqlJobTargetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceMsSqlJobTargetGroupCreateUpdate,
		Read:   resourceMsSqlJobTargetGroupRead,
		Update: resourceMsSqlJobTargetGroupCreateUpdate,
		Delete: resourceMsSqlJobTargetGroupDelete,

		Importer: azSchema.ValidateResourceIDPriorToImport(func(id string) error {
			_, err := parse.JobTargetGroupID(id)
			return err
		}),

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"job_agent_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.JobAgentID,
			},

			"target": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice(
								[]string{
									string(sql.JobTargetTypeSQLServer),
									string(sql.JobTargetTypeSQLDatabase),
									string(sql.JobTargetTypeSQLElasticPool),
									string(sql.JobTargetTypeSQLShardMap),
								},
								false,
							),
						},

						"server_name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validate.ValidateMsSqlServerName,
						},

						"database_name": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validate.ValidateMsSqlDatabaseName,
						},

						"elastic_pool_name": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validate.ValidateMsSqlElasticPoolName,
						},

						"shard_map_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"refresh_credential_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validate.JobCredentialID,
						},

						"exclude": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func resourceMsSqlJobTargetGroupCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobTargetGroupsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Job Target Group creation.")

	jaId, err := parse.JobAgentID(d.Get("job_agent_id").(string))
	if err != nil {
		return err
	}
	jobTargetGroupId := parse.NewJobTargetGroupID(jaId.SubscriptionId, jaId.ResourceGroup, jaId.ServerName, jaId.Name, d.Get("name").(string))

	if d.IsNewResource() {
		existing, err := client.Get(ctx, jobTargetGroupId.ResourceGroup, jobTargetGroupId.ServerName, jobTargetGroupId.JobAgentName, jobTargetGroupId.TargetGroupName)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing MsSql %s: %+v", jobTargetGroupId, err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_mssql_job_target_group", *existing.ID)
		}
	}

	jobTargets, err := expandMssqlJobTargetGroupTargets(d)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", jobTargetGroupId, err)
	}

	jobTargetGroup := sql.JobTargetGroup{
		Name: utils.String(d.Get("name").(string)),
		JobTargetGroupProperties: &sql.JobTargetGroupProperties{
			Members: jobTargets,
		},
	}

	_, err = client.CreateOrUpdate(ctx, jobTargetGroupId.ResourceGroup, jobTargetGroupId.ServerName, jobTargetGroupId.JobAgentName, jobTargetGroupId.TargetGroupName, jobTargetGroup)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", jobTargetGroupId, err)
	}

	d.SetId(jobTargetGroupId.ID())

	return resourceMsSqlJobTargetGroupRead(d, meta)
}

func resourceMsSqlJobTargetGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobTargetGroupsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.JobTargetGroupID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, id.ResourceGroup, id.ServerName, id.JobAgentName, id.TargetGroupName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("reading %s: %+v", id, err)
	}

	d.Set("name", resp.Name)
	d.Set("target", flattenMssqlJobTargetGroupTargets(resp.Members))

	return nil
}

func resourceMsSqlJobTargetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobTargetGroupsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.JobTargetGroupID(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, id.ResourceGroup, id.ServerName, id.JobAgentName, id.TargetGroupName)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", id.TargetGroupName, err)
	}

	return nil
}

func expandMssqlJobTargetGroupTargets(d *schema.ResourceData) (*[]sql.JobTarget, error) {
	result := make([]sql.JobTarget, 0)

	targets := d.Get("target").(*schema.Set)
	for _, t := range targets.List() {
		target := t.(map[string]interface{})

		jt := sql.JobTarget{
			Type:           sql.JobTargetType(target["type"].(string)),
			MembershipType: sql.Include,
		}

		exclude := target["exclude"].(bool)
		if exclude {
			jt.MembershipType = sql.Exclude
		}

		if serverName, serverOk := target["server_name"]; serverOk && serverName != "" {
			jt.ServerName = utils.String(serverName.(string))
		}

		if dbName, dbOk := target["database_name"]; dbOk && dbName != "" {
			jt.DatabaseName = utils.String(dbName.(string))
		}

		if epName, epOk := target["elastic_pool_name"]; epOk && epName != "" {
			jt.ElasticPoolName = utils.String(epName.(string))
		}

		if smName, smOk := target["shard_map_name"]; smOk && smName != "" {
			jt.ShardMapName = utils.String(smName.(string))
		}

		if rcId, rcOk := target["refresh_credential_id"]; rcOk && rcId != "" {
			jt.RefreshCredential = utils.String(rcId.(string))
		}

		result = append(result, jt)
	}

	return &result, nil
}

func flattenMssqlJobTargetGroupTargets(targets *[]sql.JobTarget) []interface{} {
	result := make([]interface{}, 0)

	if targets == nil {
		return result
	}

	for _, target := range *targets {
		t := make(map[string]interface{})

		t["type"] = string(target.Type)

		if target.ServerName != nil {
			t["server_name"] = *target.ServerName
		}

		if target.DatabaseName != nil {
			t["database_name"] = *target.DatabaseName
		}

		if target.ElasticPoolName != nil {
			t["elastic_pool_name"] = *target.ElasticPoolName
		}

		if target.ShardMapName != nil {
			t["shard_map_name"] = *target.ShardMapName
		}

		if target.RefreshCredential != nil {
			t["refresh_credential_id"] = *target.RefreshCredential
		}

		if target.MembershipType == sql.Include {
			t["exclude"] = false
		} else {
			t["exclude"] = true
		}

		result = append(result, t)
	}

	return result
}
