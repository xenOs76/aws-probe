package iam

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/xenos76/aws-probe/internal/awsutil"
	"github.com/xenos76/aws-probe/internal/output"
)

// RolesLister lists IAM roles.
type RolesLister interface {
	ListRoles(
		ctx context.Context,
		params *iam.ListRolesInput,
		optFns ...func(*iam.Options),
	) (*iam.ListRolesOutput, error)
}

// PoliciesLister lists IAM managed policies.
type PoliciesLister interface {
	ListPolicies(
		ctx context.Context,
		params *iam.ListPoliciesInput,
		optFns ...func(*iam.Options),
	) (*iam.ListPoliciesOutput, error)
}

// PolicyDumper retrieves managed policy details and documents.
type PolicyDumper interface {
	PoliciesLister
	GetPolicy(
		ctx context.Context,
		params *iam.GetPolicyInput,
		optFns ...func(*iam.Options),
	) (*iam.GetPolicyOutput, error)
	GetPolicyVersion(
		ctx context.Context,
		params *iam.GetPolicyVersionInput,
		optFns ...func(*iam.Options),
	) (*iam.GetPolicyVersionOutput, error)
}

// RoleDumper retrieves role details, trust policies, and attached/inline policies.
type RoleDumper interface {
	GetRole(
		ctx context.Context,
		params *iam.GetRoleInput,
		optFns ...func(*iam.Options),
	) (*iam.GetRoleOutput, error)
	ListAttachedRolePolicies(
		ctx context.Context,
		params *iam.ListAttachedRolePoliciesInput,
		optFns ...func(*iam.Options),
	) (*iam.ListAttachedRolePoliciesOutput, error)
	ListRolePolicies(
		ctx context.Context,
		params *iam.ListRolePoliciesInput,
		optFns ...func(*iam.Options),
	) (*iam.ListRolePoliciesOutput, error)
	GetRolePolicy(
		ctx context.Context,
		params *iam.GetRolePolicyInput,
		optFns ...func(*iam.Options),
	) (*iam.GetRolePolicyOutput, error)
}

// NewClient creates an IAM client from AWS config.
func NewClient(cfg aws.Config) *iam.Client {
	return iam.NewFromConfig(cfg)
}

// CollectRoles lists all IAM roles.
func CollectRoles(ctx context.Context, api RolesLister) ([]RoleSummary, error) {
	paginator := iam.NewListRolesPaginator(api, &iam.ListRolesInput{})

	var roles []RoleSummary

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing IAM roles: %w", err)
		}

		for _, role := range page.Roles {
			roles = append(roles, RoleSummary{
				RoleName:   awsutil.DerefString(role.RoleName),
				Arn:        awsutil.DerefString(role.Arn),
				Path:       awsutil.DerefString(role.Path),
				CreateDate: formatTime(role.CreateDate),
			})
		}
	}

	return roles, nil
}

// ListRoles lists IAM roles and prints them via the output package.
func ListRoles(
	ctx context.Context,
	api RolesLister,
	out io.Writer,
	format, theme string,
) error {
	roles, err := CollectRoles(ctx, api)
	if err != nil {
		return err
	}

	headers := []string{"Role Name", "Arn", "Path", "Create Date"}
	rows := make([][]string, 0, len(roles))

	for _, r := range roles {
		rows = append(rows, []string{r.RoleName, r.Arn, r.Path, r.CreateDate})
	}

	return output.Print(out, format, theme, output.TableData{
		Headers: headers,
		Rows:    rows,
		Raw:     roles,
	})
}

// CollectPolicies lists IAM managed policies for the given scope (Local, AWS, or All).
func CollectPolicies(ctx context.Context, api PoliciesLister, scope string) ([]PolicySummary, error) {
	policyScope, err := parsePolicyScope(scope)
	if err != nil {
		return nil, err
	}

	paginator := iam.NewListPoliciesPaginator(api, &iam.ListPoliciesInput{
		Scope: policyScope,
	})

	var policies []PolicySummary

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing IAM policies: %w", err)
		}

		for _, p := range page.Policies {
			policies = append(policies, PolicySummary{
				PolicyName:       awsutil.DerefString(p.PolicyName),
				Arn:              awsutil.DerefString(p.Arn),
				DefaultVersionID: awsutil.DerefString(p.DefaultVersionId),
				UpdateDate:       formatTime(p.UpdateDate),
			})
		}
	}

	return policies, nil
}

