package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/keycloak/terraform-provider-keycloak/keycloak"
)

func TestAccKeycloakRealmUserProfile_featureDisabled(t *testing.T) {
	skipIfVersionIsGreaterThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_24)

	realmName := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config:      testKeycloakRealmUserProfile_userProfileDisabled(realmName),
				ExpectError: regexp.MustCompile("User Profile is disabled"),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_featureNotSet(t *testing.T) {
	skipIfVersionIsGreaterThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_24)

	realmName := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config:      testKeycloakRealmUserProfile_userProfileEnabledNotSet(realmName),
				ExpectError: regexp.MustCompile("User Profile is disabled"),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_enabledByDefault(t *testing.T) {
	skipIfVersionIsLessThan(testCtx, t, keycloakClient, keycloak.Version_24)

	realmName := acctest.RandomWithPrefix("tf-acc")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_userProfileEnabledNotSet(realmName),
				Check:  testAccCheckKeycloakRealmUserProfileExists("keycloak_realm_user_profile.realm_user_profile"),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_basicEmpty(t *testing.T) {
	skipIfVersionIsLessThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_14)

	realmName := acctest.RandomWithPrefix("tf-acc")

	realmUserProfile := &keycloak.RealmUserProfile{}
	if ok, _ := keycloakClient.VersionIsGreaterThanOrEqualTo(testCtx, keycloak.Version_23); ok {
		// Username and email can't be removed in this version
		realmUserProfile.Attributes = []*keycloak.RealmUserProfileAttribute{{Name: "username"}, {Name: "email"}}
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, realmUserProfile),
				Check:  testAccCheckKeycloakRealmUserProfileExists("keycloak_realm_user_profile.realm_user_profile"),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_basicFull(t *testing.T) {
	skipIfVersionIsLessThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_14)

	realmName := acctest.RandomWithPrefix("tf-acc")

	mvSupported, err := keycloakClient.VersionIsGreaterThanOrEqualTo(testCtx, keycloak.Version_24)
	if err != nil {
		t.Errorf("error checking keycloak version: %v", err)
	}

	realmUserProfile := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{Name: "attribute1"},
			{
				Name:        "attribute2",
				DisplayName: "attribute 2",
				MultiValued: mvSupported,
				Group:       "group",
				Selector:    &keycloak.RealmUserProfileSelector{Scopes: []string{"roles"}},
				Required: &keycloak.RealmUserProfileRequired{
					Roles:  []string{"user"},
					Scopes: []string{"offline_access"},
				},
				Permissions: &keycloak.RealmUserProfilePermissions{
					Edit: []string{"admin", "user"},
					View: []string{"admin", "user"},
				},
				Validations: map[string]keycloak.RealmUserProfileValidationConfig{
					"person-name-prohibited-characters": map[string]interface{}{},
					"pattern":                           map[string]interface{}{"pattern": "\"^[a-z]+$\"", "error_message": "\"Error!\""},
				},
				Annotations: map[string]interface{}{
					"foo":               "\"bar\"",
					"inputOptionLabels": "{\"a\":\"b\"}",
				},
			},
		},
		Groups: []*keycloak.RealmUserProfileGroup{
			{
				Name:               "group",
				DisplayDescription: "Description",
				DisplayHeader:      "Header",
				Annotations: map[string]interface{}{
					"foo":  "\"bar\"",
					"test": "{\"a2\":\"b2\"}",
				},
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, realmUserProfile),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", realmUserProfile,
				),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_group(t *testing.T) {
	skipIfVersionIsLessThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_14)

	realmName := acctest.RandomWithPrefix("tf-acc")

	withoutGroup := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{Name: "attribute"},
		},
	}

	withGroup := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{Name: "attribute"},
		},
		Groups: []*keycloak.RealmUserProfileGroup{
			{Name: "group"},
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutGroup),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutGroup,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withGroup),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withGroup,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutGroup),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutGroup,
				),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_attributeValidator(t *testing.T) {
	skipIfVersionIsLessThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_14)

	realmName := acctest.RandomWithPrefix("tf-acc")

	withoutValidator := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{Name: "attribute"},
		},
	}

	withInitialConfig := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Validations: map[string]keycloak.RealmUserProfileValidationConfig{
					"length":  map[string]interface{}{"min": "5", "max": "10"},
					"options": map[string]interface{}{"options": "[\"cgu\"]"},
				},
			},
		},
	}

	withNewConfig := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Validations: map[string]keycloak.RealmUserProfileValidationConfig{
					"length": map[string]interface{}{"min": "6", "max": "10"},
				},
			},
		},
	}

	withNewValidator := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Validations: map[string]keycloak.RealmUserProfileValidationConfig{
					"person-name-prohibited-characters": map[string]interface{}{},
					"length":                            map[string]interface{}{"min": "6", "max": "10"},
				},
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutValidator),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutValidator,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withInitialConfig),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withInitialConfig,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withNewConfig),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withNewConfig,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withNewValidator),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withNewValidator,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withNewConfig),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withNewConfig,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutValidator),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutValidator,
				),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_attributePermissions(t *testing.T) {
	skipIfVersionIsLessThanOrEqualTo(testCtx, t, keycloakClient, keycloak.Version_14)

	realmName := acctest.RandomWithPrefix("tf-acc")

	withoutPermissions := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
			},
		},
	}

	viewAttributeMissing := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Permissions: &keycloak.RealmUserProfilePermissions{
					Edit: []string{"admin", "user"},
				},
			},
		},
	}

	editAttributeMissing := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Permissions: &keycloak.RealmUserProfilePermissions{
					View: []string{"admin", "user"},
				},
			},
		},
	}

	bothAttributesMissing := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name:        "attribute",
				Permissions: &keycloak.RealmUserProfilePermissions{},
			},
		},
	}

	withRightPermissions := &keycloak.RealmUserProfile{
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
			{
				Name: "attribute",
				Permissions: &keycloak.RealmUserProfilePermissions{
					Edit: []string{"admin", "user"},
					View: []string{"admin", "user"},
				},
			},
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutPermissions),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutPermissions,
				),
			},
			{
				Config:      testKeycloakRealmUserProfile_template(realmName, viewAttributeMissing),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			{
				Config:      testKeycloakRealmUserProfile_template(realmName, editAttributeMissing),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			{
				Config:      testKeycloakRealmUserProfile_template(realmName, bothAttributesMissing),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withRightPermissions),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withRightPermissions,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, withoutPermissions),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", withoutPermissions,
				),
			},
		},
	})
}

