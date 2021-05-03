package mssql

import (
	"bytes"
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
							Optional: true,
							ValidateFunc: validation.StringInSlice(
								[]string{
									"Server",
									"Database",
									"ElasticPool",
									"ShardMap",
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

						"refresh_credential_name": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"exclude": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
				Set: hashMsSqlJobTargetGroupTarget,
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
			MembershipType: sql.Include,
		}

		exclude := target["exclude"].(bool)
		if exclude {
			jt.MembershipType = sql.Exclude
		}

		if targetType, ok := target["type"]; ok {
			switch targetType {
			case "Server":
				jt.Type = sql.JobTargetTypeSQLServer
				break
			case "Database":
				jt.Type = sql.JobTargetTypeSQLDatabase
				break
			case "ElasticPool":
				jt.Type = sql.JobTargetTypeSQLElasticPool
				break
			case "ShardMap":
				jt.Type = sql.JobTargetTypeSQLShardMap
				break
			}
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

		if rcName, rcOk := target["refresh_credential_name"]; rcOk && rcName != "" {
			jt.RefreshCredential = utils.String(rcName.(string))
		}

		result = append(result, jt)
	}

	return &result, nil
}

func flattenMssqlJobTargetGroupTargets(targets *[]sql.JobTarget) *schema.Set {
	if targets == nil {
		return schema.NewSet(schema.HashString, make([]interface{}, 0))
	}

	result := make([]interface{}, 0)
	for _, target := range *targets {
		t := make(map[string]interface{})

		switch target.Type {
		case sql.JobTargetTypeSQLServer:
			t["type"] = "Server"
			break
		case sql.JobTargetTypeSQLDatabase:
			t["type"] = "Database"
			break
		case sql.JobTargetTypeSQLElasticPool:
			t["type"] = "ElasticPool"
			break
		case sql.JobTargetTypeSQLShardMap:
			t["type"] = "ShardMap"
			break
		}

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
			t["refresh_credential_name"] = *target.RefreshCredential
		}

		if target.MembershipType == sql.Include {
			t["exclude"] = false
		} else {
			t["exclude"] = true
		}

		result = append(result, t)
	}

	return schema.NewSet(hashMsSqlJobTargetGroupTarget, result)
}

func hashMsSqlJobTargetGroupTarget(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["server_name"].(string)))

	if val, ok := m["database_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", val.(string)))
	}

	if val, ok := m["elastic_pool_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", val.(string)))
	}

	if val, ok := m["shard_map"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", val.(string)))
	}

	if val, ok := m["refresh_credential_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", val.(string)))
	}

	if val, ok := m["exclude"]; ok {
		buf.WriteString(fmt.Sprintf("%t", val.(bool)))
	}

	return schema.HashString(buf.String())
}