// ListPolicies lists IAM managed policies for the given scope (Local, AWS, or All).
func ListPolicies(
	ctx context.Context,
	api PoliciesLister,
	scope string,
	out io.Writer,
	format, theme string,
) error {
	policies, err := CollectPolicies(ctx, api, scope)
	if err != nil {
		return err
	}

	headers := []string{"Policy Name", "Arn", "Default Version", "Update Date"}
	rows := make([][]string, 0, len(policies))

	for _, p := range policies {
		rows = append(rows, []string{p.PolicyName, p.Arn, p.DefaultVersionID, p.UpdateDate})
	}

	return output.Print(out, format, theme, output.TableData{
		Headers: headers,
		Rows:    rows,
		Raw:     policies,
	})
}

// CollectPolicyDump retrieves a managed policy document and extracted services.
func CollectPolicyDump(ctx context.Context, api PolicyDumper, nameOrArn string) (PolicyDump, error) {
	policyArn, err := resolvePolicyArn(ctx, api, nameOrArn)
	if err != nil {
		return PolicyDump{}, err
	}

	getOut, err := api.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	})
	if err != nil {
		return PolicyDump{}, fmt.Errorf("getting IAM policy %q: %w", policyArn, err)
	}

	if getOut.Policy == nil {
		return PolicyDump{}, fmt.Errorf("getting IAM policy %q: empty response", policyArn)
	}

	versionID := awsutil.DerefString(getOut.Policy.DefaultVersionId)
	if versionID == "" {
		return PolicyDump{}, fmt.Errorf("policy %q has no default version", policyArn)
	}

	verOut, err := api.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyArn),
		VersionId: aws.String(versionID),
	})
	if err != nil {
		return PolicyDump{}, fmt.Errorf("getting IAM policy version %q: %w", versionID, err)
	}

	if verOut.PolicyVersion == nil || verOut.PolicyVersion.Document == nil {
		return PolicyDump{}, fmt.Errorf("policy version %q has no document", versionID)
	}

	doc, err := decodePolicyDocument(*verOut.PolicyVersion.Document)
	if err != nil {
		return PolicyDump{}, err
	}

	return PolicyDump{
		Policy:   policyDetailFrom(getOut.Policy),
		Document: doc,
		Services: extractServices(doc),
	}, nil
}

// DumpPolicy retrieves a managed policy document and extracted services.
func DumpPolicy(
	ctx context.Context,
	api PolicyDumper,
	nameOrArn string,
	out io.Writer,
	format, theme string,
) error {
	dump, err := CollectPolicyDump(ctx, api, nameOrArn)
	if err != nil {
		return err
	}

	if format == "json" {
		return output.Print(out, format, theme, output.TableData{Raw: dump})
	}

	return printPolicyDumpTable(out, format, theme, dump)
}

func policyDetailFrom(p *types.Policy) PolicyDetail {
	var attachmentCount int32
	if p.AttachmentCount != nil {
		attachmentCount = *p.AttachmentCount
	}

	var permissionsBoundaryUsageCount int32
	if p.PermissionsBoundaryUsageCount != nil {
		permissionsBoundaryUsageCount = *p.PermissionsBoundaryUsageCount
	}

	tags := make([]IAMTag, 0, len(p.Tags))
	for _, t := range p.Tags {
		tags = append(tags, IAMTag{
			Key:   awsutil.DerefString(t.Key),
			Value: awsutil.DerefString(t.Value),
		})
	}

	return PolicyDetail{
		PolicyName:                    awsutil.DerefString(p.PolicyName),
		PolicyID:                      awsutil.DerefString(p.PolicyId),
		Arn:                           awsutil.DerefString(p.Arn),
		Path:                          awsutil.DerefString(p.Path),
		DefaultVersionID:              awsutil.DerefString(p.DefaultVersionId),
		AttachmentCount:               attachmentCount,
		PermissionsBoundaryUsageCount: permissionsBoundaryUsageCount,
		IsAttachable:                  p.IsAttachable,
		Description:                   awsutil.DerefString(p.Description),
		CreateDate:                    formatAWSCLITime(p.CreateDate),
		UpdateDate:                    formatAWSCLITime(p.UpdateDate),
		Tags:                          tags,
	}
}

