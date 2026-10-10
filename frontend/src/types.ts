import type { AgeCategory, Gender } from './data/categories'

export interface Athlete {
    id: string;
    number?: number;
    sourceLeagueAthleteId?: string;
    name: string;
    club: string;
    weightCategory: string;
    ranking?: number;
    weighedIn?: boolean;
    weighIn?: WeighIn
}

export interface Match {
    id: string
    athlete1Id: string | null
    athlete2Id: string | null
    winnerId?: string
    court: number
    order: number
    status?: 'pending' | 'ongoing' | 'completed'
    result?: MatchResult
    bracketIndex?: number
    weightCategory: string
    round: number
    side: 'left' | 'right' | 'final' | 'semifinal'
    nextMatchId?: string
    isBye?: boolean
    nextSlot?: 'athlete1' | 'athlete2',
    matchNumber?: number
    sourceCourt?: number
    sourceOrder?: number
    sourceLabel?: string
    movedAt?: string
}

export type WinType =
    | 'PTF'     // برتری امتیازی
    | 'PTG'     // اختلاف ۱۲ امتیاز
    | 'RSC'     // توقف مسابقه
    | 'SUP'     // برتری فنی (تصمیم داوران)
    | 'GDP'     // امتیاز طلایی
    | 'WDR'     // انصراف
    | 'DSQ'     // اخراج
    | 'PUN'     // اخراج به دلیل ۵ گام‌جوم
    | 'RSC_INJ' // مصدومیت
    | 'WO'      // عدم حضور / اسلات خالی

export type RoundEndReason = 'PTF' | 'PTG' | 'PUN' | 'GDP' | 'SUP' | null

/** امتیازات تفکیک‌شده یک ورزشکار در یک راند */
export interface RoundScore {
    punch: number
    bodyKick: number
    headKick: number
    turningBodyKick: number
    turningHeadKick: number
    gamJeom: number
    gamJeomLate: number
}

export interface MatchRound {
    number: number
    isGoldenPoint?: boolean
    blue: RoundScore
    red: RoundScore
    winner: 'blue' | 'red' | null
    endedBy?: RoundEndReason
    manualWinner?: boolean   // برنده دستی (SUP یا اصلاح داور)
}

export interface MatchResult {
    winType: WinType
    blueId: string | null
    redId: string | null
    winnerId: string | null
    winnerCorner?: 'blue' | 'red'
    rounds: MatchRound[]
    note?: string
    refereeId?: string
    recordedAt: string
    updatedAt?: string
}

export type TournamentFormat = 'grandPrix' | 'team'


export interface Tournament {
    id: string;
    name: string;
    date: string;
    courts: number;
    gender: Gender;
    ageCategory: AgeCategory;
    athletes: Athlete[];
    matches: Match[];
    courtAssignment: Record<string, number[]>;
    leagueId?: string | null;
    stageOrder?: number | null;
    groupId?: string | null
    weekId?: string | null;
    format: TournamentFormat;
    teamTournamentId?: string | null;
    eliminationMatches?: EliminationMatch[];
    createdAt: string;
    updatedAt: string;
}

export type EliminationRound = 'semifinal' | 'final' | 'third_place'

export interface EliminationMatch {
    id: string
    round: EliminationRound
    court: number
    athlete1Label: string
    athlete2Label: string
    scheduledTime?: string
    winnerId?: string | null
}

// ─────────────────────────────────────────────────────────────
// انواع سیستم لیگ
// ─────────────────────────────────────────────────────────────

// ─── تنظیمات امتیازدهی ───

export interface ScoringConfig {
    /** امتیاز مدال طلا */
    gold: number
    /** امتیاز مدال نقره */
    silver: number
    /** امتیاز مدال برنز (رده ۳ و ۴ هر دو برنز) */
    bronze: number
    /** امتیاز وزن‌کشی (۰ = غیرفعال) */
    weighInPoint: number
    /** امتیاز هر برد انفرادی (۰ = غیرفعال) */
    winPoint: number
    /** آیا امتیاز وزن‌کشی محاسبه شود */
    countWeighIn: boolean
    /** آیا امتیاز برد انفرادی محاسبه شود */
    countWin: boolean
    /** نگه داشتن رنکینگ بازیکن بین مراحل (برای قرعه‌کشی سیدبندی مرحله بعد) */
    keepPlayerRankingBetweenStages: boolean
}

