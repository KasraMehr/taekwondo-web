CREATE TABLE athletes (
                          id UUID PRIMARY KEY,

                          name VARCHAR(100) NOT NULL,

                          national_code VARCHAR(20),
                          birth_date DATE,
                          gender VARCHAR(10),
                          phone VARCHAR(30),
                          email VARCHAR(255),

                          is_active BOOLEAN NOT NULL DEFAULT TRUE,

                          created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                          CONSTRAINT athletes_name_not_blank
                              CHECK (LENGTH(BTRIM(name)) > 0),

                          CONSTRAINT athletes_gender_valid
                              CHECK (gender IN ('male', 'female'))
);

CREATE INDEX idx_athletes_created_at
    ON athletes (created_at DESC);

CREATE INDEX idx_athletes_is_active_created_at
    ON athletes (is_active, created_at DESC);
