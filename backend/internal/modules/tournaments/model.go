package tournaments

import "time"

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

func (g Gender) IsValid() bool {
	return g == GenderMale || g == GenderFemale
}

func (g Gender) Label() string {
	switch g {
	case GenderMale:
		return "پسران / مردان"
	case GenderFemale:
		return "دختران / زنان"
	default:
		return ""
	}
}

type AgeCategory string

const (
	AgeCategoryKhordsalan  AgeCategory = "خردسالان"
	AgeCategoryNonehalan   AgeCategory = "نونهالان"
	AgeCategoryNojavanan   AgeCategory = "نوجوانان"
	AgeCategoryOmidha      AgeCategory = "امیدها"
	AgeCategoryBozorgsalan AgeCategory = "بزرگسالان"
)

func (a AgeCategory) IsValid() bool {
	switch a {
	case AgeCategoryKhordsalan,
		AgeCategoryNonehalan,
		AgeCategoryNojavanan,
		AgeCategoryOmidha,
		AgeCategoryBozorgsalan:
		return true
	default:
		return false
	}
}

// AgeCategories ترتیب نمایش رده‌های سنی در اپ را حفظ می‌کند.
// در هر فراخوانی یک slice جدید برمی‌گرداند.
func AgeCategories() []AgeCategory {
	return []AgeCategory{
		AgeCategoryKhordsalan,
		AgeCategoryNonehalan,
		AgeCategoryNojavanan,
		AgeCategoryOmidha,
		AgeCategoryBozorgsalan,
	}
}

// دسته‌های وزنی مطابق تنظیمات فعلی اپ.
// این جدول عمداً export نشده تا خارج از پکیج تغییر نکند.
var weightCategories = map[AgeCategory]map[Gender][]string{
	AgeCategoryKhordsalan: {
		GenderMale: {
			"-26", "-28", "-30", "-33", "-36",
			"-40", "-44", "-48", "-52", "+52",
		},
		GenderFemale: {
			"-26", "-28", "-30", "-33", "-36",
			"-40", "-44", "-48", "-52", "+52",
		},
	},
	AgeCategoryNonehalan: {
		GenderMale: {
			"-33", "-37", "-41", "-45", "-49",
			"-53", "-57", "-61", "-65", "+65",
		},
		GenderFemale: {
			"-29", "-33", "-37", "-41", "-44",
			"-47", "-51", "-55", "-59", "+59",
		},
	},
	AgeCategoryNojavanan: {
		GenderMale: {
			"-45", "-48", "-51", "-55", "-59",
			"-63", "-68", "-73", "-78", "+78",
		},
		GenderFemale: {
			"-42", "-44", "-46", "-49", "-52",
			"-55", "-59", "-63", "-68", "+68",
		},
	},
	AgeCategoryOmidha: {
		GenderMale: {
			"-54", "-58", "-63", "-68",
			"-74", "-80", "-87", "+87",
		},
		GenderFemale: {
			"-46", "-49", "-53", "-57",
			"-62", "-67", "-73", "+73",
		},
	},
	AgeCategoryBozorgsalan: {
		GenderMale: {
			"-54", "-58", "-63", "-68",
			"-74", "-80", "-87", "+87",
		},
		GenderFemale: {
			"-46", "-49", "-53", "-57",
			"-62", "-67", "-73", "+73",
		},
	},
}

// WeightOptions معادل weightOptions در اپ است.
// یک کپی برمی‌گرداند تا تغییر خروجی، جدول اصلی را تغییر ندهد.
// برای ورودی نامعتبر، خروجی [] است نه nil.
func WeightOptions(age AgeCategory, gender Gender) []string {
	options := weightCategories[age][gender]

	result := make([]string, len(options))
	copy(result, options)

	return result
}

// IsValidWeightCategory معادل تابع اعتبارسنجی اپ است.
func IsValidWeightCategory(
	age AgeCategory,
	gender Gender,
	value string,
) bool {
	for _, option := range weightCategories[age][gender] {
		if option == value {
			return true
		}
	}

	return false
}

type TournamentFormat string

const (
	TournamentFormatGrandPrix TournamentFormat = "grandPrix"
	TournamentFormatTeam      TournamentFormat = "team"
)