// ─── رکورد امتیاز یک بازیکن ───

export interface PlayerPointsRecord {
    athleteId: string
    name: string
    /** باشگاهِ زمان کسب امتیاز (برای محاسبه امتیاز تیمی) */
    club: string
    weightCategory: string
    totalPoints: number
    gold: number
    silver: number
    bronze: number
    weighInPoints: number
    winPoints: number
    /** تفاضل راند (مجموع راندهای برده منهای باخته) */
    roundDiff: number
    /** برد ۲-۰ */
    wins2_0: number
    /** برد ۲-۱ */
    wins2_1: number
    /** باخت ۰-۲ */
    losses0_2: number
    /** باخت ۱-۲ */
    losses1_2: number
    /** تعداد برد کل */
    totalWins: number
    /** تعداد باخت کل */
    totalLosses: number
    /** رتبه فعلی (اگر رنکینگ نگه داشته شده باشد) */
    ranking?: number
}

// ─── رکورد امتیاز یک تیم (باشگاه) ───

export interface TeamPointsRecord {
    club: string
    totalPoints: number
    gold: number
    silver: number
    bronze: number
    weighInPoints: number
    winPoints: number
    roundDiff: number
    wins2_0: number
    wins2_1: number
    losses0_2: number
    losses1_2: number
    totalWins: number
    totalLosses: number
    /** تعداد ورزشکاران این باشگاه */
    athleteCount: number
    /** آیا صعود کرده (برای مرحله بعد) */
    promoted?: boolean
}

// ─── هفته ───

export interface Week {
    id: string
    name: string
    /** بازه زمانی (مثلاً "۱ تا ۱۵ مهر") */
    dateRange?: string
    /** ترتیب هفته در مرحله */
    order: number
    /** لینک به Tournament این هفته */
    tournamentId?: string
    /** قفل نتیجه (بعد از پایان هفته) */
    locked?: boolean
}

// ─── مرحله ───

export interface Stage {
    id: string
    name: string
    /** ترتیب مرحله در فصل (۱ یا ۲) */
    order: number
    /** هفته‌های این مرحله */
    weeks: Week[]
    /** امتیاز بازیکنان در این مرحله (صفر می‌شود اگر صعود باشد) */
    playerPoints: Record<string, PlayerPointsRecord>
    /** امتیاز تیم‌ها در این مرحله */
    teamPoints: Record<string, TeamPointsRecord>
    /** آیا این مرحله کامل شده (برای صعود خودکار) */
    completed?: boolean
    /** تیم‌های صعودکرده به مرحله بعد */
    promotedClubs?: string[]
}

// ─── فصل ───

export interface Season {
    id: string
    name: string
    /** تنظیمات امتیازدهی فصل */
    scoringConfig: ScoringConfig
    /** مراحل فصل (۱ یا ۲) */
    stages: Stage[]
    /** امتیاز تجمیعی بازیکنان کل فصل (جمع همه مراحل) */
    seasonPlayerPoints: Record<string, PlayerPointsRecord>
    /** امتیاز تجمیعی تیم‌ها کل فصل */
    seasonTeamPoints: Record<string, TeamPointsRecord>
    /** فصل کامل شده؟ */
    completed?: boolean
    groups: LeagueGroup[]
}

// ─── لیگ ───

export interface League {
    id: string
    name: string
    gender: Gender
    ageCategory: AgeCategory
    /** فقط یک فصل */
    season: Season
    /** تنظیمات صعود (چند تیم برتر صعود کنند) */
    promotionRules: {
        /** تعداد تیم‌های برتر که به مرحله بعد صعود می‌کنند */
        teamsToPromote: number
        /** خودکار صعود کند؟ */
        autoPromote: boolean
    }

    format: TournamentFormat
    teamDefaults?: { blindLineup: boolean }
    clubs: LeagueClub[]
    athletes: LeagueAthlete[]
    createdAt: string
    updatedAt?: string
}

// ─── خروجی رتبه‌بندی تیم ───

