# راهنمای فنی نصب و اجرای پروژه

## معماری و پورت‌ها

| بخش | فناوری | آدرس پیش‌فرض |
| --- | --- | --- |
| Frontend | Vue 3 + TypeScript + Vite | `http://127.0.0.1:5173` |
| Backend API | Go + Gin | `http://127.0.0.1:8080` |
| Database | PostgreSQL | `127.0.0.1:5432` |

در محیط توسعه، Vite تمام درخواست‌های `/api` را به بک‌اند روی پورت `8080` proxy می‌کند. به همین دلیل معمولاً نیازی به تعریف آدرس API در فرانت‌اند نیست.

## پیش‌نیازها

- Go 1.27.1 مطابق `backend/go.mod` یا نسخه‌ی جدیدتر سازگار
- Node.js 20 یا جدیدتر و npm
- PostgreSQL 15 یا جدیدتر
- Git اختیاری است و فقط برای مدیریت نسخه‌ها استفاده می‌شود
- Docker Desktop فقط در صورتی لازم است که PostgreSQL را با Docker اجرا کنید

نسخه‌ها را بررسی کنید:

```powershell
go version
node --version
npm --version
psql --version
```

## ۱. اجرای PostgreSQL

یکی از دو روش زیر را انتخاب کنید.

### دیتابیس آماده‌ی همین لپ‌تاپ

روی سیستم فعلی، PostgreSQL 17 و دیتابیس توسعه‌ی پروژه از قبل آماده‌اند. بعد از هر بار روشن‌شدن ویندوز این دستور را از ریشه‌ی پروژه اجرا کنید:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web
.\scripts\start-database.ps1
```

فایل `backend/.env` نیز برای اتصال به همین دیتابیس روی `127.0.0.1:55432` تنظیم شده است. پس از آماده‌شدن دیتابیس، اجرای بک‌اند فقط به این دو دستور نیاز دارد:

```powershell
cd backend
go run ./cmd/api
```

### روش اول: PostgreSQL نصب‌شده روی سیستم

پس از اجرای سرویس PostgreSQL، دیتابیس را بسازید:

```powershell
psql -U postgres -c "CREATE DATABASE taekwondo;"
```

اگر دیتابیس از قبل وجود داشته باشد، اجرای دوباره‌ی دستور لازم نیست.

### روش دوم: Docker

```powershell
docker run --name taekwondo-postgres `
  -e POSTGRES_USER=postgres `
  -e POSTGRES_PASSWORD=postgres `
  -e POSTGRES_DB=taekwondo `
  -p 5432:5432 `
  -d postgres:17
```

در اجراهای بعدی همان کانتینر را روشن کنید:

```powershell
docker start taekwondo-postgres
```

## ۲. تنظیم و اجرای بک‌اند

در PowerShell جدید وارد پوشه‌ی بک‌اند شوید:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\backend
Copy-Item .env.example .env
```

فایل `.env` را با اطلاعات دیتابیس خود تنظیم کنید. نمونه‌ی مناسب برای Docker بالا:

```dotenv
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/taekwondo?sslmode=disable
PORT=8080
AUTO_MIGRATE=true
CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

`AUTO_MIGRATE=true` هنگام شروع برنامه migrationهای اجرا نشده را به‌ترتیب اعمال می‌کند. در محیط production بهتر است migration با دستور جدا اجرا شود و سپس مقدار آن `false` باشد:

```powershell
go run ./cmd/migrate
go run ./cmd/api
```

برای توسعه‌ی محلی که migration خودکار فعال است، فقط این دستور کافی است:

```powershell
go run ./cmd/api
```

سلامت سرور را در ترمینال دیگری بررسی کنید:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health
```

پاسخ سالم:

```json
{"status":"up"}
```

## ۳. نصب و اجرای فرانت‌اند

در PowerShell دیگری اجرا کنید:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\frontend
npm install
npm run dev -- --host 0.0.0.0
```

سپس این آدرس را باز کنید:

```text
http://127.0.0.1:5173/
```

استفاده از `--host 0.0.0.0` باعث می‌شود دستگاه‌های دیگر شبکه‌ی محلی نیز بتوانند با IP کامپیوتر میزبان به برنامه وصل شوند. برای این حالت، پورت `5173` باید در Firewall مجاز باشد.

## حساب اولیه و ورود

در دیتابیس تازه، ابتدا حساب کاربری را با API ثبت‌نام بسازید. سپس با token پاسخ، سازمان را ایجاد کنید؛ سازنده‌ی سازمان با نقش `owner` و تمام دسترسی‌های مدیریتی عضو آن می‌شود. endpoint ثبت‌نام:

```text
POST http://127.0.0.1:8080/api/v1/auth/register
```

نمونه‌ی PowerShell:

```powershell
$body = @{
  name = 'مدیر مسابقات'
  email = 'admin@example.com'
  password = 'ChangeThisPassword123!'
} | ConvertTo-Json

$session = Invoke-RestMethod `
  -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/auth/register `
  -ContentType 'application/json' `
  -Body $body

