const API='/api/v1'
export type Session={token:string;orgId:string;userId:string}

async function responseData(response: Response) {
  if (response.status === 204) return null
  const text = await response.text()
  if (!text) return null
  try { return JSON.parse(text) }
  catch { return { message: text.trim() || `خطای ${response.status}` } }
}

export class Api {
  constructor(public session:Session){}
  async call<T=any>(path:string,method='GET',body?:any):Promise<T>{
    const res=await fetch(path.startsWith('/api')?path:`${API}/organizations/${this.session.orgId}${path}`,{method,headers:{Authorization:`Bearer ${this.session.token}`,...(body!==undefined?{'Content-Type':'application/json'}:{})},body:body!==undefined?JSON.stringify(body):undefined})
    const data=await responseData(res)
    if(!res.ok)throw new Error(data?.message||`خطای ${res.status}`)
    return data as T
  }
}
export async function auth(path:string,body:any){const r=await fetch(`${API}/auth/${path}`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const d=await responseData(r);if(!r.ok)throw new Error(d?.message||'ورود ناموفق بود');if(!d?.token)throw new Error('پاسخ ورود از سرور معتبر نیست');return d}
export async function organizations(token:string){const r=await fetch(`${API}/organizations`,{headers:{Authorization:`Bearer ${token}`}});const d=await responseData(r);if(!r.ok)throw new Error(d?.message||'دریافت سازمان ناموفق بود');if(!Array.isArray(d))throw new Error('پاسخ سازمان‌ها از سرور معتبر نیست');return d}
