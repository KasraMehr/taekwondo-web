export type Decimal = string
export type PoomsaeRules = {
  accuracyMax: Decimal; presentationMax: Decimal; precision: 2; tiePolicy: 'shared'
  formSelection: 'division' | 'entry'; allowRepeatedForm: boolean; version: string
}
export type EventInput = { name: string; date: string; rules: PoomsaeRules }
export type PoomsaeEvent = EventInput & { id: string; revision: number; createdAt: string }
export type DivisionInput = {
  name: string; competitionType: 'individual'; gender: 'male' | 'female' | 'mixed'; belt: string
  birthDateFrom: string | null; birthDateTo: string | null; allowedFormCodes: number[]
  form1Code: number | null; form2Code: number | null
}
export type EntryInput = {
  athleteId: string; firstName: string; lastName: string; teamName: string; birthDate: string
  gender: 'male' | 'female'; belt: string; status: 'active' | 'absent' | 'withdrawn'; reason?: string
}
export type FormScore = { code: number | null; accuracy: Decimal | null; presentation: Decimal | null }
export type Scores = { form1: FormScore; form2: FormScore; reason?: string }
export type PoomsaeEntry = EntryInput & { id: string; scores: Scores; updatedAt: string; updatedBy?: string }
export type PoomsaeDraw = { id: string; version: number; entryIds: string[]; active: boolean; createdAt: string; reason?: string }
export type Result = { entry: PoomsaeEntry; position: number | null; form1Total: Decimal | null; form2Total: Decimal | null; total: Decimal | null; rank: number | null }
export type Standings = { revision: number; final: boolean; ranked: Result[]; unranked: Result[] }
export type DivisionSummary = DivisionInput & { id: string; eventId: string; status: 'draft' | 'drawn' | 'scoring' | 'finalized'; everScored: boolean; revision: number }
export type PoomsaeDivision = DivisionSummary & { entries: PoomsaeEntry[]; draws: PoomsaeDraw[]; results: Result[]; standings: Standings }
export type PoomsaeOptions = {
  categories: { key: string; name: string; allowedFormCodes: number[] }[]
  permissions: { manage: boolean; draw: boolean; score: boolean; override: boolean; audit: boolean; createEvent: boolean; manageEvent: boolean; athletesRead: boolean; athletesManage: boolean }
}
export type Profile = { id: string; name: string; birthDate?: string | null; gender?: 'male' | 'female'; isActive: boolean; clubId?: string | null }
export type SheetMode = 'results' | 'blank' | 'standings'
export type Sheet = { event: PoomsaeEvent; division: DivisionInput; divisionId: string; revision: number; draw: PoomsaeDraw | null; final: boolean; mode: SheetMode; generatedAt: string; rows: Result[] }
export type ScoreDraft = { value: Scores; base: Scores; conflict: boolean; error: string }