export interface TeamRankingEntry {
    rank: number
    club: string
    totalPoints: number
    gold: number
    silver: number
    bronze: number
    weighInPoints: number
    winPoints: number
    roundDiff: number
    wins2_0: number
    wins2_1: number
    losses0_2: number
    losses1_2: number
    totalWins: number
    totalLosses: number
    athleteCount: number
    promoted?: boolean
    groupId?: string | null
    groupName?: string | null
}

// ─── خروجی رتبه‌بندی بازیکن ───

export interface PlayerRankingEntry {
    rank: number
    athleteId: string
    name: string
    club: string
    weightCategory: string
    totalPoints: number
    gold: number
    silver: number
    bronze: number
    weighInPoints: number
    winPoints: number
    roundDiff: number
    wins2_0: number
    wins2_1: number
    losses0_2: number
    losses1_2: number
    totalWins: number
    totalLosses: number
    groupId?: string | null
    groupName?: string | null
}

// ─────────────────────────────────────────────────────────────
// تورنمنت تیمی (Team / Grand Prix Team Event)
// ─────────────────────────────────────────────────────────────

/** اسلات وزنی؛ ۸ یا ۱۰ اسلات در هر مواجهه */
export interface TeamWeightSlot {
    slotNo: number            // 1..10
    title: string             // «‎-58 کیلوگرم»
    minKg?: number
    maxKg?: number
    gender?: Gender
}

export interface TeamMember {
    athleteId: string
    /** اسلات پیش‌فرض (مربی می‌تواند در هر مواجهه تغییر دهد) */
    slotNo?: number
    weighInKg?: number
    status?: 'active' | 'injured' | 'suspended'
    /** اسلات‌های مجازِ جایگزین (مثلاً مجاز به بازی در وزن بالاتر) */
    eligibleSlots?: number[]
}

export interface Team {
    id: string
    name: string
    club?: string
    color?: string            // برای برگه چاپی و UI
    seed?: number
    roster: TeamMember[]
}

/** آرنج اعلام‌شده مربی برای یک مواجهه */
export interface EncounterLineup {
    teamId: string
    entries: Record<number, string | null>
    meta?: Record<number, LineupSlotMeta>
    status?: LineupStatus
    submittedBy?: string
    submittedAt?: string
    lockedAt?: string
}

export interface EncounterBout {
    id: string
    slotNo: number
    homeAthleteId: string | null
    awayAthleteId: string | null
    court?: number
    status: 'pending' | 'ongoing' | 'completed'
    winnerSide: 'home' | 'away' | null
    homeRoundsWon: number
    awayRoundsWon: number
    /** کدام کرنر، تیم میزبان بوده (پیش‌فرض blue) */
    homeCorner: 'blue' | 'red'
    /** همان MatchResult انفرادی؛ موتور امتیازدهی مشترک است */
    result?: MatchResult
    isTieBreaker?: boolean
}

export type EncounterStatus =
    | 'scheduled'       // قرعه‌کشی شده، آرنج نیامده
    | 'lineupLocked'    // آرنج قفل و مبارزه‌ها ساخته شده
    | 'ongoing'
    | 'completed'

export interface TeamEncounter {
    id: string
    matchId?: string
    roundNo: number          // دور قرعه‌کشی
    slotIndex: number        // نوبت زمانی در جدول (۱ به بالا)
    court: number
    homeTeamId: string
    awayTeamId: string
    status: EncounterStatus
    lineups: EncounterLineup[]
    bouts: EncounterBout[]
    /** مجموع‌ها؛ همیشه از bouts بازمحاسبه می‌شوند، هرگز افزایشی */
    homeBouts: number
    awayBouts: number
    homeRounds: number
    awayRounds: number
    homePoints: number
    awayPoints: number
    revealedAt?: string
    groupId?: string
    finalizedAt?: string
    movedAt?: string
    winner?: 'home' | 'away' | null
}

export type TeamTieBreaker =
    | 'points'        // امتیاز تیمی
    | 'boutDiff'      // تفاضل برد انفرادی
    | 'roundDiff'     // تفاضل راند
    | 'headToHead'    // رودررو
    | 'boutsWon'      // مجموع بردهای انفرادی
    | 'roundsWon'
    | 'fewestDsq'     // کمترین اخراج/خطای انتظامی
    | 'draw'          // قرعه (ترتیب فعلی)