type Tournament struct {
	Settings           *TournamentSettings `json:"settings,omitempty"`
	Revision           int64               `json:"revision"`
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Date               time.Time           `json:"date"`
	Courts             int                 `json:"courts"`
	Gender             Gender              `json:"gender"`
	AgeCategory        AgeCategory         `json:"ageCategory"`
	Athletes           []TournamentAthlete `json:"athletes"`
	Matches            []Match             `json:"matches"`
	CourtAssignment    map[string][]int    `json:"courtAssignment"`
	Format             TournamentFormat    `json:"format"`
	EliminationMatches []EliminationMatch  `json:"eliminationMatches"`

	LeagueID         *string `json:"leagueId,omitempty"`
	StageOrder       *int    `json:"stageOrder,omitempty"`
	WeekID           *string `json:"weekId,omitempty"`
	GroupID          *string `json:"groupId,omitempty"`
	TeamTournamentID *string `json:"teamTournamentId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TournamentAthlete struct {
	ProfileID      *string  `json:"profileId,omitempty"`
	Coach          string   `json:"coach,omitempty"`
	ID             string   `json:"id"`
	Number         int      `json:"number,omitempty"`
	Name           string   `json:"name"`
	Club           string   `json:"club"`
	WeightCategory string   `json:"weightCategory"`
	Ranking        *int     `json:"ranking,omitempty"`
	WeighedIn      bool     `json:"weighedIn"`
	WeighIn        *WeighIn `json:"weighIn,omitempty"`
}

type Match struct {
	Day            int          `json:"day"`
	ID             string       `json:"id"`
	Athlete1ID     *string      `json:"athlete1Id"`
	Athlete2ID     *string      `json:"athlete2Id"`
	WinnerID       *string      `json:"winnerId,omitempty"`
	Court          int          `json:"court"`
	Order          int          `json:"order"`
	Status         *MatchStatus `json:"status,omitempty"`
	Result         *MatchResult `json:"result,omitempty"`
	BracketIndex   *int         `json:"bracketIndex,omitempty"`
	WeightCategory string       `json:"weightCategory"`
	Round          int          `json:"round"`
	Side           MatchSide    `json:"side"`
	NextMatchID    *string      `json:"nextMatchId,omitempty"`
	IsBye          *bool        `json:"isBye,omitempty"`
	NextSlot       *NextSlot    `json:"nextSlot,omitempty"`
	MatchNumber    *int         `json:"matchNumber,omitempty"`
	SourceCourt    *int         `json:"sourceCourt,omitempty"`
	SourceOrder    *int         `json:"sourceOrder,omitempty"`
	SourceLabel    *string      `json:"sourceLabel,omitempty"`
	MovedAt        *time.Time   `json:"movedAt,omitempty"`
}

type MatchStatus string

const (
	MatchStatusPending   MatchStatus = "pending"
	MatchStatusOngoing   MatchStatus = "ongoing"
	MatchStatusCompleted MatchStatus = "completed"
)

type MatchSide string

const (
	MatchSideLeft      MatchSide = "left"
	MatchSideRight     MatchSide = "right"
	MatchSideFinal     MatchSide = "final"
	MatchSideSemifinal MatchSide = "semifinal"
)

type NextSlot string

const (
	NextSlotAthlete1 NextSlot = "athlete1"
	NextSlotAthlete2 NextSlot = "athlete2"
)

type WinType string

const (
	WinTypePTF    WinType = "PTF"
	WinTypePTG    WinType = "PTG"
	WinTypeRSC    WinType = "RSC"
	WinTypeSUP    WinType = "SUP"
	WinTypeGDP    WinType = "GDP"
	WinTypeWDR    WinType = "WDR"
	WinTypeDSQ    WinType = "DSQ"
	WinTypePUN    WinType = "PUN"
	WinTypeRSCInj WinType = "RSC_INJ"
	WinTypeWO     WinType = "WO"
)

type RoundEndReason string

const (
	RoundEndPTF RoundEndReason = "PTF"
	RoundEndPTG RoundEndReason = "PTG"
	RoundEndPUN RoundEndReason = "PUN"
	RoundEndGDP RoundEndReason = "GDP"
	RoundEndSUP RoundEndReason = "SUP"
)

type Corner string

const (
	CornerBlue Corner = "blue"
	CornerRed  Corner = "red"
)

type RoundScore struct {
	Punch           int `json:"punch"`
	BodyKick        int `json:"bodyKick"`
	HeadKick        int `json:"headKick"`
	TurningBodyKick int `json:"turningBodyKick"`
	TurningHeadKick int `json:"turningHeadKick"`
	GamJeom         int `json:"gamJeom"`
	GamJeomLate     int `json:"gamJeomLate"`
}

type MatchRound struct {
	Number        int             `json:"number"`
	IsGoldenPoint bool            `json:"isGoldenPoint"`
	Blue          RoundScore      `json:"blue"`
	Red           RoundScore      `json:"red"`
	Winner        *Corner         `json:"winner,omitempty"`
	EndedBy       *RoundEndReason `json:"endedBy,omitempty"`
	ManualWinner  bool            `json:"manualWinner"`
}

type MatchResult struct {
	WinType      WinType      `json:"winType"`
	BlueID       *string      `json:"blueId"`
	RedID        *string      `json:"redId"`
	WinnerID     *string      `json:"winnerId"`
	WinnerCorner *Corner      `json:"winnerCorner,omitempty"`
	Rounds       []MatchRound `json:"rounds"`
	Note         *string      `json:"note,omitempty"`
	RefereeID    *string      `json:"refereeId,omitempty"`
	RecordedAt   time.Time    `json:"recordedAt"`
	UpdatedAt    *time.Time   `json:"updatedAt,omitempty"`
}

type EliminationRound string

const (
	EliminationRoundSemifinal  EliminationRound = "semifinal"
	EliminationRoundFinal      EliminationRound = "final"
	EliminationRoundThirdPlace EliminationRound = "third_place"
)

type EliminationMatch struct {
	ID            string           `json:"id"`
	Round         EliminationRound `json:"round"`
	Court         int              `json:"court"`
	Athlete1Label string           `json:"athlete1Label"`
	Athlete2Label string           `json:"athlete2Label"`
	ScheduledTime *time.Time       `json:"scheduledTime,omitempty"`
	WinnerID      *string          `json:"winnerId,omitempty"`
}

type WeighIn struct {
	Status        WeighInStatus     `json:"status"`
	WeightKg      *float64          `json:"weightKg,omitempty"`
	WithTolerance bool              `json:"withTolerance"`
	Attempts      []WeighInAttempt  `json:"attempts"`
	Signature     *WeighInSignature `json:"signature,omitempty"`
}

type WeighInStatus string

const (
	WeighInPending WeighInStatus = "pending"
	WeighInPassed  WeighInStatus = "passed"
	WeighInFailed  WeighInStatus = "failed"
)

type WeighInAttempt struct {
	No       int       `json:"no"`
	WeightKg float64   `json:"weightKg"`
	OK       bool      `json:"ok"`
	At       time.Time `json:"at"`
}

type WeighInSignature struct {
	ImagePath string    `json:"imagePath"`
	SignedAt  time.Time `json:"signedAt"`
}
