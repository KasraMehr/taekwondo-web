package tournaments

import "testing"

func TestHoguDefaultsUseAgeAndWeight(t *testing.T) {
	cases := []struct {
		age            AgeCategory
		category, want string
	}{
		{AgeCategoryKhordsalan, "-33", "1"},
		{AgeCategoryKhordsalan, "+52", "3"},
		{AgeCategoryNonehalan, "-49", "2"},
		{AgeCategoryNonehalan, "+65", "3"},
		{AgeCategoryNojavanan, "-45", "2"},
		{AgeCategoryNojavanan, "+78", "4"},
		{AgeCategoryBozorgsalan, "-54", "2"},
		{AgeCategoryOmidha, "+87", "4"},
	}
	for _, tc := range cases {
		if got := defaultHoguSizeForCategory(tc.age, tc.category); got != tc.want {
			t.Errorf("%s %s: got %s, want %s", tc.age, tc.category, got, tc.want)
		}
	}
}