$organization = @{ name = 'هیئت تکواندو' } | ConvertTo-Json
Invoke-RestMethod `
  -Method Post `
  -Uri http://127.0.0.1:8080/api/v1/organizations `
  -Headers @{ Authorization = "Bearer $($session.token)" } `
  -ContentType 'application/json' `
  -Body $organization
```

نام فیلدهای دقیق پاسخ و دسترسی‌ها در [مستند سطح دسترسی](../backend/docs/access-control.md) آمده است. رمز نمونه را برای محیط واقعی تغییر دهید و اطلاعات ورود را داخل مخزن ثبت نکنید.

## URLهای کاربردی

| صفحه | URL |
| --- | --- |
| ورود و پنل مدیریت | `http://127.0.0.1:5173/` |
| تورنومنت مشخص | `http://127.0.0.1:5173/tournaments/{id}` |
| لیگ مشخص | `http://127.0.0.1:5173/leagues/{id}` |
| نمایش عمومی زمین‌ها | `http://127.0.0.1:5173/OVR` |
| OVR یک تورنومنت مشخص | `http://127.0.0.1:5173/OVR?tournamentId={id}` |
| Health check | `http://127.0.0.1:8080/health` |

صفحه‌ی OVR عمومی است و به ورود نیاز ندارد. اگر `tournamentId` ارسال نشود، بک‌اند تورنومنت مناسب در حال اجرا را انتخاب می‌کند.

## ترتیب صحیح شروع و توقف

برای شروع:

1. PostgreSQL
2. Backend
3. Frontend

برای توقف فرانت و بک‌اند در ترمینال هرکدام `Ctrl+C` بزنید. اگر از Docker استفاده می‌کنید:

```powershell
docker stop taekwondo-postgres
```

## اجرای تست‌ها

### بک‌اند

تست‌های عادی:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\backend
go test ./...
go vet ./...
```

تست‌های متصل به PostgreSQL به `TEST_DATABASE_URL` نیاز دارند. بهتر است یک دیتابیس مخصوص تست بسازید:

```powershell
psql -U postgres -c "CREATE DATABASE taekwondo_test;"
$env:TEST_DATABASE_URL = 'postgres://postgres:postgres@127.0.0.1:5432/taekwondo_test?sslmode=disable'
go test ./... -count=1
```

تست‌ها schema جداگانه می‌سازند، ولی برای جلوگیری از خطر روی داده‌های مسابقه از دیتابیس production به‌عنوان `TEST_DATABASE_URL` استفاده نکنید.

### فرانت‌اند

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\frontend
npm test
npm run build
```

`npm run build` علاوه بر build، TypeScript را نیز بررسی می‌کند.

## build برای استقرار

بک‌اند:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\backend
go build -o bin\taekwondo-api.exe ./cmd/api
```

فرانت‌اند:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\frontend
npm ci
npm run build
```

خروجی فرانت در `frontend/dist` ساخته می‌شود. در production باید این پوشه با یک وب‌سرور مانند Nginx سرو شود، مسیرهای SPA به `index.html` برگردند و درخواست‌های `/api` به بک‌اند proxy شوند.

## خطاهای رایج

### پیام `database unavailable`

- از روشن بودن PostgreSQL مطمئن شوید.
- نام دیتابیس، نام کاربری، رمز و پورت `DATABASE_URL` را بررسی کنید.
- آدرس health را پس از اجرای بک‌اند دوباره امتحان کنید.

### خطای `ERR_CONNECTION_REFUSED` روی پورت 5173

فرانت‌اند اجرا نیست. در پوشه‌ی `frontend` دستور `npm run dev -- --host 0.0.0.0` را اجرا کنید.

### پاسخ 404 یا خطای proxy برای `/api`

بک‌اند باید روی پورت `8080` فعال باشد. تنظیم proxy در `frontend/vite.config.ts` قرار دارد.

### نتیجه در OVR یا لیگ تغییر نمی‌کند

- ثبت نتیجه باید در برگه‌ی TA بدون خطا کامل شود.
- OVR هر دو ثانیه اطلاعات تازه را دریافت می‌کند.
- مطمئن شوید OVR مربوط به همان تورنومنت را با `tournamentId` باز کرده‌اید.
- پاسخ endpoint عمومی را مستقیم بررسی کنید:

```text
http://127.0.0.1:8080/api/v1/public/ovr?tournamentId={id}
```

### پورت اشغال است

در Windows پردازش استفاده‌کننده از پورت را پیدا کنید:

```powershell
Get-NetTCPConnection -LocalPort 8080,5173 -ErrorAction SilentlyContinue |
  Select-Object LocalPort,State,OwningProcess
```

## مستندات تکمیلی API

- [تورنومنت و قرعه‌کشی](../backend/docs/tournament-stage.md)
- [لیگ و محاسبه‌ی امتیازات](../backend/docs/league-stage.md)
- [حساب‌ها و سطح دسترسی](../backend/docs/access-control.md)
