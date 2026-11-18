// Run against staging with k6, e.g.:
// k6 run -e BASE_URL=https://staging.example.com -e TOURNAMENT_ID=uuid scripts/public-results-load.js
// OVR scenario: add -e VIEW=ovr -e INTERVAL=2
import http from 'k6/http';
import { check, sleep } from 'k6';
export const options = {
  stages: [{duration:'30s',target:100},{duration:'1m',target:500},{duration:'5m',target:500},{duration:'30s',target:0}],
  thresholds: { http_req_failed:['rate<0.01'], http_req_duration:['p(95)<500'] },
};
export default function () {
  if (!__ENV.BASE_URL || !__ENV.TOURNAMENT_ID) throw new Error('BASE_URL and TOURNAMENT_ID are required');
  const start=Date.now();
  const path=__ENV.VIEW==='live'?`/api/v1/public/competitions/${__ENV.TOURNAMENT_ID}/live`:__ENV.VIEW==='ovr'?`/api/v1/public/ovr?tournamentId=${__ENV.TOURNAMENT_ID}`:`/api/v1/public/competitions/${__ENV.TOURNAMENT_ID}`;
  const response=http.get(__ENV.BASE_URL.replace(/\/$/,'')+path);
  check(response, {'public response succeeds': r=>r.status===200});
  sleep(Math.max(0.1,Number(__ENV.INTERVAL||5)-(Date.now()-start)/1000));
}
