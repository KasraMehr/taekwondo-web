import test from 'node:test'
import assert from 'node:assert/strict'
import * as XLSX from 'xlsx'
import { parseRoster, rosterWorkbook } from '../src/utils/leagueAthleteExcel'

const weights=['-54','-58','+87']
test('free athletes require a coach, while team athletes do not',()=>{
  const result=parseRoster(roundTrip(rosterWorkbook(weights,teams,[
    {name:'Missing coach',weightCategory:'-54',coachName:'   '},
    {name:'Team member',weightCategory:'-54',teamId:'team-1'},
  ])),weights,teams)
  assert.equal(result.rows.length,1)
  assert.equal(result.rows[0].payload.name,'Team member')
  assert.equal(result.errors.length,1)
  assert.match(result.errors[0],/مربی/)
})
test('coach survives Excel export and import for team and free athletes',()=>{
  const athletes=[{name:'Free',weightCategory:'-54',coachName:'Free Coach'}, {name:'Member',weightCategory:'-58',teamId:'team-1',coachName:'Team Coach'}]
  const result=parseRoster(roundTrip(rosterWorkbook(weights,teams,athletes)),weights,teams)
  assert.deepEqual(result.errors,[])
  assert.deepEqual(result.rows.map(r=>[r.payload.name,r.payload.coachName,r.payload.teamId]),[['Free','Free Coach',null],['Member','Team Coach','team-1']])
})
const teams=[{id:'club-1',teamId:'team-1',name:'تیم امید',groupId:'group-1'}]
const roundTrip=(wb:XLSX.WorkBook)=>XLSX.read(XLSX.write(wb,{type:'buffer',bookType:'xlsx'}),{type:'buffer'})

test('league template matches tournament weight sheets with an additional team column',()=>{
  const wb=roundTrip(rosterWorkbook(weights,teams))
  assert.deepEqual(wb.SheetNames,['وزن 1 (-54)','وزن 2 (-58)','وزن 3 (+87)'])
  const rows=XLSX.utils.sheet_to_json(wb.Sheets[wb.SheetNames[0]],{header:1})
  assert.deepEqual(rows[0],['نام','باشگاه','وزن','رنکینگ','تیم','مربی'])
  assert.equal(rows.length,17)
  assert.deepEqual(parseRoster(wb,weights,teams),{rows:[],errors:[]})
})

test('120 fake athletes survive real XLSX serialization, including free players and rankings',()=>{
  const athletes=Array.from({length:120},(_,i)=>({name:`ورزشکار آزمایشی ${i}`,coachName:`مربی ${i}`,weightCategory:weights[i%3],teamId:i%2?'team-1':null,ranking:i%5?i+1:null,club:i%2?'تیم امید':''}))
  const result=parseRoster(roundTrip(rosterWorkbook(weights,teams,athletes)),weights,teams)
  assert.deepEqual(result.errors,[])
  assert.equal(result.rows.length,120)
  for(const a of athletes){
    const p=result.rows.find(r=>r.payload.name===a.name)!.payload
    assert.equal(p.teamId,a.teamId)
    assert.equal(p.clubId,a.teamId?'club-1':null)
    assert.equal(p.groupId,a.teamId?'group-1':null)
    assert.equal(p.weightCategory,a.weightCategory)
    assert.equal(p.ranking,a.ranking)
  }
  const again=parseRoster(roundTrip(rosterWorkbook(weights,teams,athletes)),weights,teams,athletes)
  assert.equal(again.rows.length,0)
  assert.equal(again.errors.length,120)
})

test('import checks bad rows, Persian digits, sheet weight fallback and empty team despite club',()=>{
  const wb=XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb,XLSX.utils.aoa_to_sheet([
    ['نام','باشگاه','وزن','رنکینگ','تیم','مربی'],
    ['آزاد','تیم امید','','۲','','مربی آزاد'],
    ['عضو','','-۵۴','۳','تيم اميد'],
    ['عضو','','-54',3,'تیم امید'],
    ['ناشناخته','','-54','','تیم ناموجود'],
    ['وزن بد','','-99','',''],
    ['رنک بد','','-54',1.5,''],
  ]),'وزن 1 (-54)')
  const result=parseRoster(roundTrip(wb),weights,teams)
  assert.equal(result.rows.length,2)
  assert.equal(result.errors.length,4)
  assert.equal(result.rows[0].payload.teamId,null)
  assert.equal(result.rows[0].payload.ranking,2)
  assert.equal(result.rows[0].payload.weightCategory,'-54')
  assert.equal(result.rows[1].payload.teamId,'team-1')
})

test('tournament files without a team column import as free players',()=>{
  const wb=XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb,XLSX.utils.aoa_to_sheet([['name','club','weight','ranking','coach'],['Test','Club','-58',1,'Coach']]),'Athletes')
  const result=parseRoster(roundTrip(wb),weights,teams)
  assert.deepEqual(result.errors,[])
  assert.equal(result.rows[0].payload.teamId,null)
})