func printPolicyDumpTable(out io.Writer, format, theme string, dump PolicyDump) error {
	headers := []string{"Name", "Arn", "Default Version", "Services"}
	rows := [][]string{{
		dump.Policy.PolicyName,
		dump.Policy.Arn,
		dump.Policy.DefaultVersionID,
		strings.Join(dump.Services, ", "),
	}}

	return output.Print(out, format, theme, output.TableData{
		Headers: headers,
		Rows:    rows,
		Raw:     dump,
	})
}

// CollectRoleDump retrieves role trust policy and attached/inline policies.
func CollectRoleDump(ctx context.Context, api RoleDumper, nameOrArn string) (RoleDump, error) {
	roleName := roleNameFromInput(nameOrArn)

	getOut, err := api.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return RoleDump{}, fmt.Errorf("getting IAM role %q: %w", roleName, err)
	}

	if getOut.Role == nil {
		return RoleDump{}, fmt.Errorf("getting IAM role %q: empty response", roleName)
	}

	role := getOut.Role

	var trust any
	if role.AssumeRolePolicyDocument != nil {
		trust, err = decodePolicyDocument(*role.AssumeRolePolicyDocument)
		if err != nil {
			return RoleDump{}, fmt.Errorf("decoding trust policy for role %q: %w", roleName, err)
		}
	}

	attached, err := listAttachedPolicies(ctx, api, roleName)
	if err != nil {
		return RoleDump{}, err
	}

	inline, err := listInlinePolicies(ctx, api, roleName)
	if err != nil {
		return RoleDump{}, err
	}

	var maxSession int32
	if role.MaxSessionDuration != nil {
		maxSession = *role.MaxSessionDuration
	}

	tags := make([]IAMTag, 0, len(role.Tags))
	for _, t := range role.Tags {
		tags = append(tags, IAMTag{
			Key:   awsutil.DerefString(t.Key),
			Value: awsutil.DerefString(t.Value),
		})
	}

	return RoleDump{
		Role: RoleDetail{
			Path:                     awsutil.DerefString(role.Path),
			RoleName:                 awsutil.DerefString(role.RoleName),
			RoleID:                   awsutil.DerefString(role.RoleId),
			Arn:                      awsutil.DerefString(role.Arn),
			CreateDate:               formatAWSCLITime(role.CreateDate),
			AssumeRolePolicyDocument: trust,
			Description:              awsutil.DerefString(role.Description),
			MaxSessionDuration:       maxSession,
			Tags:                     tags,
		},
		AttachedPolicies: attached,
		InlinePolicies:   inline,
	}, nil
}

// DumpRole retrieves role trust policy and attached/inline policies.
func DumpRole(
	ctx context.Context,
	api RoleDumper,
	nameOrArn string,
	out io.Writer,
	format, theme string,
) error {
	dump, err := CollectRoleDump(ctx, api, nameOrArn)
	if err != nil {
		return err
	}

	if format == "json" {
		return output.Print(out, format, theme, output.TableData{Raw: dump})
	}

	return printRoleDumpTable(out, format, theme, dump)
}

