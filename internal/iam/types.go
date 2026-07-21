// Package iam inspects IAM roles and managed policies.
//
//nolint:revive // max-public-structs: DTOs exported for MCP and JSON consumers
package iam

// RoleSummary is a list-roles row.
type RoleSummary struct {
	RoleName   string `json:"roleName"`
	Arn        string `json:"arn"`
	Path       string `json:"path"`
	CreateDate string `json:"createDate"`
}

// PolicySummary is a list-policies row.
type PolicySummary struct {
	PolicyName       string `json:"policyName"`
	Arn              string `json:"arn"`
	DefaultVersionID string `json:"defaultVersionId"`
	UpdateDate       string `json:"updateDate"`
}

// AWS CLI-compatible dump JSON uses PascalCase field names (get-policy / get-role).

//nolint:tagliatelle // mirrors aws CLI JSON
type PolicyDump struct {
	Policy   PolicyDetail `json:"Policy"`
	Document any          `json:"Document"`
	Services []string     `json:"Services"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type PolicyDetail struct {
	PolicyName                    string   `json:"PolicyName"`
	PolicyID                      string   `json:"PolicyId"`
	Arn                           string   `json:"Arn"`
	Path                          string   `json:"Path"`
	DefaultVersionID              string   `json:"DefaultVersionId"`
	AttachmentCount               int32    `json:"AttachmentCount"`
	PermissionsBoundaryUsageCount int32    `json:"PermissionsBoundaryUsageCount"`
	IsAttachable                  bool     `json:"IsAttachable"`
	Description                   string   `json:"Description"`
	CreateDate                    string   `json:"CreateDate"`
	UpdateDate                    string   `json:"UpdateDate"`
	Tags                          []IAMTag `json:"Tags"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type RoleDump struct {
	Role             RoleDetail          `json:"Role"`
	AttachedPolicies []AttachedPolicyRef `json:"AttachedPolicies"`
	InlinePolicies   []InlinePolicyDump  `json:"InlinePolicies"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type RoleDetail struct {
	Path                     string   `json:"Path"`
	RoleName                 string   `json:"RoleName"`
	RoleID                   string   `json:"RoleId"`
	Arn                      string   `json:"Arn"`
	CreateDate               string   `json:"CreateDate"`
	AssumeRolePolicyDocument any      `json:"AssumeRolePolicyDocument"`
	Description              string   `json:"Description"`
	MaxSessionDuration       int32    `json:"MaxSessionDuration"`
	Tags                     []IAMTag `json:"Tags"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type IAMTag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type AttachedPolicyRef struct {
	PolicyName string `json:"PolicyName"`
	PolicyArn  string `json:"PolicyArn"`
}

//nolint:tagliatelle // mirrors aws CLI JSON
type InlinePolicyDump struct {
	PolicyName     string   `json:"PolicyName"`
	PolicyDocument any      `json:"PolicyDocument"`
	Services       []string `json:"Services"`
}
