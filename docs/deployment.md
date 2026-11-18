# Deploy روی سرور

## صفحات عمومی و ظرفیت بینندگان

مسیر `/competitions` بدون ورود، فهرست مسابقات و صفحه هر مسابقه را نشان می‌دهد. صفحه هر مسابقه هر ۵ ثانیه اطلاعات را دریافت می‌کند؛ OVR هر ۲ ثانیه. رنکینگ لیگ از دفتر امتیاز همان لیگ و رنکینگ تورنومنت مستقل با امتیاز پیش‌فرض محاسبه می‌شود. این رنکینگ، رتبه‌بندی رسمی فدراسیون نیست. API عمومی فقط فیلدهای لازم برای نمایش را برمی‌گرداند.

برآورد اولیه برای ۵۰۰ بیننده (بدون ویدئو): ۴ vCPU ترجیحاً اختصاصی، ۸ GB RAM، دیسک ۸۰–۱۰۰ GB NVMe و پورت ۱ Gbps. این برآورد تضمین ظرفیت نیست؛ تست بار باید روی سرور مقصد اجرا شود. ۵۰۰ بیننده با فاصله ۵ ثانیه حدود ۱۰۰ درخواست در ثانیه تولید می‌کنند؛ برای OVR با فاصله ۲ ثانیه حدود ۲۵۰ درخواست. پاسخ ۵۰ KB در ۱۰۰ درخواست بر ثانیه تقریباً ۴۰ Mbps و ۱۸ GB ترافیک در ساعت ایجاد می‌کند؛ اندازه واقعی و gzip تعیین‌کننده‌اند.

در بخش `http` تنظیمات Nginx یک کش کوتاه تعریف کنید:

```nginx
proxy_cache_path /var/cache/nginx/tkd-public levels=1:2 keys_zone=tkd_public:10m max_size=256m inactive=1m use_temp_path=off;
```

در server مربوط به سایت، قبل از location عمومی `/api/`، این location را اضافه کنید. مسیر cache باید برای کاربر Nginx قابل نوشتن باشد. این کش فقط برای API عمومی است:

```nginx
location ~ ^/api/v1/public/(competitions(?:/[0-9a-fA-F-]+)?|ovr)$ {
    proxy_pass http://127.0.0.1:8080;
    proxy_cache tkd_public;
    proxy_cache_key "$scheme$host$request_uri";
    proxy_cache_valid 200 2s;
    proxy_cache_lock on;
    proxy_cache_lock_timeout 5s;
    proxy_cache_use_stale off;
    add_header X-Public-Cache $upstream_cache_status always;
}
location ~ ^/api/v1/public/competitions/[0-9a-fA-F-]+/live$ {
    proxy_pass http://127.0.0.1:8080;
    proxy_cache tkd_public;
    proxy_cache_key "$scheme$host$request_uri";
    proxy_cache_valid 200 1s;
    proxy_cache_lock on;
    proxy_cache_use_stale off;
}
location / {
    try_files $uri $uri/ /index.html;
}
```

در صورت وجود `location ^~ /api/`، ابتدا ترتیب locationها را اصلاح کنید تا location کش عمومی واقعاً انتخاب شود. فشرده‌سازی `application/json` را فعال کنید. کش ۲ ثانیه‌ای همراه با تازه‌سازی اطلاعات کلی، تأخیر قابل انتظار حدود ۲ تا ۷ ثانیه دارد. امتیاز داخل راند از مسیر جداگانهٔ `/live` هر ۱ ثانیه دریافت می‌شود؛ با کش ۱ ثانیه‌ای و شبکه، تأخیر معمول حدود ۱ تا ۲ ثانیه است و تایمر مسابقه ندارد. ۵۰۰ بیننده حدود ۵۰۰ درخواست زنده در ثانیه به لایه کش اضافه می‌کنند؛ آزمون بار قبلی باید این مسیر را نیز پوشش دهد. قبل از reload، `nginx -t` را اجرا کنید.

سناریوی تست بار در `scripts/public-results-load.js` است؛ با k6 و شناسه یک مسابقه در محیط staging اجرا شود:

```bash
k6 run -e BASE_URL=https://staging.example.com -e TOURNAMENT_ID=UUID scripts/public-results-load.js
k6 run -e BASE_URL=https://staging.example.com -e TOURNAMENT_ID=UUID -e VIEW=ovr -e INTERVAL=2 scripts/public-results-load.js
```

معیار اولیه: کمتر از ۱٪ خطا و زمان پاسخ صدک ۹۵ زیر ۵۰۰ میلی‌ثانیه. هم‌زمان با تست، ثبت نتیجه توسط اپراتور، مصرف CPU/RAM، اتصال‌های PostgreSQL و نرخ cache HIT را بررسی کنید. این فایل به‌تنهایی به معنی اجرای موفق تست بار نیست.

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
