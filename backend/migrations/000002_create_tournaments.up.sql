-- اگر قبلاً در پروژه فعال شده، اجرای دوباره مشکلی ایجاد نمی‌کند.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS tournaments (
                                           id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,
    event_date DATE NOT NULL,

    courts INTEGER NOT NULL DEFAULT 1
    CHECK (courts > 0),

    gender TEXT NOT NULL
    CHECK (gender IN ('male', 'female')),

    age_category TEXT NOT NULL
    CHECK (
              age_category IN (
              'خردسالان',
              'نونهالان',
              'نوجوانان',
              'امیدها',
              'بزرگسالان'
            )
        ),

    format TEXT NOT NULL
        CHECK (format IN ('grandPrix', 'team')),

    /*
     * معادل Tournament.athletes
     */
    athletes JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(athletes) = 'array'),

    /*
     * معادل Tournament.matches
     */
    matches JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(matches) = 'array'),

    /*
     * معادل Tournament.courtAssignment
     *
     * نمونه:
     * {
     *   "1": [1, 2, 3],
     *   "2": [4, 5]
     * }
     */
    court_assignment JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(court_assignment) = 'object'),

    /*
     * معادل Tournament.eliminationMatches
     */
    elimination_matches JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(elimination_matches) = 'array'),

    /*
     * ارتباط‌های احتمالی با ماژول‌های دیگر.
     * فعلاً Foreign Key اضافه نشده تا migration به جدول‌های
     * league / week / team_tournament وابسته نباشد.
     */
    league_id UUID NULL,
    stage_order INTEGER NULL
        CHECK (stage_order IS NULL OR stage_order >= 0),
    week_id UUID NULL,
    team_tournament_id UUID NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tournaments_event_date
    ON tournaments (event_date);

CREATE INDEX IF NOT EXISTS idx_tournaments_gender_age_category
    ON tournaments (gender, age_category);

CREATE INDEX IF NOT EXISTS idx_tournaments_format
    ON tournaments (format);

CREATE INDEX IF NOT EXISTS idx_tournaments_league_id
    ON tournaments (league_id)
    WHERE league_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tournaments_week_id
    ON tournaments (week_id)
    WHERE week_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tournaments_team_tournament_id
    ON tournaments (team_tournament_id)
    WHERE team_tournament_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tournaments_athletes_gin
    ON tournaments
    USING GIN (athletes);

CREATE INDEX IF NOT EXISTS idx_tournaments_matches_gin
    ON tournaments
    USING GIN (matches);

CREATE INDEX IF NOT EXISTS idx_tournaments_elimination_matches_gin
    ON tournaments
    USING GIN (elimination_matches);
