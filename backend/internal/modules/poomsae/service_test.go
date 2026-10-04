package poomsae

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }
func fixture() (Event, Division) {
	e := Event{EventInput: EventInput{Name: "جام", Date: "2026-10-03", Rules: Rules{AccuracyMax: 300, PresentationMax: 700, Precision: 2, TiePolicy: "shared", FormSelection: "entry", AllowRepeatedForm: ptr(false), Version: "test"}}}
	d := Division{ID: uuid.NewString(), DivisionInput: DivisionInput{Name: "رده", CompetitionType: "individual", Gender: "male", BirthDateFrom: ptr("2012-01-01"), BirthDateTo: ptr("2014-12-31"), AllowedFormCodes: []int{3, 5, 7, 8, 9, 10, 11}}, Status: "draft", Revision: 1, Entries: []Entry{}, Draws: []Draw{}}
	return e, d
}
func entryInput() EntryInput {
	return EntryInput{AthleteID: uuid.NewString(), FirstName: "علی", LastName: "نمونه", TeamName: "تیم", BirthDate: "2013-01-01", Gender: "male", Status: "active"}
}
func score(a, b, c, d Decimal) ScoreInput {
	return ScoreInput{Form1: FormScore{Code: ptr(3), Accuracy: ptr(a), Presentation: ptr(b)}, Form2: FormScore{Code: ptr(5), Accuracy: ptr(c), Presentation: ptr(d)}}
}
func TestExactDecimalContract(t *testing.T) {
	for _, s := range []string{`"-1"`, `1.2`, `"1.234"`, `"NaN"`, `"1e2"`, `"۱.۲"`, `null`, `"10000"`, `" 1.00"`} {
		var v Decimal
		if json.Unmarshal([]byte(s), &v) == nil {
			t.Errorf("accepted %s", s)
		}
	}
	for s, want := range map[string]Decimal{`"0"`: 0, `"2.5"`: 250, `"0.01"`: 1, `"9999.99"`: 999999} {
		var v Decimal
		if err := json.Unmarshal([]byte(s), &v); err != nil || v != want {
			t.Fatalf("%s: %d %v", s, v, err)
		}
	}
	e, d := fixture()
	a, _ := d.AddEntry(e, entryInput())
	id := a.ID
	_ = d.Draw("actor", "")
	if err := d.SetScores(e, id, "actor", score(10, 20, 10, 20)); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(d.Standings())
	var decoded map[string]any
	_ = json.Unmarshal(b, &decoded)
	total := decoded["ranked"].([]any)[0].(map[string]any)["total"]
	if total != "0.60" {
		t.Fatalf("inexact total: %v", total)
	}
}
func TestBirthBoundsAndNonContinuousForms(t *testing.T) {
	e, d := fixture()
	if err := ValidateEvent(e.EventInput); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDivision(e, d.DivisionInput); err != nil {
		t.Fatal(err)
	}
	for _, dob := range []string{"2012-01-01", "2014-12-31"} {
		in := entryInput()
		in.BirthDate = dob
		if err := d.ValidateEntry(e, in); err != nil {
			t.Fatal(err)
		}
	}
	for _, dob := range []string{"2011-12-31", "2015-01-01", "2013-02-29", "2027-01-01"} {
		in := entryInput()
		in.BirthDate = dob
		if d.ValidateEntry(e, in) == nil {
			t.Errorf("accepted %s", dob)
		}
	}
	a, _ := d.AddEntry(e, entryInput())
	id := a.ID
	_ = d.Draw("actor", "")
	bad := score(250, 600, 260, 610)
	bad.Form1.Code = ptr(4)
	if !errors.Is(d.SetScores(e, id, "actor", bad), ErrInvalid) {
		t.Fatal("gap form accepted")
	}
	bad.Form1.Code = ptr(5)
	if d.SetScores(e, id, "actor", bad) == nil {
		t.Fatal("repeated form accepted")
	}
	bad = score(301, 600, 260, 610)
	if d.SetScores(e, id, "actor", bad) == nil {
		t.Fatal("score over maximum accepted")
	}
}
func TestDrawLifecycleAndZeroScores(t *testing.T) {
	e, d := fixture()
	if d.Draw("actor", "") == nil {
		t.Fatal("empty draw accepted")
	}
	ids := map[string]bool{}
	for i := 0; i < 20; i++ {
		a, err := d.AddEntry(e, entryInput())
		if err != nil {
			t.Fatal(err)
		}
		ids[a.ID] = true
	}
	if err := d.Draw("actor", ""); err != nil {
		t.Fatal(err)
	}
	for _, id := range d.ActiveDraw().EntryIDs {
		if !ids[id] {
			t.Fatal("duplicate/unknown entry")
		}
		delete(ids, id)
	}
	if len(ids) != 0 {
		t.Fatal("entries missing")
	}
	if d.Draw("actor", "") == nil {
		t.Fatal("redraw without reason")
	}
	if err := d.Draw("actor", "corrected"); err != nil {
		t.Fatal(err)
	}
	if len(d.Draws) != 2 || d.Draws[0].Active {
		t.Fatal("history lost")
	}
	if err := d.InvalidateDraw("roster change"); err != nil {
		t.Fatal(err)
	}
	if d.ActiveDraw() != nil || d.Status != "draft" {
		t.Fatal("draw not invalidated")
	}
	_ = d.Draw("actor", "new roster")
	id := d.Entries[0].ID
	partial := ScoreInput{Form1: FormScore{Code: ptr(3), Accuracy: ptr(Decimal(0))}, Form2: FormScore{Code: ptr(5)}}
	if err := d.SetScores(e, id, "actor", partial); err != nil {
		t.Fatal(err)
	}
	if len(d.Standings().Ranked) != 0 {
		t.Fatal("incomplete score ranked")
	}
	if d.Draw("actor", "retry") == nil || d.InvalidateDraw("retry") == nil || d.Finalize() == nil {
		t.Fatal("scoring locks bypassed")
	}
	if err := d.SetScores(e, id, "actor", score(0, 0, 0, 0)); err != nil {
		t.Fatal("filling missing fields should not require a correction reason", err)
	}
	if got := d.Standings(); len(got.Ranked) != 1 || *got.Ranked[0].Total != 0 {
		t.Fatal("zero confused with missing")
	}
	cleared := ScoreInput{Reason: "clear mistake"}
	if err := d.SetScores(e, id, "actor", cleared); err != nil {
		t.Fatal(err)
	}
	if !d.EverScored || d.Draw("actor", "retry") == nil {
		t.Fatal("clearing scores unlocked draw")
	}
}
func TestSharedRanksFinalizationAndCorrection(t *testing.T) {
	e, d := fixture()
	for i := 0; i < 5; i++ {
		_, _ = d.AddEntry(e, entryInput())
	}
	_ = d.Draw("actor", "")
	values := []Decimal{300, 250, 250, 200}
	for i, v := range values {
		if err := d.SetScores(e, d.Entries[i].ID, "actor", score(v, 600, v, 600)); err != nil {
			t.Fatal(err)
		}
	}
	ranks := []int{}
	for _, r := range d.Standings().Ranked {
		ranks = append(ranks, *r.Rank)
	}
	if !reflect.DeepEqual(ranks, []int{1, 2, 2, 4}) {
		t.Fatal(ranks)
	}
	absent := d.Entries[4].EntryInput
	absent.Status = "absent"
	absent.Reason = "did not attend"
	if err := d.UpdateEntry(e, d.Entries[4].ID, absent); err != nil {
		t.Fatal(err)
	}
	if err := d.Finalize(); err != nil {
		t.Fatal(err)
	}
	if !d.Standings().Final {
		t.Fatal("not final")
	}
	change := score(100, 600, 100, 600)
	if d.SetScores(e, d.Entries[0].ID, "actor", change) == nil {
		t.Fatal("final score edited")
	}
	if d.Reopen("") == nil {
		t.Fatal("reopen without reason")
	}
	_ = d.Reopen("review")
	if d.SetScores(e, d.Entries[0].ID, "actor", change) == nil {
		t.Fatal("correction without reason")
	}
	change.Reason = "transcription"
	if err := d.SetScores(e, d.Entries[0].ID, "actor", change); err != nil {
		t.Fatal(err)
	}
	if d.Standings().Ranked[3].Entry.ID != d.Entries[0].ID {
		t.Fatal("ranking stale")
	}
}
func TestDivisionFormsAndIdentityLocks(t *testing.T) {
	e, d := fixture()
	e.Rules.FormSelection = "division"
	d.Form1Code = ptr(3)
	d.Form2Code = ptr(5)
	if err := ValidateDivision(e, d.DivisionInput); err != nil {
		t.Fatal(err)
	}
	in := entryInput()
	a, _ := d.AddEntry(e, in)
	id := a.ID
	if _, err := d.AddEntry(e, in); err == nil {
		t.Fatal("duplicate accepted")
	}
	_ = d.Draw("actor", "")
	in.AthleteID = uuid.NewString()
	if d.UpdateEntry(e, id, in) == nil {
		t.Fatal("identity changed")
	}
	s := score(250, 600, 260, 610)
	s.Form1.Code = nil
	s.Form2.Code = nil
	if err := d.SetScores(e, id, "actor", s); err != nil {
		t.Fatal(err)
	}
	if *d.Standings().Ranked[0].Total != 1720 {
		t.Fatal("bad total")
	}
	s.Form1.Code = ptr(7)
	if d.SetScores(e, id, "actor", s) == nil {
		t.Fatal("wrong division form accepted")
	}
}
