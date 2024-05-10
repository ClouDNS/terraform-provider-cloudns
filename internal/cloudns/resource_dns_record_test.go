package cloudns

import (
	"fmt"
	"os"
	"testing"

	"github.com/ClouDNS/cloudns-go"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var testZone = os.Getenv(EnvVarAcceptanceTestsZone)

const recordTpl = `
resource "cloudns_dns_record" "%s" {
  name     = "%s"
  zone     = "%s"
  type     = "%s"
  value    = "%s"
  ttl      = "600"
  priority = %s
}
`

func record(recType string, resourceName string, name string, value string) string {
	return fmt.Sprintf(recordTpl, resourceName, name, testZone, recType, value, "null")
}

func mxRecord(resourceName string, name string, value string, priority string) string {
	return fmt.Sprintf(recordTpl, resourceName, name, testZone, "MX", value, priority)
}

func checkRecord(recType string, resourceName string, name string, value string) resource.TestCheckFunc {
	path := fmt.Sprintf("cloudns_dns_record.%s", resourceName)
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(path, "name", name),
		resource.TestCheckResourceAttr(path, "zone", testZone),
		resource.TestCheckResourceAttr(path, "type", recType),
		resource.TestCheckResourceAttr(path, "value", value),
		resource.TestCheckResourceAttr(path, "ttl", "600"),
	)
}

func checkMXRecord(resourceName string, name string, value string, priority string) resource.TestCheckFunc {
	path := fmt.Sprintf("cloudns_dns_record.%s", resourceName)
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(path, "name", name),
		resource.TestCheckResourceAttr(path, "zone", testZone),
		resource.TestCheckResourceAttr(path, "type", "MX"),
		resource.TestCheckResourceAttr(path, "value", value),
		resource.TestCheckResourceAttr(path, "ttl", "600"),
		resource.TestCheckResourceAttr(path, "priority", priority),
	)
}

func TestAccDnsARecord(t *testing.T) {
	testUuid := uuid.NewString()
	initialRecordValue := "1.2.3.4"
	updatedRecordValue := "5.6.7.8"

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: record("A", "some-record", testUuid, initialRecordValue),
				Check:  checkRecord("A", "some-record", testUuid, initialRecordValue),
			},
			{
				Config: record("A", "some-record", testUuid, updatedRecordValue),
				Check:  checkRecord("A", "some-record", testUuid, updatedRecordValue),
			},
		},
		CheckDestroy: CheckDestroyedRecords,
	})
}

func TestAccDnsARecordMultiMatch(t *testing.T) {
	testUuid := uuid.NewString()
	r1value := "1.2.3.4"
	r2value := "5.6.7.8"
	r1res := record("A", "some-record-1", testUuid, r1value)
	r2res := record("A", "some-record-2", testUuid, r2value)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s\n%s", r1res, r2res),
				Check: resource.ComposeTestCheckFunc(
					checkRecord("A", "some-record-1", testUuid, r1value),
					checkRecord("A", "some-record-2", testUuid, r2value),
				),
			},
		},
		CheckDestroy: CheckDestroyedRecords,
	})
}

func TestAccDnsCNAMERecord(t *testing.T) {
	testUuid := uuid.NewString()
	initialRecordValue := fmt.Sprintf("target-init.%s", testZone)
	updatedRecordValue := fmt.Sprintf("target-updated.%s", testZone)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: record("CNAME", "some-record", testUuid, initialRecordValue),
				Check:  checkRecord("CNAME", "some-record", testUuid, initialRecordValue),
			},
			{
				Config: record("CNAME", "some-record", testUuid, updatedRecordValue),
				Check:  checkRecord("CNAME", "some-record", testUuid, updatedRecordValue),
			},
		},
		CheckDestroy: CheckDestroyedRecords,
	})
}

func TestAccDnsMXRecord(t *testing.T) {
	testUuid := uuid.NewString()
	initialRecordValue := fmt.Sprintf("target-init.%s", testZone)
	updatedRecordValue := fmt.Sprintf("target-updated.%s", testZone)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: mxRecord("some-record", testUuid, initialRecordValue, "10"),
				Check:  checkMXRecord("some-record", testUuid, initialRecordValue, "10"),
			},
			{
				Config: mxRecord("some-record", testUuid, updatedRecordValue, "0"),
				Check:  checkMXRecord("some-record", testUuid, updatedRecordValue, "0"),
			},
		},
		CheckDestroy: CheckDestroyedRecords,
	})
}

func TestAccDnsImportMXRecord(t *testing.T) {
	testUuid := uuid.NewString()

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: mxRecord("record-to-import", testUuid, "mail.example.com", "10"),
				Check:  checkMXRecord("record-to-import", testUuid, "mail.example.com", "10"),
			},
			{
				ResourceName:        "cloudns_dns_record.record-to-import",
				ImportState:         true,
				ImportStateIdPrefix: fmt.Sprintf("%s/", testZone),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected single state, found: %+v", s)
					}
					rs := s[0]
					expectedState := map[string]string{
						"name":     testUuid,
						"type":     "MX",
						"value":    "mail.example.com",
						"zone":     testZone,
						"ttl":      "600",
						"priority": "10",
					}
					for k, exp := range expectedState {
						val := rs.Attributes[k]
						if exp != val {
							return fmt.Errorf("bad %#v: %#v expected: %#v", k, val, exp)
						}
					}
					return nil
				},
			},
		},
		CheckDestroy: CheckDestroyedRecords,
	})
}

func CheckDestroyedRecords(state *terraform.State) error {
	provider := testAccProvider
	apiAccess := provider.Meta().(ClientConfig).apiAccess
	records, err := cloudns.Zone{
		Ztype:  "master",
		Domain: testZone,
	}.List(&apiAccess)

	if err != nil {
		return err
	}

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "cloudns_dns_record" {
			continue
		}

		fmt.Printf("Checking that cloudns_dns_record#%s was properly deleted\n", rs.Primary.ID)

		for _, record := range records {
			existingRecordId := record.ID
			if rs.Primary.ID == existingRecordId {
				return fmt.Errorf(
					"record %s (%s.%s %d in %s %s) still exists",
					record.ID,
					record.Host,
					record.Domain,
					record.TTL,
					record.Rtype,
					record.Record,
				)
			}
		}
	}

	return nil
}
