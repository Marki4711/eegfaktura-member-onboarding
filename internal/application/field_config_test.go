package application

import (
	"testing"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// mkCfg builds a FieldConfigEntry map from plain state strings for brevity in tests.
func mkCfg(pairs ...string) map[string]FieldConfigEntry {
	m := make(map[string]FieldConfigEntry)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = FieldConfigEntry{State: pairs[i+1]}
	}
	return m
}

// --- effectiveState ---

func TestEffectiveState_ExplicitOverride(t *testing.T) {
	cfg := mkCfg("phone", "required")
	if got := effectiveState(cfg, "phone"); got != "required" {
		t.Errorf("expected required, got %s", got)
	}
}

func TestEffectiveState_FallbackToRegisteredDefault(t *testing.T) {
	if got := effectiveState(map[string]FieldConfigEntry{}, "phone"); got != "optional" {
		t.Errorf("expected optional, got %s", got)
	}
}

func TestEffectiveState_FallbackToHiddenForNewField(t *testing.T) {
	if got := effectiveState(map[string]FieldConfigEntry{}, "heat_pump"); got != "hidden" {
		t.Errorf("expected hidden, got %s", got)
	}
}

func TestEffectiveState_UnknownFieldReturnsHidden(t *testing.T) {
	if got := effectiveState(map[string]FieldConfigEntry{}, "totally_unknown_field"); got != "hidden" {
		t.Errorf("expected hidden for unknown field, got %s", got)
	}
}

func TestEffectiveState_AdminOnlyCountsAsState(t *testing.T) {
	cfg := mkCfg("membership_start_date", "admin_only")
	if got := effectiveState(cfg, "membership_start_date"); got != "admin_only" {
		t.Errorf("expected admin_only, got %s", got)
	}
}

// --- validateConfigurableRequiredFields ---

func baseAppWithAllOptional() *shared.Application {
	return &shared.Application{
		MemberType:           shared.MemberTypePrivate,
		Email:                "test@example.at",
		ResidentStreet:       "Teststr.",
		ResidentStreetNumber: "1",
		ResidentZip:          "4020",
		ResidentCity:         "Linz",
	}
}

