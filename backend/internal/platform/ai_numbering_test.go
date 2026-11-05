package platform

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNormalizeAIIDPlanMapsModelIDsToRealCategories(t *testing.T) {
	categories := []string{"+73", "-49", "-57"}
	plan := normalizeAIIDPlan(json.RawMessage(`[3,2,2,99]`), "model reason", categories)
	want := []string{"-57", "-49", "+73"}
	if !reflect.DeepEqual(plan.Priority, want) {
		t.Fatalf("priority = %v, want %v", plan.Priority, want)
	}
	if plan.Reason == "model reason" {
		t.Fatal("reason should reflect a repaired order")
	}
}

func TestNormalizeAIPlanRepairsProviderCategories(t *testing.T) {
	categories := []string{"+73", "-49", "-57"}
	plan := normalizeAIPlan(aiNumberingPlan{
		Priority: []string{"وزن −۵۷", "-49", "-49", "imaginary"},
		Reason:   "A model explanation for its original order",
	}, categories)
	want := []string{"-57", "-49", "+73"}
	if !reflect.DeepEqual(plan.Priority, want) {
		t.Fatalf("priority = %v, want %v", plan.Priority, want)
	}
	if plan.Reason == "A model explanation for its original order" {
		t.Fatal("reason should not describe a repaired model order")
	}
}

func TestNormalizeAIPlanFallsBackWhenNoCategoryMatches(t *testing.T) {
	categories := []string{"+73", "-49"}
	plan := normalizeAIPlan(aiNumberingPlan{Priority: []string{"unknown"}}, categories)
	if !reflect.DeepEqual(plan.Priority, categories) {
		t.Fatalf("priority = %v, want %v", plan.Priority, categories)
	}
}
