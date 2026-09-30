# Taekwondo tournament backend

> راهنمای یکپارچه‌ی اجرای دیتابیس، بک‌اند و فرانت‌اند در [مستند فنی پروژه](../docs/technical-setup.md) قرار دارد.

API تورنومنت تکواندو با Go، Gin و PostgreSQL. در مرحلهٔ فعلی، حساب و سازمان، پروفایل ورزشکار، تورنومنت انفرادی، وزن‌کشی، قرعه‌کشی، برنامهٔ یک یا دو روزه، سه روش تخصیص زمین، شماره‌گذاری و ثبت نتیجه آماده‌اند.

## اجرای محلی روی Windows

یک دیتابیس PostgreSQL بسازید و متغیرها را تنظیم کنید:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\backend
$env:DATABASE_URL = 'postgres://postgres:password@127.0.0.1:5432/taekwondo?sslmode=disable'
$env:AUTO_MIGRATE = 'true'
$env:PORT = '8080'
$env:CORS_ORIGINS = 'http://localhost:5173'
go run ./cmd/api
```

پس از نمایش سرور، این آدرس‌ها باید پاسخ بدهند:

```text
GET http://localhost:8080/
GET http://localhost:8080/health
```

در ترمینال دوم، تست کامل URLها را اجرا کنید:

```powershell
cd C:\Users\Kasra\Desktop\taekwondo_web\backend
.\scripts\smoke-tournament.ps1
```

اسکریپت یک حساب آزمایشی با ایمیل یکتا، سازمان، تورنومنت دو روزه و ۸ ورزشکار می‌سازد؛ وزن‌کشی، قرعه و شماره‌گذاری از ۱۰۱ را انجام می‌دهد و شناسه‌ها را چاپ می‌کند. داده‌ها در دیتابیس انتخاب‌شده باقی می‌مانند تا با Postman یا مرورگر بررسی شوند.

برای اجرای سرور روی پورت یا میزبان دیگری:

```powershell
.\scripts\smoke-tournament.ps1 -BaseUrl http://localhost:9090
```

جزئیات قرارداد API، تنظیمات و محدودیت‌های این مرحله در [docs/tournament-stage.md](docs/tournament-stage.md) آمده است.

## تست‌ها

```powershell
go test ./...
go vet ./...
```

تست PostgreSQL با schema جدا:

```powershell
$env:TEST_DATABASE_URL = 'postgres://postgres:password@127.0.0.1:5432/postgres?sslmode=disable'
go test ./... -count=1
```

مهاجرت بدون اجرای خودکار سرور:

```powershell
go run ./cmd/migrate
```

## احراز هویت در Postman

ابتدا حساب را طبق [راهنمای ساخت مدیر](../docs/admin-provisioning.md) با دسترسی دیتابیس بسازید. ثبت‌نام عمومی بسته است. سپس `POST /api/v1/auth/login` را با ایمیل و رمز بزنید و مقدار `token` پاسخ را برای درخواست‌های بعدی در Header بگذارید:

```text
Authorization: Bearer TOKEN
Content-Type: application/json
```

تمام مسیرهای تورنومنت زیر سازمان هستند:

```text
/api/v1/organizations/{organizationId}/tournaments
```

برای جلوگیری از ذخیرهٔ تغییر روی نسخهٔ قدیمی، در درخواست‌های تغییردهنده می‌توانید revision آخر پاسخ را بفرستید:

```text
If-Match: "12"
```

نسخهٔ قدیمی پاسخ `409 Conflict` می‌گیرد.

## Makefile قدیمی

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

API flows and request examples:

- [Tournament API](docs/tournament-stage.md)
- [League API](docs/league-stage.md)
- [Accounts and access control](docs/access-control.md)