export interface TeamScoringConfig {
    pointsWin: number         // 3
    pointsDraw: number        // 1
    pointsLoss: number        // 0
    /** راندهای اعطاشده به برنده در WO/انصراف بدون امتیاز راند */
    walkoverRounds: number    // 2
    tieBreakers: TeamTieBreaker[]
    /** آرنج حریف تا ارسال هر دو مربی پنهان بماند */
    blindLineup: boolean
}

export interface TeamDrawOptions {
    /** حداقل فاصله نوبتی بین دو بازی یک تیم */
    minRest: number
    /** چند زمین به هر مواجهه اختصاص یابد (اسلات‌ها موازی اجرا شوند) */
    matsPerEncounter: number
    /** موازی‌سازی چند مواجهه هم‌زمان */
    parallel: boolean
    drawType?: 'random' | 'seeded'
    iterations?: number
}

export type TeamTournamentStatus =
    | 'draft' | 'teamsRegistered' | 'rosterLocked'
    | 'drawn' | 'inProgress' | 'finished'

export interface TeamTournament {
    id: string
    name: string
    date: string
    courts: number
    gender: Gender
    ageCategory: AgeCategory
    status: TeamTournamentStatus
    scoring: TeamScoringConfig
    drawOptions: TeamDrawOptions
    weightSlots: TeamWeightSlot[]
    athletes: Athlete[]
    teams: Team[]
    groups: TeamGroup[]
    matches: TeamMatch[]
    courtsPerGroup: Record<string, number[]>
    encounters: TeamEncounter[]
    leagueId?: string | null
    stageOrder?: number | null
    weekId?: string | null
    sourceTournamentId?: string | null
    createdAt: string
    updatedAt: string
}

export interface TeamStandingEntry {
    rank: number
    teamId: string
    teamName: string
    played: number
    wins: number
    draws: number
    losses: number
    points: number
    boutsWon: number
    boutsLost: number
    boutDiff: number
    roundsWon: number
    roundsLost: number
    roundDiff: number
    dsqAgainst: number
}
export interface ScheduleViolation {
    encounterId: string
    kind: 'conflict' | 'rest' | 'courtOverflow'
    message: string
}
export interface TeamMatch {
    id: string
    groupId: string
    teamAId: string
    teamBId: string
    round: number
    courtId: number | null
    slotId: number | null
    scoreA: number | null
    scoreB: number | null
}

export interface TeamGroup {
    id: string
    name: string
    teamIds: string[]
}

export const isScheduled = (m: TeamMatch) =>
    m.courtId !== null && m.slotId !== null

export interface TimeSlot {
    id: number
    label: string
}

export const TIME_SLOTS = [
    { id: 1, label: '۸ تا ۱۰' },
    { id: 2, label: '۱۰ تا ۱۲' },
    { id: 3, label: '۱ تا ۳' },
    { id: 4, label: '۳ تا ۵' },
    { id: 5, label: '۵ تا ۷' },
] as const

export const COURTS = [1, 2, 3, 4] as const

export const MAX_COURTS_PER_GROUP = 2

// ─────────────────────────────────────────────────────────────
// ترکیب تیم (Lineup)
// ─────────────────────────────────────────────────────────────

/** وضعیت ترکیب اعلام‌شده مربی */
export type LineupStatus =
    | 'draft'        // مربی در حال ویرایش
    | 'submitted'    // ثبت نهایی توسط مربی (در blindLineup هنوز پنهان)
    | 'locked'       // قفل توسط برگزارکننده؛ bouts ساخته شده

/** دلیل خالی ماندن یک اسلات وزنی — نتیجه‌اش WO برای حریف است */
export type ForfeitReason =
    | 'noAthlete'     // ورزشکاری در این وزن ندارد
    | 'injury'
    | 'weighInFail'   // وزن‌کشی ناموفق
    | 'suspended'
    | 'withdrawn'

