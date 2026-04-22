package application

import (
	"testing"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// --- effectiveState ---

func TestEffectiveState_ExplicitOverride(t *testing.T) {
	cfg := map[string]string{"phone": "required"}
	if got := effectiveState(cfg, "phone"); got != "required" {
		t.Errorf("expected required, got %s", got)
	}
}

func TestEffectiveState_FallbackToRegisteredDefault(t *testing.T) {
	// phone default is "optional"; no override → must return "optional"
	if got := effectiveState(map[string]string{}, "phone"); got != "optional" {
		t.Errorf("expected optional, got %s", got)
	}
}

func TestEffectiveState_FallbackToHiddenForNewField(t *testing.T) {
	// heat_pump default is "hidden"
	if got := effectiveState(map[string]string{}, "heat_pump"); got != "hidden" {
		t.Errorf("expected hidden, got %s", got)
	}
}

func TestEffectiveState_UnknownFieldReturnsHidden(t *testing.T) {
	if got := effectiveState(map[string]string{}, "totally_unknown_field"); got != "hidden" {
		t.Errorf("expected hidden for unknown field, got %s", got)
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
	// empty config → all fields use defaults → none required
	if err := validateConfigurableRequiredFields(app, map[string]string{}); err != nil {
		t.Fatalf("expected no error with all-optional config, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_PhoneRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.Phone = nil
	cfg := map[string]string{"phone": "required"}
	err := validateConfigurableRequiredFields(app, cfg)
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
	cfg := map[string]string{"phone": "required"}
	if err := validateConfigurableRequiredFields(app, cfg); err != nil {
		t.Fatalf("expected no error when required phone is present, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_BirthDateRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.BirthDate = nil
	cfg := map[string]string{"birth_date": "required"}
	err := validateConfigurableRequiredFields(app, cfg)
	if err == nil {
		t.Fatal("expected error for required birth_date that is nil")
	}
}

func TestValidateConfigurableRequiredFields_MembershipStartDateRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.MembershipStartDate = nil
	cfg := map[string]string{"membership_start_date": "required"}
	err := validateConfigurableRequiredFields(app, cfg)
	if err == nil {
		t.Fatal("expected error for required membership_start_date that is nil")
	}
}

func TestValidateConfigurableRequiredFields_PvPowerRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.PvPowerKwp = nil
	cfg := map[string]string{"pv_power_kwp": "required"}
	err := validateConfigurableRequiredFields(app, cfg)
	if err == nil {
		t.Fatal("expected error for required pv_power_kwp that is nil")
	}
}

func TestValidateConfigurableRequiredFields_HeatPumpRequired_Missing(t *testing.T) {
	app := baseAppWithAllOptional()
	app.HeatPump = nil
	cfg := map[string]string{"heat_pump": "required"}
	err := validateConfigurableRequiredFields(app, cfg)
	if err == nil {
		t.Fatal("expected error for required heat_pump that is nil")
	}
}

func TestValidateConfigurableRequiredFields_HeatPumpRequired_Present(t *testing.T) {
	app := baseAppWithAllOptional()
	v := true
	app.HeatPump = &v
	cfg := map[string]string{"heat_pump": "required"}
	if err := validateConfigurableRequiredFields(app, cfg); err != nil {
		t.Fatalf("expected no error when required heat_pump is set, got: %v", err)
	}
}

func TestValidateConfigurableRequiredFields_MultipleFieldsMissing(t *testing.T) {
	app := baseAppWithAllOptional()
	now := time.Now()
	app.MembershipStartDate = &now // present
	// phone and heat_pump missing
	cfg := map[string]string{
		"phone":     "required",
		"heat_pump": "required",
	}
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

// --- validateConfigurableMeteringPointFields ---

func TestValidateMPFields_TransformerRequired_Missing(t *testing.T) {
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", Transformer: nil},
	}
	cfg := map[string]string{"transformer": "required"}
	err := validateConfigurableMeteringPointFields(points, cfg)
	if err == nil {
		t.Fatal("expected error for required transformer that is nil")
	}
}

func TestValidateMPFields_TransformerOptional_Missing_NoError(t *testing.T) {
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", Transformer: nil},
	}
	// default for transformer is "hidden" (not "required") → no error
	if err := validateConfigurableMeteringPointFields(points, map[string]string{}); err != nil {
		t.Fatalf("expected no error for optional/hidden transformer, got: %v", err)
	}
}

func TestValidateMPFields_TransformerRequired_Present(t *testing.T) {
	tr := "T1"
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", Transformer: &tr},
	}
	cfg := map[string]string{"transformer": "required"}
	if err := validateConfigurableMeteringPointFields(points, cfg); err != nil {
		t.Fatalf("expected no error when required transformer is present, got: %v", err)
	}
}

func TestValidateMPFields_InstallationNumberRequired_Missing(t *testing.T) {
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", InstallationNumber: nil},
	}
	cfg := map[string]string{"installation_number": "required"}
	if err := validateConfigurableMeteringPointFields(points, cfg); err == nil {
		t.Fatal("expected error for required installation_number that is nil")
	}
}

func TestValidateMPFields_MultiplePoints_SecondMissing(t *testing.T) {
	tr1 := "T1"
	points := []shared.MeteringPoint{
		{MeteringPoint: "AT0001", Transformer: &tr1},
		{MeteringPoint: "AT0002", Transformer: nil},
	}
	cfg := map[string]string{"transformer": "required"}
	if err := validateConfigurableMeteringPointFields(points, cfg); err == nil {
		t.Fatal("expected error when second metering point is missing required transformer")
	}
}
