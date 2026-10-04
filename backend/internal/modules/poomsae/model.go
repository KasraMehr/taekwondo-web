// Package poomsae implements individual, two-form poomsae competitions.
package poomsae

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid poomsae input")
var ErrConflict = errors.New("poomsae state conflict")
var ErrNotFound = errors.New("poomsae resource not found")

func invalid(s string) error  { return fmt.Errorf("%w: %s", ErrInvalid, s) }
func conflict(s string) error { return fmt.Errorf("%w: %s", ErrConflict, s) }

// Decimal is an exact hundredth, serialized as an ASCII decimal string.
// JSON numbers are deliberately not accepted (the API has no floating point scores).
type Decimal int64

var decimalPattern = regexp.MustCompile(`^[0-9]{1,4}(\.[0-9]{1,2})?$`)

func (d *Decimal) UnmarshalJSON(b []byte) error {
	var s string
	if string(b) == "null" {
		return invalid("decimal must be a string")
	}
	if err := json.Unmarshal(b, &s); err != nil || !decimalPattern.MatchString(s) {
		return invalid("decimal must be a nonnegative string with at most two decimal places")
	}
	parts := strings.Split(s, ".")
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	frac := int64(0)
	if len(parts) == 2 {
		f := parts[1]
		if len(f) == 1 {
			f += "0"
		}
		frac, _ = strconv.ParseInt(f, 10, 64)
	}
	*d = Decimal(whole*100 + frac)
	return nil
}
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%d.%02d", d/100, d%100))
}

type Rules struct {
	AccuracyMax       Decimal `json:"accuracyMax"`
	PresentationMax   Decimal `json:"presentationMax"`
	Precision         int     `json:"precision"`
	TiePolicy         string  `json:"tiePolicy"`
	FormSelection     string  `json:"formSelection"` // division or entry
	AllowRepeatedForm *bool   `json:"allowRepeatedForm"`
	Version           string  `json:"version"`
}
type EventInput struct {
	Name  string `json:"name"`
	Date  string `json:"date"`
	Rules Rules  `json:"rules"`
}
type Event struct {
	ID string `json:"id"`
	EventInput
	Revision  int64     `json:"revision"`
	CreatedAt time.Time `json:"createdAt"`
}

// Birth bounds are explicit Gregorian ISO dates, inclusive; no guessed age formula.
type DivisionInput struct {
	Name             string  `json:"name"`
	CompetitionType  string  `json:"competitionType"`
	Gender           string  `json:"gender"`
	Belt             string  `json:"belt"`
	BirthDateFrom    *string `json:"birthDateFrom"`
	BirthDateTo      *string `json:"birthDateTo"`
	AllowedFormCodes []int   `json:"allowedFormCodes"`
	Form1Code        *int    `json:"form1Code"`
	Form2Code        *int    `json:"form2Code"`
}
type Division struct {
	ID      string `json:"id"`
	EventID string `json:"eventId"`
	DivisionInput
	Status     string  `json:"status"`
	EverScored bool    `json:"everScored"`
	Revision   int64   `json:"revision"`
	Entries    []Entry `json:"entries"`
	Draws      []Draw  `json:"draws"`
}
type EntryInput struct {
	AthleteID string `json:"athleteId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	TeamName  string `json:"teamName"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
	Belt      string `json:"belt"`
	Status    string `json:"status"` // active, absent, withdrawn
	Reason    string `json:"reason,omitempty"`
}
type FormScore struct {
	Code         *int     `json:"code"`
	Accuracy     *Decimal `json:"accuracy"`
	Presentation *Decimal `json:"presentation"`
}
type ScoreInput struct {
	Form1  FormScore `json:"form1"`
	Form2  FormScore `json:"form2"`
	Reason string    `json:"reason,omitempty"`
}
type Entry struct {
	ID string `json:"id"`
	EntryInput
	Scores    ScoreInput `json:"scores"`
	UpdatedBy string     `json:"updatedBy,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}
type Draw struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	EntryIDs  []string  `json:"entryIds"`
	Active    bool      `json:"active"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	Reason    string    `json:"reason,omitempty"`
}
type Result struct {
	Entry      Entry    `json:"entry"`
	Position   *int     `json:"position"`
	Form1Total *Decimal `json:"form1Total"`
	Form2Total *Decimal `json:"form2Total"`
	Total      *Decimal `json:"total"`
	Rank       *int     `json:"rank"`
}
type Standings struct {
	Revision int64    `json:"revision"`
	Final    bool     `json:"final"`
	Ranked   []Result `json:"ranked"`
	Unranked []Result `json:"unranked"`
}
type ReasonInput struct {
	Reason string `json:"reason"`
}

func date(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }
func ValidateEvent(in EventInput) error {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 300 {
		return invalid("name is required (max 300 bytes)")
	}
	if _, err := date(in.Date); err != nil {
		return invalid("date must be an ISO date")
	}
	r := in.Rules
	if r.AccuracyMax <= 0 || r.PresentationMax <= 0 || r.AccuracyMax > 999999 || r.PresentationMax > 999999 || r.Precision != 2 {
		return invalid("positive score maxima and precision 2 are required")
	}
	if r.TiePolicy != "shared" {
		return invalid("tiePolicy must explicitly be shared")
	}
	if r.FormSelection != "division" && r.FormSelection != "entry" {
		return invalid("formSelection must be division or entry")
	}
	if r.AllowRepeatedForm == nil || strings.TrimSpace(r.Version) == "" {
		return invalid("allowRepeatedForm and rules version are required")
	}
	return nil
}
func ValidateDivision(e Event, in DivisionInput) error {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 300 || in.CompetitionType != "individual" {
		return invalid("named individual division required")
	}
	if in.Gender != "male" && in.Gender != "female" && in.Gender != "mixed" {
		return invalid("gender must be male, female or mixed")
	}
	if in.BirthDateFrom == nil && in.BirthDateTo == nil {
		return invalid("at least one explicit birth date bound is required")
	}
	for _, v := range []*string{in.BirthDateFrom, in.BirthDateTo} {
		if v != nil {
			if _, err := date(*v); err != nil || *v > e.Date {
				return invalid("invalid birth date bound")
			}
		}
	}
	if in.BirthDateFrom != nil && in.BirthDateTo != nil && *in.BirthDateFrom > *in.BirthDateTo {
		return invalid("birth date bounds are reversed")
	}
	if len(in.AllowedFormCodes) == 0 || len(in.AllowedFormCodes) > 16 {
		return invalid("allowedFormCodes required")
	}
	seen := map[int]bool{}
	for _, c := range in.AllowedFormCodes {
		if c < 1 || c > 16 || seen[c] {
			return invalid("form codes must be distinct and between 1 and 16")
		}
		seen[c] = true
	}
	if e.Rules.FormSelection == "division" {
		return validateCodes(e.Rules, in, in.Form1Code, in.Form2Code, true)
	}
	if in.Form1Code != nil || in.Form2Code != nil {
		return invalid("entry selection cannot set division form codes")
	}
	return nil
}
func validateCodes(r Rules, d DivisionInput, a, b *int, required bool) error {
	allowed := func(v *int) bool {
		if v == nil {
			return !required
		}
		for _, c := range d.AllowedFormCodes {
			if c == *v {
				return true
			}
		}
		return false
	}
	if !allowed(a) || !allowed(b) {
		return invalid("select allowed form codes")
	}
	if a != nil && b != nil && *a == *b && !*r.AllowRepeatedForm {
		return invalid("repeated forms are not allowed")
	}
	return nil
}