func TestValidateConfigurableRequiredFields_AllOptional_NoError(t *testing.T) {
	app := baseAppWithAllOptional()
	if err := validateConfigurableRequiredFields(app, map[string]FieldConfigEntry{}); err != nil {
		t.Fatalf("expected no error with all-optional config, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_PhoneRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.Phone = nil
	err := validateConfigurableRequiredFields(app, mkCfg("phone", "required"))
	if err == nil {
		t.Fatal("expected error for required phone that is nil")
	}
	ve, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, hasField := ve.Fields["phone"]; !hasField {
		t.Errorf("expected 'phone' field error, got: %v", ve.Fields)
	}
}

func TestValidateConfigurableRequiredFields_PhoneRequired_Present(t *testing.T) {
	app := baseAppWithAllOptional()
	p := "+43 664 1234567"
	app.Phone = &p
	if err := validateConfigurableRequiredFields(app, mkCfg("phone", "required")); err != nil {
		t.Fatalf("expected no error when required phone is present, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_BirthDateRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.BirthDate = nil
	if err := validateConfigurableRequiredFields(app, mkCfg("birth_date", "required")); err == nil {
		t.Fatal("expected error for required birth_date that is nil")
	}
}

func TestValidateConfigurableRequiredFields_MembershipStartDateRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.MembershipStartDate = nil
	if err := validateConfigurableRequiredFields(app, mkCfg("membership_start_date", "required")); err == nil {
		t.Fatal("expected error for required membership_start_date that is nil")
	}
}

func TestValidateConfigurableRequiredFields_PvPowerRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.PvPowerKwp = nil
	if err := validateConfigurableRequiredFields(app, mkCfg("pv_power_kwp", "required")); err == nil {
		t.Fatal("expected error for required pv_power_kwp that is nil")
	}
}

func TestValidateConfigurableRequiredFields_HeatPumpRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.HeatPump = nil
	if err := validateConfigurableRequiredFields(app, mkCfg("heat_pump", "required")); err == nil {
		t.Fatal("expected error for required heat_pump that is nil")
	}
}

func TestValidateConfigurableRequiredFields_HeatPumpRequired_Present(t *testing.T) {
	app := baseAppWithAllOptional()
	v := true
	app.HeatPump = &v
	if err := validateConfigurableRequiredFields(app, mkCfg("heat_pump", "required")); err != nil {
		t.Fatalf("expected no error when required heat_pump is set, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_MultipleFieldsMissing(t *testing.T) {
	app := baseAppWithAllOptional()
	now := time.Now()
	app.MembershipStartDate = &now
	cfg := mkCfg("phone", "required", "heat_pump", "required")
	err := validateConfigurableRequiredFields(app, cfg)
	if err == nil {
		t.Fatal("expected validation error for multiple missing required fields")
	}
	ve, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, hasPhone := ve.Fields["phone"]; !hasPhone {
		t.Errorf("expected 'phone' field error")
	}
	if _, hasHP := ve.Fields["heatPump"]; !hasHP {
		t.Errorf("expected 'heatPump' field error")
	}
}

// --- admin_only: applyAdminValues ---

func TestApplyAdminValues_SetsIntField(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "3"
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.PersonsInHousehold == nil || *app.PersonsInHousehold != 3 {
		t.Errorf("expected persons_in_household=3, got %v", app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_DoesNotOverwriteExisting(t *testing.T) {
	app := baseAppWithAllOptional()
	existing := 5
	app.PersonsInHousehold = &existing
	val := "99"
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if *app.PersonsInHousehold != 5 {
		t.Errorf("expected existing value preserved, got %d", *app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_SkipsNonAdminOnly(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "7"
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "optional", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.PersonsInHousehold != nil {
		t.Errorf("expected nil for non-admin_only field, got %v", app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_SetsBoolField(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "true"
	cfg := map[string]FieldConfigEntry{
		"heat_pump": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.HeatPump == nil || *app.HeatPump != true {
		t.Errorf("expected heat_pump=true, got %v", app.HeatPump)
	}
}

func TestApplyAdminValues_SetsBoolFieldFalse(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "false"
	cfg := map[string]FieldConfigEntry{
		"electric_vehicle": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.ElectricVehicle == nil || *app.ElectricVehicle != false {
		t.Errorf("expected electric_vehicle=false, got %v", app.ElectricVehicle)
	}
}

func TestApplyAdminValues_SetsFloatField(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "10.5"
	cfg := map[string]FieldConfigEntry{
		"pv_power_kwp": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.PvPowerKwp == nil || *app.PvPowerKwp != 10.5 {
		t.Errorf("expected pv_power_kwp=10.5, got %v", app.PvPowerKwp)
	}
}

func TestApplyAdminValues_InvalidIntValue_LeavesNil(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "abc"
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.PersonsInHousehold != nil {
		t.Errorf("expected nil for invalid int value, got %v", app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_EmptyAdminValue_LeavesNil(t *testing.T) {
	app := baseAppWithAllOptional()
	empty := ""
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "admin_only", AdminValue: &empty},
	}
	applyAdminValues(app, cfg)
	if app.PersonsInHousehold != nil {
		t.Errorf("expected nil for empty admin_value, got %v", app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_NilAdminValue_LeavesNil(t *testing.T) {
	app := baseAppWithAllOptional()
	cfg := map[string]FieldConfigEntry{
		"persons_in_household": {State: "admin_only", AdminValue: nil},
	}
	applyAdminValues(app, cfg)
	if app.PersonsInHousehold != nil {
		t.Errorf("expected nil when AdminValue is nil, got %v", app.PersonsInHousehold)
	}
}

func TestApplyAdminValues_SetsMembershipStartDate(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "2026-05-01"
	cfg := map[string]FieldConfigEntry{
		"membership_start_date": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.MembershipStartDate == nil {
		t.Error("expected membership_start_date to be set")
	}
}

func TestApplyAdminValues_InvalidDateValue_LeavesNil(t *testing.T) {
	app := baseAppWithAllOptional()
	val := "not-a-date"
	cfg := map[string]FieldConfigEntry{
		"membership_start_date": {State: "admin_only", AdminValue: &val},
	}
	applyAdminValues(app, cfg)
	if app.MembershipStartDate != nil {
		t.Error("expected nil for invalid date value")
	}
}

// --- validateConfigurableMeteringPointFields ---

func TestValidateMPFields_TransformerRequired_Missing(t *testing.T) {
	points := []shared.MeteringPoint{{MeteringPoint: "AT0001", Transformer: nil}}
	if err := validateConfigurableMeteringPointFields(points, mkCfg("transformer", "required")); err == nil {
		t.Fatal("expected error for required transformer that is nil")
	}
}

func TestValidateMPFields_TransformerOptional_Missing_NoError(t *testing.T) {
	points := []shared.MeteringPoint{{MeteringPoint: "AT0001", Transformer: nil}}
	if err := validateConfigurableMeteringPointFields(points, map[string]FieldConfigEntry{}); err != nil {
		t.Fatalf("expected no error for optional/hidden transformer, got: %v", err)
	}
}

func TestValidateMPFields_TransformerRequired_Present(t *testing.T) {
	tr := "T1"
	points := []shared.MeteringPoint{{MeteringPoint: "AT0001", Transformer: &tr}}
	if err := validateConfigurableMeteringPointFields(points, mkCfg("transformer", "required")); err != nil {
		t.Fatalf("expected no error when required transformer is present, got: %v", err)
	}
}

func TestValidateMPFields_InstallationNumberRequired_Missing(t *testing.T) {
	points := []shared.MeteringPoint{{MeteringPoint: "AT0001", InstallationNumber: nil}}
	if err := validateConfigurableMeteringPointFields(points, mkCfg("installation_number", "required")); err == nil {
		t.Fatal("expected error for required installation_number that is nil")
	}
}

func TestValidateMPFields_MultiplePoints_SecondMissing(t *testing.T) {
	tr1 := "T1"
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", Transformer: &tr1},
		{MeteringPoint: "AT0002", Transformer: nil},
	}
	if err := validateConfigurableMeteringPointFields(points, mkCfg("transformer", "required")); err == nil {
		t.Fatal("expected error when second metering point is missing required transformer")
	}
}
