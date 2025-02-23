// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package quicksight

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
)

const (
	DefaultGroupNamespace = "default"
)

// @SDKResource("aws_quicksight_group", name="Group")
func ResourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceGroupCreate,
		ReadWithoutTimeout:   resourceGroupRead,
		UpdateWithoutTimeout: resourceGroupUpdate,
		DeleteWithoutTimeout: resourceGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaFunc: func() map[string]*schema.Schema {
			return map[string]*schema.Schema{
				"arn": {
					Type:     schema.TypeString,
					Computed: true,
				},

				"aws_account_id": {
					Type:     schema.TypeString,
					Optional: true,
					Computed: true,
					ForceNew: true,
				},

				"description": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"group_name": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},

				"namespace": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Default:  DefaultGroupNamespace,
					ValidateFunc: validation.All(
						validation.StringLenBetween(1, 63),
						validation.StringMatch(regexache.MustCompile(`^[a-zA-Z0-9._-]*$`), "must contain only alphanumeric characters, hyphens, underscores, and periods"),
					),
				},
			}
		},
	}
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).QuickSightConn(ctx)

	awsAccountID := meta.(*conns.AWSClient).AccountID
	namespace := d.Get("namespace").(string)

	if v, ok := d.GetOk("aws_account_id"); ok {
		awsAccountID = v.(string)
	}

	createOpts := &quicksight.CreateGroupInput{
		AwsAccountId: aws.String(awsAccountID),
		Namespace:    aws.String(namespace),
		GroupName:    aws.String(d.Get("group_name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		createOpts.Description = aws.String(v.(string))
	}

	resp, err := conn.CreateGroupWithContext(ctx, createOpts)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating QuickSight Group: %s", err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", awsAccountID, namespace, aws.StringValue(resp.Group.GroupName)))

	return append(diags, resourceGroupRead(ctx, d, meta)...)
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).QuickSightConn(ctx)

	awsAccountID, namespace, groupName, err := GroupParseID(d.Id())
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading QuickSight Group (%s): %s", d.Id(), err)
	}

	descOpts := &quicksight.DescribeGroupInput{
		AwsAccountId: aws.String(awsAccountID),
		Namespace:    aws.String(namespace),
		GroupName:    aws.String(groupName),
	}

	resp, err := conn.DescribeGroupWithContext(ctx, descOpts)
	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, quicksight.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] QuickSight Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading QuickSight Group (%s): %s", d.Id(), err)
	}

	d.Set("arn", resp.Group.Arn)
	d.Set("aws_account_id", awsAccountID)
	d.Set("group_name", resp.Group.GroupName)
	d.Set("description", resp.Group.Description)
	d.Set("namespace", namespace)

	return diags
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).QuickSightConn(ctx)

	awsAccountID, namespace, groupName, err := GroupParseID(d.Id())
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "updating QuickSight Group (%s): %s", d.Id(), err)
	}

	updateOpts := &quicksight.UpdateGroupInput{
		AwsAccountId: aws.String(awsAccountID),
		Namespace:    aws.String(namespace),
		GroupName:    aws.String(groupName),
	}

	if v, ok := d.GetOk("description"); ok {
		updateOpts.Description = aws.String(v.(string))
	}

	_, err = conn.UpdateGroupWithContext(ctx, updateOpts)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "updating QuickSight Group %s: %s", d.Id(), err)
	}

	return append(diags, resourceGroupRead(ctx, d, meta)...)
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).QuickSightConn(ctx)

	awsAccountID, namespace, groupName, err := GroupParseID(d.Id())
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting QuickSight Group (%s): %s", d.Id(), err)
	}

	deleteOpts := &quicksight.DeleteGroupInput{
		AwsAccountId: aws.String(awsAccountID),
		Namespace:    aws.String(namespace),
		GroupName:    aws.String(groupName),
	}

	if _, err := conn.DeleteGroupWithContext(ctx, deleteOpts); err != nil {
		if tfawserr.ErrCodeEquals(err, quicksight.ErrCodeResourceNotFoundException) {
			return diags
		}
		return sdkdiag.AppendErrorf(diags, "deleting QuickSight Group %s: %s", d.Id(), err)
	}

	return diags
}

func GroupParseID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected AWS_ACCOUNT_ID/NAMESPACE/GROUP_NAME", id)
	}
	return parts[0], parts[1], parts[2], nil
}
