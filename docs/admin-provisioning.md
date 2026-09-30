# ساخت مدیر با دسترسی سرور

ثبت‌نام عمومی و ساخت سازمان از طریق HTTP غیرفعال‌اند و پاسخ 403 می‌دهند.
ساخت owner فقط با ابزار `cmd/create-admin` و دسترسی مستقیم به دیتابیس ممکن است.
ورود کاربران موجود و ساخت اعضا توسط مدیر مجاز سازمان همچنان برقرار است.
این تغییر حساب‌ها یا نشست‌های قبلی را حذف نمی‌کند؛ حساب‌های ناشناس قبلی باید جدا بررسی شوند.

## سرور Ubuntu

برای به‌روزرسانی نصب موجود در `/opt/tkdhub`، دستورهای زیر را با کاربر root اجرا کنید.
در صورت خطا ادامه ندهید. ابتدا نسخه جدید ساخته می‌شود؛ سپس فایل اجرایی جایگزین و سرویس دوباره اجرا می‌شود.
این تغییر به ساخت مجدد دیتابیس یا فرانت‌اند نیازی ندارد.

```bash
set -e
cd /opt/tkdhub
git pull --ff-only origin master
mkdir -p /opt/tkdhub/bin
docker run --rm \
  -v /opt/tkdhub/backend:/src \
  -v /opt/tkdhub/bin:/out \
  -w /src -e CGO_ENABLED=0 golang:1.27.1 \
  sh -c 'go build -o /out/api.new ./cmd/api && go build -o /out/create-admin ./cmd/create-admin'
mv /opt/tkdhub/bin/api.new /opt/tkdhub/bin/api
systemctl restart tkdhub
for attempt in $(seq 1 30); do
  if curl --fail --silent http://127.0.0.1:8080/health; then break; fi
  sleep 1
done
curl --fail http://127.0.0.1:8080/health
```

فایل `/etc/tkdhub.env` باید تنظیم DATABASE_URL همین سرور را داشته باشد.
ساخت حساب با یک دستور روی خود سرور:

```bash
sudo bash /opt/tkdhub/scripts/create-admin.sh
```

اسکریپت ایمیل و رمز را می‌پرسد. رمز در آرگومان‌های پردازش یا تاریخچه شل قرار نمی‌گیرد.
حساب تکراری رد می‌شود و رمز یا دسترسی حساب موجود تغییر نمی‌کند.
ابزار migration اجرا نمی‌کند؛ ابتدا باید دیتابیس برنامه راه‌اندازی شده باشد.

برای بررسی بسته‌بودن مسیرها، این درخواست‌ها باید HTTP 403 بدهند:

```bash
curl -i -X POST https://tkdhub.ir/api/v1/auth/register
curl -i -X POST https://tkdhub.ir/api/v1/organizations
```

## توسعه محلی ویندوز

از ریشه پروژه `./scripts/create-admin.ps1` را اجرا کنید.
این ابزار از تنظیمات دیتابیس محلی `backend/.env` استفاده می‌کند و دیگر حساب را با API عمومی نمی‌سازد.
برای سرور از SSH و اسکریپت لینوکس استفاده کنید.
