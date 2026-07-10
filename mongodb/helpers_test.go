package mongodb

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"pgregory.net/rapid"
)

// TestValidateAuthMechanism_Valid — "MONGODB-AWS" and "" return no errors.
func TestValidateAuthMechanism_Valid(t *testing.T) {
	for _, v := range []string{"MONGODB-AWS", ""} {
		diags := validateAuthMechanism(v, cty.Path{})
		if len(diags) != 0 {
			t.Errorf("expected no diagnostics for %q, got: %v", v, diags)
		}
	}
}

// TestValidateAuthMechanism_Invalid — arbitrary non-MONGODB-AWS strings return errors.
func TestValidateAuthMechanism_Invalid(t *testing.T) {
	for _, v := range []string{"SCRAM-SHA-256", "mongodb-aws", "AWS", "plain", "x509"} {
		diags := validateAuthMechanism(v, cty.Path{})
		if len(diags) == 0 {
			t.Errorf("expected error diagnostic for %q, got none", v)
		}
	}
}

// TestValidateIAMARN_Valid — well-formed user and role ARNs return no errors.
func TestValidateIAMARN_Valid(t *testing.T) {
	valid := []string{
		"arn:aws:iam::123456789012:user/alice",
		"arn:aws:iam::000000000000:role/MyRole",
		"arn:aws:iam::999999999999:user/path/to/user",
		"arn:aws:iam::123456789012:role/service-role/MyLambdaRole",
		"arn:aws:iam::123456789012:user/user+name@example.com",
	}
	for _, v := range valid {
		diags := validateIAMARN(v, cty.Path{})
		if len(diags) != 0 {
			t.Errorf("expected no diagnostics for %q, got: %v", v, diags)
		}
	}
}

// TestValidateIAMARN_Invalid — malformed strings return errors.
func TestValidateIAMARN_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"arn:aws:iam::12345:user/alice",          // account ID too short
		"arn:aws:iam::123456789012:group/admins", // unsupported type
		"arn:aws:s3:::my-bucket",                 // wrong service
		"arn:aws:iam::123456789012:user/",        // empty name
		"123456789012:user/alice",                // missing arn prefix
		"arn:aws:iam::123456789012:user",         // missing slash+name
	}
	for _, v := range invalid {
		diags := validateIAMARN(v, cty.Path{})
		if len(diags) == 0 {
			t.Errorf("expected error diagnostic for %q, got none", v)
		}
	}
}

// ---------------------------------------------------------------------------
// Property-based tests
// ---------------------------------------------------------------------------

// Feature: documentdb-iam-auth, Property 1: invalid auth_mechanism values are always rejected
// Validates: Requirements 1.4
func TestProperty_InvalidAuthMechanismAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings that are neither "MONGODB-AWS" nor empty.
		val := rapid.StringMatching(`[A-Za-z0-9_\-]{1,50}`).Filter(func(s string) bool {
			return s != "MONGODB-AWS" && s != ""
		}).Draw(t, "auth_mechanism")

		diags := validateAuthMechanism(val, cty.Path{})
		if len(diags) == 0 {
			t.Fatalf("expected at least one error diagnostic for %q, got none", val)
		}
	})
}

// Feature: documentdb-iam-auth, Property 2: valid IAM ARNs always pass ARN validation
// Validates: Requirements 8.1, 8.3
func TestProperty_ValidIAMARNAlwaysPasses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		accountID := rapid.StringMatching(`\d{12}`).Draw(t, "account_id")
		iamType := rapid.SampledFrom([]string{"user", "role"}).Draw(t, "iam_type")
		name := rapid.StringMatching(`[\w][\w+=,.@/-]*`).Draw(t, "name")

		arn := fmt.Sprintf("arn:aws:iam::%s:%s/%s", accountID, iamType, name)
		diags := validateIAMARN(arn, cty.Path{})
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics for valid ARN %q, got: %v", arn, diags)
		}
	})
}

// Feature: documentdb-iam-auth, Property 3: invalid strings always fail ARN validation
// Validates: Requirements 8.1, 8.2, 8.4
func TestProperty_InvalidARNAlwaysFails(t *testing.T) {
	arnPattern := regexp.MustCompile(`^arn:aws:iam::\d{12}:(user|role)/[\w+=,.@/-]+$`)

	rapid.Check(t, func(t *rapid.T) {
		val := rapid.String().Filter(func(s string) bool {
			return !arnPattern.MatchString(s)
		}).Draw(t, "invalid_arn")

		diags := validateIAMARN(val, cty.Path{})
		if len(diags) == 0 {
			t.Fatalf("expected at least one error diagnostic for %q, got none", val)
		}
	})
}

// Feature: documentdb-iam-auth, Property 4: auth_database is always "$external" for IAM users
// Validates: Requirements 2.2
func TestProperty_IAMUserDatabaseAlwaysExternal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// For any arbitrary auth_database value, resolveAuthDatabase with MONGODB-AWS
		// must always return "$external".
		currentAuthDB := rapid.String().Draw(t, "auth_database")
		resolved := resolveAuthDatabase("MONGODB-AWS", currentAuthDB)
		if resolved != "$external" {
			t.Fatalf("expected auth_database to be \"$external\" for IAM user, got %q (input: %q)", resolved, currentAuthDB)
		}
	})
}

// Feature: documentdb-iam-auth, Property 5: non-empty password always rejected for IAM users
// Validates: Requirements 3.1
func TestProperty_NonEmptyPasswordAlwaysRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		password := rapid.StringN(1, 100, -1).Draw(t, "password")
		err := validateDBUserDiff("MONGODB-AWS", password)
		if err == nil {
			t.Fatalf("expected error for MONGODB-AWS with non-empty password %q, got nil", password)
		}
	})
}