func TestAccKeycloakRealmUserProfile_unmanagedPolicyEnabled(t *testing.T) {
	skipIfVersionIsLessThan(testCtx, t, keycloakClient, keycloak.Version_24)

	realmName := acctest.RandomWithPrefix("tf-acc")

	unmanagedPolicyEnabled := &keycloak.RealmUserProfile{
		Groups: []*keycloak.RealmUserProfileGroup{},
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
		},
		UnmanagedAttributePolicy: stringPointer(ENABLED),
	}

	unmanagedPolicyDisabled := &keycloak.RealmUserProfile{
		Groups: []*keycloak.RealmUserProfileGroup{},
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
		},
		UnmanagedAttributePolicy: stringPointer(DISABLED),
	}

	unmanagedPolicyAdminEdit := &keycloak.RealmUserProfile{
		Groups: []*keycloak.RealmUserProfileGroup{},
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
		},
		UnmanagedAttributePolicy: stringPointer(ADMIN_EDIT),
	}

	unmanagedPolicyAdminView := &keycloak.RealmUserProfile{
		Groups: []*keycloak.RealmUserProfileGroup{},
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
		},
		UnmanagedAttributePolicy: stringPointer(ADMIN_VIEW),
	}

	unmanagedPolicyNotSet := &keycloak.RealmUserProfile{
		Groups: []*keycloak.RealmUserProfileGroup{},
		Attributes: []*keycloak.RealmUserProfileAttribute{
			{Name: "username"}, {Name: "email"}, // Version >=23 needs these
		},
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testAccCheckKeycloakRealmUserProfileDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyEnabled),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyEnabled,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyDisabled),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyNotSet,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyNotSet),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyNotSet,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyAdminEdit),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyAdminEdit,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyAdminView),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyAdminView,
				),
			},
			{
				Config: testKeycloakRealmUserProfile_template(realmName, unmanagedPolicyNotSet),
				Check: testAccCheckKeycloakRealmUserProfileStateEqual(
					"keycloak_realm_user_profile.realm_user_profile", unmanagedPolicyNotSet,
				),
			},
		},
	})
}

func testKeycloakRealmUserProfile_userProfileDisabled(realm string) string {
	return fmt.Sprintf(`
resource "keycloak_realm" "realm" {
	realm = "%s"

	attributes = {
		userProfileEnabled  = false
	}
}
resource "keycloak_realm_user_profile" "realm_user_profile" {
	realm_id = keycloak_realm.realm.id
}
`, realm)
}

func testKeycloakRealmUserProfile_userProfileEnabledNotSet(realm string) string {
	return fmt.Sprintf(`
resource "keycloak_realm" "realm" {
	realm = "%s"
}
resource "keycloak_realm_user_profile" "realm_user_profile" {
	realm_id = keycloak_realm.realm.id

	attribute {
		name = "username"
    }
	attribute {
		name = "email"
    }
}
`, realm)
}

