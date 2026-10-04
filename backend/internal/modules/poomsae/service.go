package poomsae

import (
	"crypto/rand"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (d *Division) Entry(id string) (*Entry, error) {
	for i := range d.Entries {
		if d.Entries[i].ID == id {
			return &d.Entries[i], nil
		}
	}
	return nil, ErrNotFound
}
func (d *Division) ActiveDraw() *Draw {
	for i := range d.Draws {
		if d.Draws[i].Active {
			return &d.Draws[i]
		}
	}
	return nil
}
func (d *Division) ValidateEntry(e Event, in EntryInput) error {
	if _, err := uuid.Parse(in.AthleteID); err != nil {
		return invalid("athleteId must reference a profile")
	}
	if strings.TrimSpace(in.FirstName) == "" || strings.TrimSpace(in.LastName) == "" || strings.TrimSpace(in.TeamName) == "" || len(in.FirstName) > 150 || len(in.LastName) > 150 || len(in.TeamName) > 300 {
		return invalid("firstName, lastName and teamName are required")
	}
	if _, err := date(in.BirthDate); err != nil || in.BirthDate > e.Date {
		return invalid("invalid birthDate")
	}
	if d.BirthDateFrom != nil && in.BirthDate < *d.BirthDateFrom || d.BirthDateTo != nil && in.BirthDate > *d.BirthDateTo {
		return invalid("athlete is outside birth date bounds")
	}
	if in.Gender != "male" && in.Gender != "female" {
		return invalid("athlete gender required")
	}
	if d.Gender != "mixed" && d.Gender != in.Gender {
		return invalid("athlete gender does not match division")
	}
	if d.Belt != "" && d.Belt != in.Belt {
		return invalid("athlete belt does not match division")
	}
	if in.Status != "active" && in.Status != "absent" && in.Status != "withdrawn" {
		return invalid("invalid attendance status")
	}
	return nil
}
func (d *Division) AddEntry(e Event, in EntryInput) (*Entry, error) {
	if d.Status != "draft" || d.EverScored {
		return nil, conflict("entries require a draft without scores")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if err := d.ValidateEntry(e, in); err != nil {
		return nil, err
	}
	for _, a := range d.Entries {
		if a.AthleteID == in.AthleteID {
			return nil, conflict("athlete already registered")
		}
	}
	a := Entry{ID: uuid.NewString(), EntryInput: in, UpdatedAt: time.Now().UTC()}
	d.Entries = append(d.Entries, a)
	return &d.Entries[len(d.Entries)-1], nil
}
func (d *Division) UpdateEntry(e Event, id string, in EntryInput) error {
	if d.Status == "finalized" {
		return conflict("reopen finalized division first")
	}
	old, err := d.Entry(id)
	if err != nil {
		return err
	}
	if in.Status == "" {
		in.Status = old.Status
	}
	if err = d.ValidateEntry(e, in); err != nil {
		return err
	}
	if in.AthleteID != old.AthleteID {
		return conflict("profile identity cannot be changed; remove and register in draft")
	}
	if d.Status != "draft" && (in.BirthDate != old.BirthDate || in.Gender != old.Gender || in.Belt != old.Belt) {
		return conflict("eligibility is locked after draw")
	}
	if in.Status != old.Status && d.Status != "draft" {
		if strings.TrimSpace(in.Reason) == "" {
			return invalid("attendance change requires a reason")
		}
		if in.Status == "active" {
			found := false
			if draw := d.ActiveDraw(); draw != nil {
				for _, v := range draw.EntryIDs {
					found = found || v == id
				}
			}
			if !found {
				return conflict("athlete was not included in the draw")
			}
		}
	}
	if d.EverScored && strings.TrimSpace(in.Reason) == "" {
		return invalid("correction requires a reason")
	}
	old.EntryInput = in
	old.UpdatedAt = time.Now().UTC()
	return nil
}
func (d *Division) DeleteEntry(id string) error {
	if d.Status != "draft" || d.EverScored {
		return conflict("invalidate unscored draw before deleting entries")
	}
	for i, a := range d.Entries {
		if a.ID == id {
			d.Entries = append(d.Entries[:i], d.Entries[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
func (d *Division) Draw(actor, reason string) error {
	if d.EverScored || d.Status == "finalized" {
		return conflict("draw locked after scoring")
	}
	if len(d.Draws) > 0 && strings.TrimSpace(reason) == "" {
		return invalid("redraw requires a reason")
	}
	ids := []string{}
	for _, a := range d.Entries {
		if a.Status == "active" {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return conflict("no eligible athletes")
	}
	for i := len(ids) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		ids[i], ids[j] = ids[j], ids[i]
	}
	for i := range d.Draws {
		d.Draws[i].Active = false
	}
	d.Draws = append(d.Draws, Draw{ID: uuid.NewString(), Version: len(d.Draws) + 1, EntryIDs: ids, Active: true, CreatedBy: actor, CreatedAt: time.Now().UTC(), Reason: reason})
	d.Status = "drawn"
	return nil
}
func (d *Division) InvalidateDraw(reason string) error {
	if d.EverScored || d.Status != "drawn" {
		return conflict("only an unscored draw can be invalidated")
	}
	if strings.TrimSpace(reason) == "" {
		return invalid("reason required")
	}
	for i := range d.Draws {
		d.Draws[i].Active = false
	}
	d.Status = "draft"
	return nil
}
func HasScores(s ScoreInput) bool {
	return s.Form1.Accuracy != nil || s.Form1.Presentation != nil || s.Form2.Accuracy != nil || s.Form2.Presentation != nil
}
func (d *Division) SetScores(e Event, id, actor string, in ScoreInput) error {
	if d.Status != "drawn" && d.Status != "scoring" {
		return conflict("scores require an open drawn division")
	}
	a, err := d.Entry(id)
	if err != nil {
		return err
	}
	if a.Status != "active" {
		return conflict("absent or withdrawn athlete cannot be scored")
	}
	found := false
	if draw := d.ActiveDraw(); draw != nil {
		for _, v := range draw.EntryIDs {
			found = found || v == id
		}
	}
	if !found {
		return conflict("athlete missing from draw")
	}
	if e.Rules.FormSelection == "division" {
		if in.Form1.Code != nil && *in.Form1.Code != *d.Form1Code || in.Form2.Code != nil && *in.Form2.Code != *d.Form2Code {
			return invalid("form codes differ from division selection")
		}
		in.Form1.Code = d.Form1Code
		in.Form2.Code = d.Form2Code
	}
	if err = validateCodes(e.Rules, d.DivisionInput, in.Form1.Code, in.Form2.Code, HasScores(in)); err != nil {
		return err
	}
	for _, f := range []FormScore{in.Form1, in.Form2} {
		if f.Accuracy != nil && (*f.Accuracy < 0 || *f.Accuracy > e.Rules.AccuracyMax) || f.Presentation != nil && (*f.Presentation < 0 || *f.Presentation > e.Rules.PresentationMax) {
			return invalid("score exceeds configured range")
		}
	}
	if correctsScores(a.Scores, in) && strings.TrimSpace(in.Reason) == "" {
		return invalid("editing existing scores requires a reason")
	}
	a.Scores = in
	a.UpdatedBy = actor
	a.UpdatedAt = time.Now().UTC()
	if HasScores(in) {
		d.EverScored = true
		d.Status = "scoring"
	}
	return nil
}

// Filling empty fields is normal entry; replacing saved values is a correction.
func correctsScores(old, next ScoreInput) bool {
	changed := func(a, b *Decimal) bool { return a != nil && (b == nil || *a != *b) }
	for i, a := range []FormScore{old.Form1, old.Form2} {
		b := []FormScore{next.Form1, next.Form2}[i]
		if changed(a.Accuracy, b.Accuracy) || changed(a.Presentation, b.Presentation) {
			return true
		}
		if (a.Accuracy != nil || a.Presentation != nil) && a.Code != nil && (b.Code == nil || *a.Code != *b.Code) {
			return true
		}
	}
	return false
}
func formTotal(f FormScore) *Decimal {
	if f.Code == nil || f.Accuracy == nil || f.Presentation == nil {
		return nil
	}
	n := *f.Accuracy + *f.Presentation
	return &n
}
func (d *Division) Results() []Result {
	positions := map[string]int{}
	if draw := d.ActiveDraw(); draw != nil {
		for i, id := range draw.EntryIDs {
			positions[id] = i + 1
		}
	}
	out := make([]Result, 0, len(d.Entries))
	for _, a := range d.Entries {
		r := Result{Entry: a, Form1Total: formTotal(a.Scores.Form1), Form2Total: formTotal(a.Scores.Form2)}
		if p, ok := positions[a.ID]; ok {
			r.Position = &p
		}
		if r.Form1Total != nil && r.Form2Total != nil {
			n := *r.Form1Total + *r.Form2Total
			r.Total = &n
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Position == nil {
			return b.Position == nil && a.Entry.ID < b.Entry.ID
		}
		if b.Position == nil {
			return true
		}
		return *a.Position < *b.Position
	})
	return out
}
func (d *Division) Standings() Standings {
	out := Standings{Revision: d.Revision, Final: d.Status == "finalized", Ranked: []Result{}, Unranked: []Result{}}
	for _, r := range d.Results() {
		if r.Entry.Status == "active" && r.Total != nil {
			out.Ranked = append(out.Ranked, r)
		} else {
			out.Unranked = append(out.Unranked, r)
		}
	}
	sort.SliceStable(out.Ranked, func(i, j int) bool { return *out.Ranked[i].Total > *out.Ranked[j].Total })
	rank := 0
	for i := range out.Ranked {
		if i == 0 || *out.Ranked[i].Total != *out.Ranked[i-1].Total {
			rank = i + 1
		}
		n := rank
		out.Ranked[i].Rank = &n
	}
	return out
}
func (d *Division) Finalize() error {
	if d.Status != "drawn" && d.Status != "scoring" {
		return conflict("division is not open")
	}
	if d.ActiveDraw() == nil {
		return conflict("draw required")
	}
	for _, r := range d.Results() {
		if r.Entry.Status == "active" && r.Total == nil {
			return conflict("active athlete has incomplete scores")
		}
	}
	d.Status = "finalized"
	return nil
}
func (d *Division) Reopen(reason string) error {
	if d.Status != "finalized" {
		return conflict("division is not finalized")
	}
	if strings.TrimSpace(reason) == "" {
		return invalid("reason required")
	}
	if d.EverScored {
		d.Status = "scoring"
	} else {
		d.Status = "drawn"
	}
	return nil
}
