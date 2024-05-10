package cloudns

import (
	"fmt"
	"testing"

	"github.com/ClouDNS/cloudns-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDnsZone_basic(t *testing.T) {
	testUuid := uuid.NewString()

	resource.UnitTest(t, resource.TestCase{
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
    name = "%s"
    type = "master"
}
`, resourceName, name)
}

func checkZone(resourceName string, name string) resource.TestCheckFunc {
	path := fmt.Sprintf("cloudns_dns_zone.%s", resourceName)
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(path, "name", name),
		resource.TestCheckResourceAttr(path, "type", "master"),
	)
}

func CheckDestroyedZones(state *terraform.State) error {
	provider := testAccProvider
	apiAccess := provider.Meta().(ClientConfig).apiAccess
	zones, err := cloudns.Zone{}.List(&apiAccess)

	if err != nil {
		return err
	}

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "cloudns_dns_zone" {
			continue
		}

		fmt.Printf("Checking that cloudns_dns_zone#%s was properly deleted\n", rs.Primary.ID)

		for _, zone := range zones {
			existingZoneId := zone.ID
			if rs.Primary.ID == existingZoneId {
				return fmt.Errorf(
					"zone %s still exists",
					zone.ID,
				)
			}
		}
	}

	return nil
}
