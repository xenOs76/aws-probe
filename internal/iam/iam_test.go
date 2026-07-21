package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockIAM struct {
	mock.Mock
}

func (m *mockIAM) ListRoles(
	ctx context.Context,
	params *iam.ListRolesInput,
	_ ...func(*iam.Options),
) (*iam.ListRolesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.ListRolesOutput
	if val, ok := args.Get(0).(*iam.ListRolesOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) ListPolicies(
	ctx context.Context,
	params *iam.ListPoliciesInput,
	_ ...func(*iam.Options),
) (*iam.ListPoliciesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.ListPoliciesOutput
	if val, ok := args.Get(0).(*iam.ListPoliciesOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) GetPolicy(
	ctx context.Context,
	params *iam.GetPolicyInput,
	_ ...func(*iam.Options),
) (*iam.GetPolicyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.GetPolicyOutput
	if val, ok := args.Get(0).(*iam.GetPolicyOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) GetPolicyVersion(
	ctx context.Context,
	params *iam.GetPolicyVersionInput,
	_ ...func(*iam.Options),
) (*iam.GetPolicyVersionOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.GetPolicyVersionOutput
	if val, ok := args.Get(0).(*iam.GetPolicyVersionOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) GetRole(
	ctx context.Context,
	params *iam.GetRoleInput,
	_ ...func(*iam.Options),
) (*iam.GetRoleOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.GetRoleOutput
	if val, ok := args.Get(0).(*iam.GetRoleOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) ListAttachedRolePolicies(
	ctx context.Context,
	params *iam.ListAttachedRolePoliciesInput,
	_ ...func(*iam.Options),
) (*iam.ListAttachedRolePoliciesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.ListAttachedRolePoliciesOutput
	if val, ok := args.Get(0).(*iam.ListAttachedRolePoliciesOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) ListRolePolicies(
	ctx context.Context,
	params *iam.ListRolePoliciesInput,
	_ ...func(*iam.Options),
) (*iam.ListRolePoliciesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.ListRolePoliciesOutput
	if val, ok := args.Get(0).(*iam.ListRolePoliciesOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *mockIAM) GetRolePolicy(
	ctx context.Context,
	params *iam.GetRolePolicyInput,
	_ ...func(*iam.Options),
) (*iam.GetRolePolicyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *iam.GetRolePolicyOutput
	if val, ok := args.Get(0).(*iam.GetRolePolicyOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func encodeDoc(t *testing.T, doc string) *string {
	t.Helper()

	encoded := url.QueryEscape(doc)

	return &encoded
}

func TestExtractServices(t *testing.T) {
	doc := map[string]any{
		"Version": "2012-10-17",
		"Statement": []any{
			map[string]any{
				"Effect": "Allow",
				"Action": []any{"s3:GetObject", "s3:ListBucket", "iam:PassRole"},
			},
			map[string]any{
				"Effect":    "Deny",
				"NotAction": "ec2:*",
			},
			map[string]any{
				"Effect": "Allow",
				"Action": "*",
			},
		},
	}

	assert.Equal(t, []string{"*", "ec2", "iam", "s3"}, extractServices(doc))
}

func TestDecodePolicyDocument(t *testing.T) {
	raw := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`
	encoded := url.QueryEscape(raw)

	doc, err := decodePolicyDocument(encoded)
	require.NoError(t, err)

	m, ok := doc.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2012-10-17", m["Version"])
	assert.Equal(t, []string{"s3"}, extractServices(doc))
}

func TestDecodePolicyDocument_Invalid(t *testing.T) {
	_, err := decodePolicyDocument("%zz")
	require.Error(t, err)

	_, err = decodePolicyDocument(url.QueryEscape("not-json"))
	require.Error(t, err)
}

func TestListRoles_Pagination(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	api.On("ListRoles", ctx, mock.MatchedBy(func(in *iam.ListRolesInput) bool {
		return in.Marker == nil
	})).Return(&iam.ListRolesOutput{
		Roles: []types.Role{{
			RoleName:   aws.String("role-a"),
			Arn:        aws.String("arn:aws:iam::1:role/role-a"),
			Path:       aws.String("/"),
			CreateDate: &created,
		}},
		IsTruncated: true,
		Marker:      aws.String("next"),
	}, nil).Once()

	api.On("ListRoles", ctx, mock.MatchedBy(func(in *iam.ListRolesInput) bool {
		return in.Marker != nil && *in.Marker == "next"
	})).Return(&iam.ListRolesOutput{
		Roles: []types.Role{{
			RoleName:   aws.String("role-b"),
			Arn:        aws.String("arn:aws:iam::1:role/role-b"),
			Path:       aws.String("/app/"),
			CreateDate: &created,
		}},
		IsTruncated: false,
	}, nil).Once()

	var buf bytes.Buffer

	err := ListRoles(ctx, api, &buf, "json", "none")
	require.NoError(t, err)

	var roles []RoleSummary
	require.NoError(t, json.Unmarshal(buf.Bytes(), &roles))
	require.Len(t, roles, 2)
	assert.Equal(t, "role-a", roles[0].RoleName)
	assert.Equal(t, "role-b", roles[1].RoleName)
	assert.Equal(t, "2024-01-02T03:04:05Z", roles[0].CreateDate)
	api.AssertExpectations(t)
}

func TestListPolicies_Scope(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	updated := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	api.On("ListPolicies", ctx, mock.MatchedBy(func(in *iam.ListPoliciesInput) bool {
		return in.Scope == types.PolicyScopeTypeLocal
	})).Return(&iam.ListPoliciesOutput{
		Policies: []types.Policy{{
			PolicyName:       aws.String("aws-probe-s3-read-only"),
			Arn:              aws.String("arn:aws:iam::1:policy/aws-probe-s3-read-only"),
			DefaultVersionId: aws.String("v1"),
			UpdateDate:       &updated,
		}},
	}, nil).Once()

	var buf bytes.Buffer

	err := ListPolicies(ctx, api, "Local", &buf, "json", "none")
	require.NoError(t, err)

	var policies []PolicySummary
	require.NoError(t, json.Unmarshal(buf.Bytes(), &policies))
	require.Len(t, policies, 1)
	assert.Equal(t, "aws-probe-s3-read-only", policies[0].PolicyName)
	api.AssertExpectations(t)
}

func TestListPolicies_InvalidScope(t *testing.T) {
	err := ListPolicies(context.Background(), new(mockIAM), "bogus", &bytes.Buffer{}, "json", "none")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid policy scope")
}

func TestDumpPolicy_ByName(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	arn := "arn:aws:iam::1:policy/aws-probe-s3-read-only"
	created := time.Date(2026, 7, 20, 13, 27, 28, 0, time.UTC)
	doc := `{"Version":"2012-10-17","Statement":[{` +
		`"Effect":"Allow","Action":["s3:ListBucket","s3:GetObject"],"Resource":"*"}]}`

	api.On("ListPolicies", ctx, mock.MatchedBy(func(in *iam.ListPoliciesInput) bool {
		return in.Scope == types.PolicyScopeTypeAll
	})).Return(&iam.ListPoliciesOutput{
		Policies: []types.Policy{{
			PolicyName:       aws.String("aws-probe-s3-read-only"),
			Arn:              aws.String(arn),
			DefaultVersionId: aws.String("v1"),
		}},
	}, nil).Once()

	api.On("GetPolicy", ctx, mock.MatchedBy(func(in *iam.GetPolicyInput) bool {
		return aws.ToString(in.PolicyArn) == arn
	})).Return(&iam.GetPolicyOutput{
		Policy: &types.Policy{
			PolicyName:                    aws.String("aws-probe-s3-read-only"),
			PolicyId:                      aws.String("ANPA9BB64EE2A35B47BFA"),
			Arn:                           aws.String(arn),
			Path:                          aws.String("/"),
			DefaultVersionId:              aws.String("v1"),
			AttachmentCount:               aws.Int32(1),
			PermissionsBoundaryUsageCount: aws.Int32(0),
			IsAttachable:                  true,
			CreateDate:                    &created,
			UpdateDate:                    &created,
			Tags: []types.Tag{{
				Key: aws.String("Project"), Value: aws.String("aws-probe"),
			}},
		},
	}, nil).Once()

	api.On("GetPolicyVersion", ctx, mock.MatchedBy(func(in *iam.GetPolicyVersionInput) bool {
		return aws.ToString(in.PolicyArn) == arn && aws.ToString(in.VersionId) == "v1"
	})).Return(&iam.GetPolicyVersionOutput{
		PolicyVersion: &types.PolicyVersion{Document: encodeDoc(t, doc)},
	}, nil).Once()

	var buf bytes.Buffer
	require.NoError(t, DumpPolicy(ctx, api, "aws-probe-s3-read-only", &buf, "json", "none"))

	var dump PolicyDump
	require.NoError(t, json.Unmarshal(buf.Bytes(), &dump))
	assert.Equal(t, "aws-probe-s3-read-only", dump.Policy.PolicyName)
	assert.Equal(t, "ANPA9BB64EE2A35B47BFA", dump.Policy.PolicyID)
	assert.Equal(t, arn, dump.Policy.Arn)
	assert.Equal(t, "/", dump.Policy.Path)
	assert.Equal(t, "v1", dump.Policy.DefaultVersionID)
	assert.Equal(t, int32(1), dump.Policy.AttachmentCount)
	assert.True(t, dump.Policy.IsAttachable)
	assert.Equal(t, "2026-07-20T13:27:28+00:00", dump.Policy.CreateDate)
	assert.Equal(t, "2026-07-20T13:27:28+00:00", dump.Policy.UpdateDate)
	require.Len(t, dump.Policy.Tags, 1)
	assert.Equal(t, "Project", dump.Policy.Tags[0].Key)
	assert.Equal(t, "aws-probe", dump.Policy.Tags[0].Value)
	assert.NotNil(t, dump.Document)
	assert.Equal(t, []string{"s3"}, dump.Services)
	api.AssertExpectations(t)
}

func TestDumpRole(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	created := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	trust := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow",` +
		`"Principal":{"AWS":"arn:aws:iam::1:user/aws-probe-assumer"},"Action":"sts:AssumeRole"}]}`
	inline := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow",` +
		`"Action":"secretsmanager:GetSecretValue","Resource":"*"}]}`

	api.On("GetRole", ctx, mock.MatchedBy(func(in *iam.GetRoleInput) bool {
		return aws.ToString(in.RoleName) == "aws-probe-s3-read-only"
	})).Return(&iam.GetRoleOutput{
		Role: &types.Role{
			RoleName:                 aws.String("aws-probe-s3-read-only"),
			RoleId:                   aws.String("AROAEXAMPLEID"),
			Arn:                      aws.String("arn:aws:iam::1:role/aws-probe-s3-read-only"),
			Path:                     aws.String("/"),
			CreateDate:               &created,
			AssumeRolePolicyDocument: encodeDoc(t, trust),
			Description:              aws.String(""),
			MaxSessionDuration:       aws.Int32(3600),
			Tags: []types.Tag{{
				Key: aws.String("Project"), Value: aws.String("aws-probe"),
			}},
		},
	}, nil).Once()

	api.On("ListAttachedRolePolicies", ctx, mock.Anything).Return(&iam.ListAttachedRolePoliciesOutput{
		AttachedPolicies: []types.AttachedPolicy{{
			PolicyName: aws.String("aws-probe-s3-read-only"),
			PolicyArn:  aws.String("arn:aws:iam::1:policy/aws-probe-s3-read-only"),
		}},
	}, nil).Once()

	api.On("ListRolePolicies", ctx, mock.Anything).Return(&iam.ListRolePoliciesOutput{
		PolicyNames: []string{"inline-extra"},
	}, nil).Once()

	api.On("GetRolePolicy", ctx, mock.MatchedBy(func(in *iam.GetRolePolicyInput) bool {
		return aws.ToString(in.RoleName) == "aws-probe-s3-read-only" &&
			aws.ToString(in.PolicyName) == "inline-extra"
	})).Return(&iam.GetRolePolicyOutput{
		PolicyName: aws.String("inline-extra"), PolicyDocument: encodeDoc(t, inline),
	}, nil).Once()

	var buf bytes.Buffer
	require.NoError(t, DumpRole(ctx, api, "arn:aws:iam::1:role/aws-probe-s3-read-only", &buf, "json", "none"))

	var dump RoleDump
	require.NoError(t, json.Unmarshal(buf.Bytes(), &dump))
	assert.Equal(t, "aws-probe-s3-read-only", dump.Role.RoleName)
	assert.Equal(t, "AROAEXAMPLEID", dump.Role.RoleID)
	assert.Equal(t, "2024-03-01T12:00:00+00:00", dump.Role.CreateDate)
	assert.Equal(t, int32(3600), dump.Role.MaxSessionDuration)
	require.Len(t, dump.Role.Tags, 1)
	assert.Equal(t, "Project", dump.Role.Tags[0].Key)
	assert.Equal(t, "aws-probe", dump.Role.Tags[0].Value)
	assert.NotNil(t, dump.Role.AssumeRolePolicyDocument)
	require.Len(t, dump.AttachedPolicies, 1)
	assert.Equal(t, "aws-probe-s3-read-only", dump.AttachedPolicies[0].PolicyName)
	assert.Equal(t, "arn:aws:iam::1:policy/aws-probe-s3-read-only", dump.AttachedPolicies[0].PolicyArn)
	require.Len(t, dump.InlinePolicies, 1)
	assert.Equal(t, "inline-extra", dump.InlinePolicies[0].PolicyName)
	assert.Equal(t, []string{"secretsmanager"}, dump.InlinePolicies[0].Services)
	assert.NotNil(t, dump.InlinePolicies[0].PolicyDocument)
	api.AssertExpectations(t)
}

func TestRoleNameFromInput(t *testing.T) {
	assert.Equal(t, "my-role", roleNameFromInput("my-role"))
	assert.Equal(t, "my-role", roleNameFromInput("arn:aws:iam::123:role/my-role"))
	assert.Equal(t, "foo", roleNameFromInput("arn:aws:iam::123:role/service-role/foo"))
}

func TestParsePolicyScope(t *testing.T) {
	s, err := parsePolicyScope("local")
	require.NoError(t, err)
	assert.Equal(t, types.PolicyScopeTypeLocal, s)

	s, err = parsePolicyScope("")
	require.NoError(t, err)
	assert.Equal(t, types.PolicyScopeTypeLocal, s)

	s, err = parsePolicyScope("AWS")
	require.NoError(t, err)
	assert.Equal(t, types.PolicyScopeTypeAws, s)

	s, err = parsePolicyScope("All")
	require.NoError(t, err)
	assert.Equal(t, types.PolicyScopeTypeAll, s)
}

func TestFormatPrincipalAndActions(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatPrincipal(nil))
	assert.Equal(t, "*", formatPrincipal("*"))
	assert.Equal(t, "AWS=arn:aws:iam::1:root; Service=ec2.amazonaws.com",
		formatPrincipal(map[string]any{
			"Service": "ec2.amazonaws.com",
			"AWS":     "arn:aws:iam::1:root",
		}))
	assert.Equal(t, "a, b", formatPrincipal([]any{"a", "b"}))
	assert.Equal(t, "42", formatPrincipal(42))

	assert.Empty(t, formatActions(nil))
	assert.Equal(t, "sts:AssumeRole", formatActions("sts:AssumeRole"))
	assert.Equal(t, "a, b", formatActions([]any{"a", "b"}))
	assert.Equal(t, "7", formatActions(7))
}

func TestTrustStatementRows(t *testing.T) {
	t.Parallel()

	assert.Nil(t, trustStatementRows(nil))
	assert.Nil(t, trustStatementRows("not-a-map"))
	assert.Nil(t, trustStatementRows(map[string]any{"Version": "2012-10-17"}))
	assert.Nil(t, trustStatementRows(map[string]any{"Statement": "bad"}))

	rows := trustStatementRows(map[string]any{
		"Statement": map[string]any{
			"Effect":    "Allow",
			"Principal": map[string]any{"AWS": "arn:aws:iam::1:root"},
			"Action":    "sts:AssumeRole",
		},
	})
	require.Len(t, rows, 1)
	assert.Equal(t, []string{"Allow", "AWS=arn:aws:iam::1:root", "sts:AssumeRole", ""}, rows[0])

	rows = trustStatementRows(map[string]any{
		"Statement": []any{
			map[string]any{
				"Effect":    "Allow",
				"Principal": "*",
				"Action":    []any{"sts:AssumeRole", "sts:TagSession"},
			},
			"skip-me",
		},
	})
	require.Len(t, rows, 1)
	assert.Equal(t, "Allow", rows[0][0])
	assert.Equal(t, "*", rows[0][1])
	assert.Equal(t, "sts:AssumeRole, sts:TagSession", rows[0][2])
}

func TestFormatTimeHelpers(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatTime(nil))
	assert.Empty(t, formatAWSCLITime(nil))

	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	assert.Equal(t, "2024-01-02T03:04:05Z", formatTime(&ts))
	assert.Equal(t, "2024-01-02T03:04:05+00:00", formatAWSCLITime(&ts))
}

func TestListRoles_Error(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	api.On("ListRoles", ctx, mock.Anything).Return(nil, assert.AnError).Once()

	err := ListRoles(ctx, api, &bytes.Buffer{}, "json", "none")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing IAM roles")
	api.AssertExpectations(t)
}

func TestListPolicies_Error(t *testing.T) {
	ctx := context.Background()
	api := new(mockIAM)
	api.On("ListPolicies", ctx, mock.Anything).Return(nil, assert.AnError).Once()

	err := ListPolicies(ctx, api, "Local", &bytes.Buffer{}, "json", "none")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing IAM policies")
	api.AssertExpectations(t)
}

func TestDumpPolicy_ByARN_TableAndCSV(t *testing.T) {
	ctx := context.Background()
	arn := "arn:aws:iam::1:policy/aws-probe-s3-read-only"
	doc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`

	setup := func(t *testing.T) *mockIAM {
		t.Helper()

		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.MatchedBy(func(in *iam.GetPolicyInput) bool {
			return aws.ToString(in.PolicyArn) == arn
		})).Return(&iam.GetPolicyOutput{
			Policy: &types.Policy{
				PolicyName:       aws.String("aws-probe-s3-read-only"),
				Arn:              aws.String(arn),
				DefaultVersionId: aws.String("v1"),
			},
		}, nil).Once()
		api.On("GetPolicyVersion", ctx, mock.Anything).Return(&iam.GetPolicyVersionOutput{
			PolicyVersion: &types.PolicyVersion{Document: encodeDoc(t, doc)},
		}, nil).Once()

		return api
	}

	t.Run("table", func(t *testing.T) {
		api := setup(t)

		var buf bytes.Buffer

		require.NoError(t, DumpPolicy(ctx, api, arn, &buf, "table", "none"))
		assert.Contains(t, buf.String(), "aws-probe-s3-read-only")
		assert.Contains(t, buf.String(), "s3")
		api.AssertExpectations(t)
	})

	t.Run("csv", func(t *testing.T) {
		api := setup(t)

		var buf bytes.Buffer

		require.NoError(t, DumpPolicy(ctx, api, arn, &buf, "csv", "none"))
		assert.Contains(t, buf.String(), "aws-probe-s3-read-only")
		assert.Contains(t, buf.String(), arn)
		api.AssertExpectations(t)
	})
}

func TestDumpPolicy_Errors(t *testing.T) { //nolint:revive // table of error cases
	ctx := context.Background()
	arn := "arn:aws:iam::1:policy/p"

	t.Run("not found by name", func(t *testing.T) {
		api := new(mockIAM)
		api.On("ListPolicies", ctx, mock.Anything).Return(&iam.ListPoliciesOutput{
			Policies: []types.Policy{{PolicyName: aws.String("other"), Arn: aws.String(arn)}},
		}, nil).Once()
		err := DumpPolicy(ctx, api, "missing", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		api.AssertExpectations(t)
	})

	t.Run("empty ARN on match", func(t *testing.T) {
		api := new(mockIAM)
		api.On("ListPolicies", ctx, mock.Anything).Return(&iam.ListPoliciesOutput{
			Policies: []types.Policy{{PolicyName: aws.String("p"), Arn: aws.String("")}},
		}, nil).Once()
		err := DumpPolicy(ctx, api, "p", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty ARN")
		api.AssertExpectations(t)
	})

	t.Run("resolve list error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("ListPolicies", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpPolicy(ctx, api, "p", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolving policy name")
		api.AssertExpectations(t)
	})

	t.Run("get policy error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpPolicy(ctx, api, arn, &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting IAM policy")
		api.AssertExpectations(t)
	})

	t.Run("nil policy", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.Anything).Return(&iam.GetPolicyOutput{}, nil).Once()
		err := DumpPolicy(ctx, api, arn, &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty response")
		api.AssertExpectations(t)
	})

	t.Run("no default version", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.Anything).Return(&iam.GetPolicyOutput{
			Policy: &types.Policy{Arn: aws.String(arn)},
		}, nil).Once()
		err := DumpPolicy(ctx, api, arn, &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no default version")
		api.AssertExpectations(t)
	})

	t.Run("get version error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.Anything).Return(&iam.GetPolicyOutput{
			Policy: &types.Policy{Arn: aws.String(arn), DefaultVersionId: aws.String("v1")},
		}, nil).Once()
		api.On("GetPolicyVersion", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpPolicy(ctx, api, arn, &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting IAM policy version")
		api.AssertExpectations(t)
	})

	t.Run("nil document", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetPolicy", ctx, mock.Anything).Return(&iam.GetPolicyOutput{
			Policy: &types.Policy{Arn: aws.String(arn), DefaultVersionId: aws.String("v1")},
		}, nil).Once()
		api.On("GetPolicyVersion", ctx, mock.Anything).Return(&iam.GetPolicyVersionOutput{
			PolicyVersion: &types.PolicyVersion{},
		}, nil).Once()
		err := DumpPolicy(ctx, api, arn, &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no document")
		api.AssertExpectations(t)
	})
}

func TestDumpRole_TableAndErrors(t *testing.T) { //nolint:revive // table of error cases
	ctx := context.Background()
	trust := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow",` +
		`"Principal":{"AWS":"arn:aws:iam::1:root"},"Action":"sts:AssumeRole"}]}`

	t.Run("table", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{
			Role: &types.Role{
				RoleName:                 aws.String("r"),
				Arn:                      aws.String("arn:aws:iam::1:role/r"),
				Path:                     aws.String("/"),
				AssumeRolePolicyDocument: encodeDoc(t, trust),
			},
		}, nil).Once()
		api.On("ListAttachedRolePolicies", ctx, mock.Anything).Return(&iam.ListAttachedRolePoliciesOutput{
			AttachedPolicies: []types.AttachedPolicy{{
				PolicyName: aws.String("p"), PolicyArn: aws.String("arn:aws:iam::1:policy/p"),
			}},
		}, nil).Once()
		api.On("ListRolePolicies", ctx, mock.Anything).Return(&iam.ListRolePoliciesOutput{}, nil).Once()

		var buf bytes.Buffer
		require.NoError(t, DumpRole(ctx, api, "r", &buf, "table", "none"))
		assert.Contains(t, buf.String(), "Trust")
		assert.Contains(t, buf.String(), "Attached")
		api.AssertExpectations(t)
	})

	t.Run("get role error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting IAM role")
		api.AssertExpectations(t)
	})

	t.Run("nil role", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{}, nil).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty response")
		api.AssertExpectations(t)
	})

	t.Run("bad trust encoding", func(t *testing.T) {
		api := new(mockIAM)
		bad := "%zz"
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{
			Role: &types.Role{
				RoleName:                 aws.String("r"),
				AssumeRolePolicyDocument: &bad,
			},
		}, nil).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding trust policy")
		api.AssertExpectations(t)
	})

	t.Run("attached list error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{
			Role: &types.Role{RoleName: aws.String("r")},
		}, nil).Once()
		api.On("ListAttachedRolePolicies", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing attached policies")
		api.AssertExpectations(t)
	})

	t.Run("inline list error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{
			Role: &types.Role{RoleName: aws.String("r")},
		}, nil).Once()
		api.On("ListAttachedRolePolicies", ctx, mock.Anything).
			Return(&iam.ListAttachedRolePoliciesOutput{}, nil).Once()
		api.On("ListRolePolicies", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing inline policies")
		api.AssertExpectations(t)
	})

	t.Run("get inline error", func(t *testing.T) {
		api := new(mockIAM)
		api.On("GetRole", ctx, mock.Anything).Return(&iam.GetRoleOutput{
			Role: &types.Role{RoleName: aws.String("r")},
		}, nil).Once()
		api.On("ListAttachedRolePolicies", ctx, mock.Anything).
			Return(&iam.ListAttachedRolePoliciesOutput{}, nil).Once()
		api.On("ListRolePolicies", ctx, mock.Anything).Return(&iam.ListRolePoliciesOutput{
			PolicyNames: []string{"inline"},
		}, nil).Once()
		api.On("GetRolePolicy", ctx, mock.Anything).Return(nil, assert.AnError).Once()
		err := DumpRole(ctx, api, "r", &bytes.Buffer{}, "json", "none")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting inline policy")
		api.AssertExpectations(t)
	})
}

func TestDecodePolicyDocument_Empty(t *testing.T) {
	t.Parallel()

	_, err := decodePolicyDocument("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty policy document")
}
