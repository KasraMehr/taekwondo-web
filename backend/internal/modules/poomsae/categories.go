package poomsae

// CategoryTemplate preserves the supplied table's form lists. Birth bounds must
// be supplied for the specific event year; labels are not an age calculation.
type CategoryTemplate struct {
	Key              string `json:"key"`
	Name             string `json:"name"`
	AllowedFormCodes []int  `json:"allowedFormCodes"`
}

func CategoryTemplates() []CategoryTemplate {
	return []CategoryTemplate{
		{"under12", "زیر ۱۲ سال", []int{2, 3, 4, 5, 6, 7, 8, 9}},
		{"12to14", "۱۲ الی ۱۴ سال", []int{3, 5, 7, 8, 9, 10, 11}},
		{"15to17", "۱۵ الی ۱۷ سال", []int{5, 6, 7, 8, 9, 10, 11, 12}},
		{"18to30", "۱۸ الی ۳۰ سال", []int{7, 8, 9, 10, 11, 12, 13, 14}},
		{"31to40", "۳۱ الی ۴۰ سال", []int{7, 8, 9, 10, 11, 12, 13, 14}},
		{"41to50", "۴۱ الی ۵۰ سال", []int{8, 9, 10, 11, 12, 13, 14, 15}},
		{"51to60", "۵۱ الی ۶۰ سال", []int{9, 10, 11, 12, 13, 14, 15, 16}},
		{"61to65", "۶۱ الی ۶۵ سال", []int{9, 10, 11, 12, 13, 14, 15, 16}},
		{"over65", "بالای ۶۵ سال", []int{9, 10, 11, 12, 13, 14, 15, 16}},
	}
}