func printRoleDumpTable(out io.Writer, format, theme string, dump RoleDump) error {
	headers := []string{"Section", "Name", "Detail", "Extra"}
	rows := [][]string{
		{"Role", dump.Role.RoleName, dump.Role.Arn, dump.Role.Path},
		{"CreateDate", dump.Role.CreateDate, "", ""},
	}

	for _, stmt := range trustStatementRows(dump.Role.AssumeRolePolicyDocument) {
		rows = append(rows, []string{
			"Trust",
			stmt[0], // Effect
			stmt[1], // Principal
			stmt[2], // Action
		})
	}

	for _, p := range dump.AttachedPolicies {
		rows = append(rows, []string{"Attached", p.PolicyName, p.PolicyArn, ""})
	}

	for _, p := range dump.InlinePolicies {
		rows = append(rows, []string{
			"Inline",
			p.PolicyName,
			strings.Join(p.Services, ", "),
			"",
		})
	}

	return output.Print(out, format, theme, output.TableData{
		Headers: headers,
		Rows:    rows,
		Raw:     dump,
	})
}

func listAttachedPolicies(
	ctx context.Context,
	api RoleDumper,
	roleName string,
) ([]AttachedPolicyRef, error) {
	paginator := iam.NewListAttachedRolePoliciesPaginator(api, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})

	var attached []AttachedPolicyRef

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing attached policies for role %q: %w", roleName, err)
		}

		for _, p := range page.AttachedPolicies {
			attached = append(attached, AttachedPolicyRef{
				PolicyName: awsutil.DerefString(p.PolicyName),
				PolicyArn:  awsutil.DerefString(p.PolicyArn),
			})
		}
	}

	return attached, nil
}

func listInlinePolicies(
	ctx context.Context,
	api RoleDumper,
	roleName string,
) ([]InlinePolicyDump, error) {
	paginator := iam.NewListRolePoliciesPaginator(api, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})

	var names []string

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing inline policies for role %q: %w", roleName, err)
		}

		names = append(names, page.PolicyNames...)
	}

	inline := make([]InlinePolicyDump, 0, len(names))

	for _, name := range names {
		polOut, err := api.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
			RoleName:   aws.String(roleName),
			PolicyName: aws.String(name),
		})
		if err != nil {
			return nil, fmt.Errorf("getting inline policy %q on role %q: %w", name, roleName, err)
		}

		var doc any
		if polOut.PolicyDocument != nil {
			doc, err = decodePolicyDocument(*polOut.PolicyDocument)
			if err != nil {
				return nil, fmt.Errorf("decoding inline policy %q: %w", name, err)
			}
		}

		inline = append(inline, InlinePolicyDump{
			PolicyName:     name,
			PolicyDocument: doc,
			Services:       extractServices(doc),
		})
	}

	return inline, nil
}

func resolvePolicyArn(ctx context.Context, api PolicyDumper, nameOrArn string) (string, error) {
	if strings.HasPrefix(nameOrArn, "arn:") {
		return nameOrArn, nil
	}

	paginator := iam.NewListPoliciesPaginator(api, &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeAll,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("resolving policy name %q: %w", nameOrArn, err)
		}

		for _, p := range page.Policies {
			if awsutil.DerefString(p.PolicyName) == nameOrArn {
				arn := awsutil.DerefString(p.Arn)
				if arn == "" {
					return "", fmt.Errorf("policy %q has empty ARN", nameOrArn)
				}

				return arn, nil
			}
		}
	}

	return "", fmt.Errorf("policy %q not found", nameOrArn)
}

func roleNameFromInput(nameOrArn string) string {
	if !strings.HasPrefix(nameOrArn, "arn:") {
		return nameOrArn
	}

	// arn:aws:iam::ACCOUNT:role[/path]/name
	const marker = ":role/"

	idx := strings.Index(nameOrArn, marker)
	if idx < 0 {
		return nameOrArn
	}

	rest := nameOrArn[idx+len(marker):]
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		return rest[i+1:]
	}

	return rest
}

func parsePolicyScope(scope string) (types.PolicyScopeType, error) {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", "local":
		return types.PolicyScopeTypeLocal, nil
	case "aws":
		return types.PolicyScopeTypeAws, nil
	case "all":
		return types.PolicyScopeTypeAll, nil
	default:
		return "", fmt.Errorf("invalid policy scope %q: use Local, AWS, or All", scope)
	}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}

// formatAWSCLITime matches aws CLI JSON timestamps (e.g. 2026-07-20T13:27:28+00:00).
func formatAWSCLITime(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.UTC().Format("2006-01-02T15:04:05-07:00")
}