/** متادیتای یک اسلات در ترکیب؛ کلید نگهدارنده‌اش slotNo است */
export interface LineupSlotMeta {
    forfeit?: ForfeitReason | null
    /** یادداشت مربی */
    note?: string | null
    /** وزن‌کشی ثبت‌شده مخصوص همین مواجهه */
    weighInKg?: number
}

export interface LeagueBundle {
    kind: "tkd-league-bundle"
    version: number
    exportedAt: string
    league: League
    tournaments: Tournament[]
}

// ─── باشگاه ثبت‌شده در لیگ ───
export interface LeagueClub {
    id: string
    /** نام باشگاه؛ کلید اتصال با Athlete.club و کلید رکوردهای امتیاز */
    name: string
    color?: string
    logo?: string
    /** گروهی که باشگاه در آن قرار دارد (null = تخصیص‌نیافته) */
    groupId?: string | null
    active?: boolean
    createdAt: string
}

// ─── گروه ───
export interface LeagueGroup {
    id: string
    /** «گروه A» */
    name: string
    order: number
    /** نام باشگاه‌های عضو (منبع حقیقت برای عضویت) */
    clubs: string[]
    athleteIds?: string[]
    /** ورزشکارانی که با وجود عضویت باشگاهشان، از این گروه مستثنا شده‌اند */
    excludedAthleteIds?: string[]
    capacity ?: number
}

export interface GroupTeamRankings {
    groupId: string
    groupName: string
    order: number
    rows: TeamRankingEntry[]
}

export interface LeagueAthlete {
    id: string
    name: string
    nationalId: string | null
    clubId: string | null
    /** اسنپ‌شات نام باشگاه در لحظهٔ ثبت؛ منبع حقیقت clubId است */
    club: string | null
    isActive: boolean
    weight: number | null
    weightCategory: string | null
    gender: Gender | null
    birthDate: string | null
    birthYear: number | null
    beltDegree: string | null
    memberCode: string | null
    groupId: string | null
    createdAt: string
    updatedAt: string
}

export type ImportRowStatus = "create" | "update" | "skip" | "error"

export interface AthleteImportRow {
    excelRow: number
    status: ImportRowStatus
    /** ورزشکار آمادهٔ درج/به‌روزرسانی. برای status === "error" مقدار null است */
    athlete: LeagueAthlete | null
    /** id ورزشکار موجود در صورت update/skip */
    matchedAthleteId: string | null
    /** نام نرمال‌شدهٔ باشگاه؛ اگر باشگاه وجود نداشته باشد در clubsToCreate هم می‌آید */
    clubName: string
    errors: string[]
    warnings: string[]
}

export interface AthleteImportPreview {
    leagueId: string
    /** برای تشخیص کهنه‌شدن پیش‌نمایش */
    baseUpdatedAt: string
    sheetName: string
    unknownHeaders: string[]
    rows: AthleteImportRow[]
    clubsToCreate: string[]
    counts: Record<ImportRowStatus, number>
}

export interface AthleteImportOptions {
    /** باشگاه‌های ناموجود ساخته شوند یا ردیف خطا بخورد */
    createMissingClubs?: boolean
    /** رفتار روی ورزشکار تکراری */
    onDuplicate?: "skip" | "update"
    /** کد ملی اجباری باشد */
    requireNationalId?: boolean
    /** کد ملی نامعتبر: خطا یا فقط هشدار */
    strictNationalId?: boolean
    /** ورزشکار به گروهِ باشگاهش هم منتسب شود */
    assignGroupFromClub?: boolean
}

export interface AthleteImportResult {
    created: number
    updated: number
    skipped: number
    failed: number
    clubsCreated: string[]
}

export interface AthleteDraft {
    name: string
    clubId: string | null
    weightCategory: string | null
    nationalId?: string | null
    birthDate?: string | null
    birthYear?: number | null
    beltDegree?: string | null
    memberCode?: string | null
    groupId?: string | null
    isActive?: boolean
    gender?: Gender
}

export interface WeighInAttempt {
    no: number
    weightKg: number
    ok: boolean
    at: string
}

export interface WeighIn {
    status: 'pending' | 'passed' | 'failed'
    weightKg: number | null
    withTolerance: boolean
    attempts: WeighInAttempt[]
    signature: { imagePath: string; signedAt: string } | null
}


