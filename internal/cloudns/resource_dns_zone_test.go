package cloudns

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDnsZone_basic(t *testing.T) {
	testUuid := uuid.NewString()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: zone("some-zone", testUuid),
				Check:  checkZone("some-zone", testUuid),
			},
		},
		CheckDestroy: CheckDestroyedZones,
	})
}

func zone(resourceName string, name string) string {
	return fmt.Sprintf(`
resource "cloudns_dns_zone" "%s" {
    domain = "%s"
    type = "master"
}
`, resourceName, name)
}

func checkZone(resourceName string, name string) resource.TestCheckFunc {
	path := fmt.Sprintf("cloudns_dns_zone.%s", resourceName)
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(path, "domain", name),
		resource.TestCheckResourceAttr(path, "type", "master"),
	)
}

func CheckDestroyedZones(state *terraform.State) error {
	provider := testAccProvider
	apiAccess := provider.Meta().(ClientConfig).apiAccess
	zones, err := apiAccess.Listzones()

	if err != nil {
		return err
	}

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "cloudns_dns_zone" {
			continue
		}

		fmt.Printf("Checking that cloudns_dns_zone#%s was properly deleted\n", rs.Primary.ID)

		for _, zone := range zones {
			existingZoneId := zone.Domain
			if rs.Primary.ID == existingZoneId {
				return fmt.Errorf(
					"zone %s still exists",
					zone.Domain,
				)
			}
		}
	}

	return nil
}
