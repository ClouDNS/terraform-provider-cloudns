package cloudns

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const EnvVarAcceptanceTestsZone = "CLOUDNS_ACCEPTANCE_TESTS_ZONE"

var providerFactories = map[string]func() (*schema.Provider, error){
	"cloudns": func() (*schema.Provider, error) {
		return New()(), nil
	},
}

func TestProviderDeclaration(t *testing.T) {
	if err := New()().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

var testAccProvider = New()()

func testAccPreCheck(t *testing.T) {
	authId := os.Getenv(EnvVarAuthId)
	subAuthId := os.Getenv(EnvVarSubAuthId)
	if authId == "" && subAuthId == "" {
		t.Fatalf("One of %s or %s must be set for acceptance tests but neither were set.", EnvVarAuthId, EnvVarSubAuthId)
	}

	if len(authId) > 0 && len(subAuthId) > 0 {
		t.Fatalf("Exactly one of %s or %s must be set for acceptance tests but both were set.", EnvVarAuthId, EnvVarSubAuthId)
	}

	if v := os.Getenv(EnvVarPassword); v == "" {
		t.Fatalf("%s must be set for acceptance tests but it wasn't set.", EnvVarPassword)
	}

	if v := os.Getenv(EnvVarAcceptanceTestsZone); v == "" {
		t.Fatalf("%s must be set for acceptance tests but it wasn't set.", EnvVarAcceptanceTestsZone)
	}

	err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}