func testKeycloakRealmUserProfile_template(realm string, realmUserProfile *keycloak.RealmUserProfile) string {
	tmpl, err := template.New("").Funcs(template.FuncMap{"StringsJoin": strings.Join}).Parse(`
resource "keycloak_realm" "realm" {
	realm 	   = "{{ .realm }}"

	attributes = {
		userProfileEnabled  = true
	}
}

resource "keycloak_realm_user_profile" "realm_user_profile" {
	realm_id = keycloak_realm.realm.id

	{{- if .userProfile.UnmanagedAttributePolicy }}
	unmanaged_attribute_policy = "{{ .userProfile.UnmanagedAttributePolicy}}"
	{{- end }}

	{{- range $_, $attribute := .userProfile.Attributes }}
	attribute {
        name = "{{ $attribute.Name }}"
		{{- if $attribute.DisplayName }}
        display_name = "{{ $attribute.DisplayName }}"
		{{- end }}

		{{- if $attribute.MultiValued }}
        multi_valued = "{{ $attribute.MultiValued }}"
		{{- end }}

		{{- if $attribute.Group }}
        group = "{{ $attribute.Group }}"
		{{- end }}

		{{- if $attribute.Selector }}
		{{- if $attribute.Selector.Scopes }}
        enabled_when_scope = ["{{ StringsJoin $attribute.Selector.Scopes "\", \"" }}"]
		{{- end }}
		{{- end }}

		{{- if $attribute.Required }}
		{{- if $attribute.Required.Roles }}
        required_for_roles = ["{{ StringsJoin $attribute.Required.Roles "\", \"" }}"]
		{{- end }}
		{{- end }}

		{{- if $attribute.Required }}
		{{- if $attribute.Required.Scopes }}
        required_for_scopes = ["{{ StringsJoin $attribute.Required.Scopes "\", \"" }}"]
		{{- end }}
		{{- end }}

		{{- if $attribute.Permissions }}
        permissions {
			{{- if $attribute.Permissions.View }}
            view = ["{{ StringsJoin $attribute.Permissions.View "\", \"" }}"]
			{{- end }}
			{{- if $attribute.Permissions.Edit }}
            edit = ["{{ StringsJoin $attribute.Permissions.Edit "\", \"" }}"]
			{{- end }}
        }
		{{- end }}

		{{- if $attribute.Validations }}
		{{ range $name, $config := $attribute.Validations }}
        validator {
            name = "{{ $name }}"
            {{- if $config }}
            config = {
                {{- range $key, $value := $config }}
                {{ $key }} = jsonencode ( {{ $value }} )
                {{- end }}
            }
            {{- end }}
        }
		{{- end }}
		{{- end }}

		{{- if $attribute.Annotations }}
        annotations = {
            {{- range $key, $value := $attribute.Annotations }}
            {{ $key }} = jsonencode ( {{ $value }} )
            {{- end }}
        }
		{{- end }}
    }
	{{- end }}

	{{- range $_, $group := .userProfile.Groups }}
    group {
        name = "{{ $group.Name }}"

		{{- if $group.DisplayHeader }}
        display_header = "{{ $group.DisplayHeader }}"
		{{- end }}

		{{- if $group.DisplayDescription }}
        display_description = "{{ $group.DisplayDescription }}"
		{{- end }}

		{{- if $group.Annotations }}
        annotations = {
            {{- range $key, $value := $group.Annotations }}
            {{ $key }} = jsonencode ( {{ $value }} )
            {{- end }}
        }
		{{- end }}
    }
	{{- end }}
}
	`)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	var tmplBuf bytes.Buffer
	err = tmpl.Execute(&tmplBuf, map[string]interface{}{"realm": realm, "userProfile": realmUserProfile})
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return tmplBuf.String()
}

func testAccCheckKeycloakRealmUserProfileExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := getRealmUserProfileFromState(s, resourceName)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckKeycloakRealmUserProfileStateEqual(resourceName string, realmUserProfile *keycloak.RealmUserProfile) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		realmUserProfileFromState, err := getRealmUserProfileFromState(s, resourceName)
		if err != nil {
			return err
		}

		// JSON is not as stable to compare as a struct, ex. empty arrays are not always present in JSON
		// TODO this should be replaced with an actual comparison the json == json is a quick fix.
		if !reflect.DeepEqual(realmUserProfile, realmUserProfileFromState) {
			j1, _ := json.Marshal(realmUserProfile)
			j2, _ := json.Marshal(realmUserProfileFromState)
			sj1 := string(j1)
			sj2 := string(j2)

			if sj1 == sj2 { // might be a dialect difference, ex. empty arrays represented as null
				return nil
			}

			return fmt.Errorf("%v\nshould be equal to\n%v", sj1, sj2)
		}

		return nil
	}
}

func testAccCheckKeycloakRealmUserProfileDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "keycloak_realm_user_profile" {
				continue
			}

			realm := rs.Primary.Attributes["realm_id"]

			realmUserProfile, _ := keycloakClient.GetRealmUserProfile(testCtx, realm)
			if realmUserProfile != nil {
				return fmt.Errorf("user profile for realm %s", realm)
			}
		}

		return nil
	}
}

func getRealmUserProfileFromState(s *terraform.State, resourceName string) (*keycloak.RealmUserProfile, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", resourceName)
	}

	realm := rs.Primary.Attributes["realm_id"]

	realmUserProfile, err := keycloakClient.GetRealmUserProfile(testCtx, realm)
	if err != nil {
		return nil, fmt.Errorf("error getting realm user profile: %s", err)
	}

	return realmUserProfile, nil
}
