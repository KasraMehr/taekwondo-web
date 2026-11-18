package leagues

import (
	"backend/internal/modules/tournaments"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"strings"
)

var ErrStructureLocked = errors.New("پس از شروع اولین مبارزه، ساختار لیگ و گروه‌بندی قابل تغییر نیست")

func (s *Service) structureLocked(ctx context.Context, id string) (bool, error) {
	var locked bool
	err := s.db.QueryRow(ctx, `SELECT structure_locked FROM leagues WHERE id=$1 AND organization_id=$2 FOR UPDATE`, id, s.organizationID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return locked, err
}
func (s *Service) requireEditable(ctx context.Context, id string) error {
	locked, err := s.structureLocked(ctx, id)
	if err != nil {
		return err
	}
	if locked {
		return ErrStructureLocked
	}
	return nil
}

type EditInput struct {
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	AgeCategory string `json:"ageCategory"`
	SeasonName  string `json:"seasonName"`
	StageCount  int    `json:"stageCount"`
}

func (s *Service) Edit(ctx context.Context, id string, in EditInput) (*League, error) {
	if err := s.requireEditable(ctx, id); err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || !tournaments.Gender(in.Gender).IsValid() || !tournaments.AgeCategory(in.AgeCategory).IsValid() || in.StageCount < 1 || in.StageCount > 10 {
		return nil, ErrInvalid
	}
	l, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Gender != l.Gender || in.AgeCategory != l.AgeCategory {
		// Existing registrations must remain valid; never discard athletes or brackets to change a category.
		for _, a := range l.Athletes {
			if in.Gender != l.Gender || !tournaments.IsValidWeightCategory(tournaments.AgeCategory(in.AgeCategory), tournaments.Gender(in.Gender), a.WeightCategory) {
				return nil, fmt.Errorf("%w: ابتدا ورزشکاران ناسازگار با ردهٔ جدید را اصلاح کنید", ErrInvalid)
			}
		}
	}
	var linked bool
	err = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tournaments t JOIN league_weeks w ON w.id=t.week_id JOIN league_stages st ON st.id=w.stage_id WHERE t.league_id=$1 AND st.position>$2)`, id, in.StageCount).Scan(&linked)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, fmt.Errorf("%w: مراحل دارای مسابقه را نمی‌توان حذف کرد", ErrInvalid)
	}
	if _, err = s.db.Exec(ctx, `DELETE FROM league_stages WHERE league_id=$1 AND position>$2`, id, in.StageCount); err != nil {
		return nil, err
	}
	weeks := 1
	if len(l.Stages) > 0 && len(l.Stages[0].Weeks) > 0 {
		weeks = len(l.Stages[0].Weeks)
	}
	for i := len(l.Stages) + 1; i <= in.StageCount; i++ {
		sid := uuid.NewString()
		if _, err = s.db.Exec(ctx, `INSERT INTO league_stages(id,league_id,name,position) VALUES($1,$2,$3,$4)`, sid, id, fmt.Sprintf("مرحله %d", i), i); err != nil {
			return nil, err
		}
		for j := 1; j <= weeks; j++ {
			if _, err = s.db.Exec(ctx, `INSERT INTO league_weeks(league_id,stage_id,name,position) VALUES($1,$2,$3,$4)`, id, sid, fmt.Sprintf("هفته %d", j), j); err != nil {
				return nil, err
			}
		}
	}
	if _, err = s.db.Exec(ctx, `UPDATE leagues SET name=$2,gender=$3,age_category=$4,config=jsonb_set(config,'{seasonName}',to_jsonb($5::text)),revision=revision+1,updated_at=now() WHERE id=$1`, id, in.Name, in.Gender, in.AgeCategory, strings.TrimSpace(in.SeasonName)); err != nil {
		return nil, err
	}
	if _, err = s.db.Exec(ctx, `UPDATE tournaments SET gender=$2,age_category=$3,revision=revision+1,updated_at=now() WHERE league_id=$1`, id, in.Gender, in.AgeCategory); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}
