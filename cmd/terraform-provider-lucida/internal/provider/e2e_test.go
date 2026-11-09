package provider

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// needTerraform skips unless a terraform (or tofu) binary is available: these tests drive the real
// CLI against the fake API, covering plan, apply, refresh drift, import and destroy.
func needTerraform(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC_TERRAFORM_PATH") != "" {
		return
	}
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("no terraform binary (set TF_ACC_TERRAFORM_PATH to terraform or tofu)")
	}
}

func fakeConfig(host, body string) string {
	return fmt.Sprintf("provider \"lucida\" {\n  host = %q\n  api_key = %q\n  project_id = %q\n}\n%s", host, testKey, testProject, body)
}

func TestFlagLifecycleWithTerraform(t *testing.T) {
	needTerraform(t)
	f, api := newFake(t)
	flag := func(rollout int) string {
		return fakeConfig(api.Host, fmt.Sprintf(`resource "lucida_flag" "x" {
  key        = "new-checkout"
  definition = jsonencode({ name = "New checkout", enabled = true, filters = { rollout = %d } })
}
`, rollout))
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6,
		Steps: []resource.TestStep{
			{Config: flag(25), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet("lucida_flag.x", "id"),
				resource.TestCheckResourceAttr("lucida_flag.x", "key", "new-checkout"))},
			// Drift: changed outside Terraform, refresh reports it and apply repairs it.
			{PreConfig: func() { f.row("flags", "key", "new-checkout")["enabled"] = false }, Config: flag(25),
				Check: func(*terraform.State) error {
					if f.row("flags", "key", "new-checkout")["enabled"] != true {
						return fmt.Errorf("apply did not repair the drift")
					}
					return nil
				}},
			{Config: flag(75)},
			{ResourceName: "lucida_flag.x", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"definition"}},
		},
	})
	if f.row("flags", "key", "new-checkout") != nil {
		t.Fatal("destroy must delete the flag")
	}
}

func TestPlanTimeValidationWithTerraform(t *testing.T) {
	needTerraform(t)
	_, api := newFake(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6,
		Steps: []resource.TestStep{{
			Config:      fakeConfig(api.Host, "resource \"lucida_flag\" \"x\" {\n  key = \"k\"\n  definition = jsonencode({ colour = \"red\" })\n}\n"),
			ExpectError: regexp.MustCompile(`unknown field "colour"`),
		}},
	})
}

func TestKeySecretWithTerraform(t *testing.T) {
	needTerraform(t)
	_, api := newFake(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6,
		Steps: []resource.TestStep{{
			Config: fakeConfig(api.Host, "resource \"lucida_key\" \"ci\" {\n  name = \"ci\"\n  definition = jsonencode({ kind = \"project\", scopes = [\"read\"] })\n}\n"),
			Check:  resource.TestMatchResourceAttr("lucida_key.ci", "secret", regexp.MustCompile(`^lcsk_secret_`)),
		}},
	})
}
