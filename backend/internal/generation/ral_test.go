package generation

import "testing"

func TestParseStatedRAL_EuroRangeWithRALKeyword_ReturnsStated(t *testing.T) {
	jd := "We offer a RAL of €45,000 - €55,000 depending on experience."
	ral, ok := ParseStatedRAL(jd)
	if !ok {
		t.Fatal("expected a stated RAL to be found")
	}
	if ral.Source != RALSourceStated {
		t.Errorf("expected source stated, got %v", ral.Source)
	}
	if ral.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %q", ral.Currency)
	}
	if ral.Min == nil || *ral.Min != 45000 {
		t.Errorf("expected min 45000, got %v", ral.Min)
	}
	if ral.Max == nil || *ral.Max != 55000 {
		t.Errorf("expected max 55000, got %v", ral.Max)
	}
}

func TestParseStatedRAL_DollarSingleFigure_ReturnsPointEstimate(t *testing.T) {
	jd := "Compensation: $90,000 per year, plus equity."
	ral, ok := ParseStatedRAL(jd)
	if !ok {
		t.Fatal("expected a stated RAL to be found")
	}
	if ral.Currency != "USD" {
		t.Errorf("expected currency USD, got %q", ral.Currency)
	}
	if ral.Min == nil || *ral.Min != 90000 {
		t.Errorf("expected min 90000, got %v", ral.Min)
	}
	if ral.Max == nil || *ral.Max != 90000 {
		t.Errorf("expected max 90000, got %v", ral.Max)
	}
}

func TestParseStatedRAL_KSuffixRange_ParsesThousands(t *testing.T) {
	jd := "RAL 40-50k."
	ral, ok := ParseStatedRAL(jd)
	if !ok {
		t.Fatal("expected a stated RAL to be found")
	}
	if ral.Min == nil || *ral.Min != 40000 {
		t.Errorf("expected min 40000, got %v", ral.Min)
	}
	if ral.Max == nil || *ral.Max != 50000 {
		t.Errorf("expected max 50000, got %v", ral.Max)
	}
}

func TestParseStatedRAL_NoSalaryText_ReturnsFalse(t *testing.T) {
	jd := "We are looking for a Go backend engineer with 3+ years of experience in distributed systems."
	if _, ok := ParseStatedRAL(jd); ok {
		t.Fatal("expected no stated RAL to be found")
	}
}

func TestParseStatedRAL_UnrelatedDashedNumbers_ReturnsFalse(t *testing.T) {
	jd := "Requires 5-10 years of experience, available 2024-2026."
	if _, ok := ParseStatedRAL(jd); ok {
		t.Fatal("expected dashed numbers without a currency/RAL signal not to be treated as a salary")
	}
}
