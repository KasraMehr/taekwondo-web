CREATE TABLE match_live_scores (
 tournament_id UUID NOT NULL,
 match_id UUID NOT NULL,
 version BIGINT NOT NULL DEFAULT 1,
 data JSONB NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(tournament_id,match_id),
 FOREIGN KEY(tournament_id,match_id) REFERENCES tournament_matches(tournament_id,id) ON DELETE CASCADE
);
