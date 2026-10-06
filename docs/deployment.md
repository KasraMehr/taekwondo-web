# Deploy روی سرور

پس از آماده‌سازی اولیه‌ی Ubuntu، باید Nginx ریشه‌ی سایت را روی `/opt/tkdhub/frontend/dist` تنظیم کرده باشد و `/etc/tkdhub.env` دارای `DATABASE_URL` دیتابیس برنامه باشد. PostgreSQL، Docker، Git، Nginx، `pg_dump` و سرویس systemd با نام `tkdhub` باید نصب و آماده باشند.

از SSH، با کاربر root:

```bash
sudo bash /opt/tkdhub/scripts/deploy.sh
```

اسکریپت به‌صورت پیش‌فرض `origin/master` را deploy می‌کند. این مراحل را انجام می‌دهد:

1. checkout را بررسی می‌کند و اگر تغییر ثبت‌نشده داشته باشد متوقف می‌شود.
2. `master` را فقط با fast-forward به‌روز می‌کند.
3. API، ابزار migration، ابزار ساخت مدیر، و فرانت‌اند production را می‌سازد.
4. پیش از تغییر دیتابیس یک backup با فرمت custom در `/var/backups/tkdhub` می‌سازد.
5. تمام migrationهای `.up.sql` که هنوز در `schema_migrations` ثبت نشده‌اند اجرا می‌کند.
6. API و فایل‌های فرانت‌اند را نصب، سرویس را restart و health endpoint را تا ۳۰ ثانیه بررسی می‌کند.

اگر build یا backup یا migration شکست بخورد، نسخه‌ی برنامه نصب نمی‌شود. اگر health check بعد از نصب شکست بخورد، فایل‌های API و فرانت‌اند قبلی بازگردانده می‌شوند. تغییر دیتابیس خودکار rollback نمی‌شود؛ backup پیش از migration حفظ می‌شود و مسیرش در خروجی script چاپ می‌شود. قبل از بازیابی دیتابیس، وضعیت خطا را بررسی کنید.

برای بررسی خروجی سرویس:

```bash
systemctl status tkdhub --no-pager -l
journalctl -u tkdhub -n 80 --no-pager
```

اگر Nginx از مسیر دیگری برای فایل‌های فرانت‌اند استفاده می‌کند، آن را با ریشه‌ی پیش‌فرض script هماهنگ کنید یا مسیر را به‌صورت environment variable بدهید:

```bash
FRONTEND_DIR=/var/www/tkdhub sudo -E bash /opt/tkdhub/scripts/deploy.sh
```

برای deploy شاخه‌ی دیگری می‌توان نام شاخه را به‌عنوان آرگومان داد؛ حالت معمول production همان `master` است.
