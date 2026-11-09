package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests run against a real Lucida: TF_ACC=1 LUCIDA_API_KEY=... LUCIDA_PROJECT_ID=...
// [LUCIDA_HOST=...] go test ./cmd/terraform-provider-lucida/... -run TestAcc. They create objects
// with a per-run suffix and destroy them. Use a throwaway project.
func accPreCheck(t *testing.T) {
	t.Helper()
	for _, v := range []string{"LUCIDA_API_KEY", "LUCIDA_PROJECT_ID"} {
		if os.Getenv(v) == "" {
			t.Skipf("%s is not set", v)
		}
	}
}

func runID() string { return fmt.Sprintf("tfacc-%d", time.Now().UnixNano()%1e9) }

func TestAccFlagAndCohort(t *testing.T) {
	accPreCheck(t)
	id := runID()
	cfg := func(rollout int) string {
		return fmt.Sprintf(`
resource "lucida_flag" "f" {
  key        = "%[1]s"
  definition = jsonencode({ name = "acc %[1]s", enabled = false, filters = { rollout = %[2]d } })
}
resource "lucida_cohort" "c" {
  name       = "%[1]s"
  definition = jsonencode({ kind = "static" })
}
data "lucida_flag" "f" {
  key        = lucida_flag.f.key
  depends_on = [lucida_flag.f]
}
data "lucida_project" "p" {}
`, id, rollout)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6,
		Steps: []resource.TestStep{
			{Config: cfg(10), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrPair("data.lucida_flag.f", "id", "lucida_flag.f", "id"),
				resource.TestCheckResourceAttrSet("data.lucida_project.p", "name"))},
			{Config: cfg(20)},
			{ResourceName: "lucida_flag.f", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"definition"}},
		},
	})
}

func TestAccKeyAndTrafficExclusion(t *testing.T) {
	accPreCheck(t)
	id := runID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
resource "lucida_key" "k" {
  name       = "%[1]s"
  definition = jsonencode({ kind = "project", scopes = ["read"] })
}
resource "lucida_traffic_exclusion" "t" {
  note       = "%[1]s"
  definition = jsonencode({ kind = "ip", value = "203.0.113.7", enabled = true })
}
`, id),
			Check: resource.TestMatchResourceAttr("lucida_key.k", "secret", regexp.MustCompile(`.{16,}`)),
		}},
	})
}
