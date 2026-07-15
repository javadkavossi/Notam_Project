#!/usr/bin/env bash
#
# ساخت خودکار لیبل‌ها، milestoneها و issueهای فاز ۱ روی گیت‌هاب از روی TASKS.md
#
# پیش‌نیاز:
#   1) gh نصب باشد (نصب است: gh 2.94.0)
#   2) یک‌بار لاگین کنید:  gh auth login
#      یا:  export GH_TOKEN=ghp_xxx   (توکن با دسترسی repo)
#
# اجرا:
#   bash docs/phase1/create_github_issues.sh
#
# ویژگی‌ها:
#   - idempotent نسبی: لیبل/milestone تکراری را رد می‌کند؛ issue تکراری بر اساس عنوان ساخته نمی‌شود.
#   - DRY_RUN=1 برای پیش‌نمایش بدون ساخت:  DRY_RUN=1 bash docs/phase1/create_github_issues.sh
#
set -euo pipefail

REPO="${REPO:-javadkavossi/Notam_Project}"
DRY_RUN="${DRY_RUN:-0}"

say() { printf '\033[36m%s\033[0m\n' "$*"; }
warn() { printf '\033[33m%s\033[0m\n' "$*"; }
run() {
  if [ "$DRY_RUN" = "1" ]; then
    printf '  [dry-run] %s\n' "$*"
  else
    eval "$@"
  fi
}

# ---- بررسی احراز هویت ----
if ! gh auth status >/dev/null 2>&1 && [ -z "${GH_TOKEN:-}" ]; then
  warn "ابتدا وارد شوید:  gh auth login    (یا  export GH_TOKEN=...)"
  exit 1
fi
say "Repo: $REPO   (DRY_RUN=$DRY_RUN)"

# =====================================================================
# ۱) لیبل‌ها
# =====================================================================
say "→ ساخت لیبل‌ها"
create_label() { # name color desc
  if gh label list --repo "$REPO" --limit 200 2>/dev/null | grep -qiE "^$1[[:space:]]"; then
    printf '  = label «%s» موجود است\n' "$1"
  else
    run "gh label create '$1' --repo '$REPO' --color '$2' --description '$3' || true"
  fi
}
# اولویت
create_label "P0"           "b60205" "پایه و مسدودکننده"
create_label "P1"           "d93f0b" "ضروری فاز ۱"
create_label "P2"           "fbca04" "مهم ولی قابل‌تعویق"
# نوع
create_label "epic"         "5319e7" "Epic سطح بالا"
create_label "phase-1"      "0e8a16" "فاز ۱"
# اندازه
create_label "size/S"       "c2e0c6" "≤۱ روز"
create_label "size/M"       "bfdadc" "چند روز"
create_label "size/L"       "bfd4f2" "هفته‌ای"
# دامنه‌ها (Epicها)
create_label "E0-foundation"    "1d76db" "پایه و امن‌سازی"
create_label "E1-ingest"        "1d76db" "دریافت تضمین‌شده"
create_label "E2-pipeline"      "1d76db" "پردازش"
create_label "E3-analysis"      "1d76db" "تحلیل Q-code/امتیاز"
create_label "E4-reliability"   "1d76db" "اعتمادپذیری"
create_label "E5-briefing"      "1d76db" "موتور بریفینگ"
create_label "E6-realtime"      "1d76db" "بلادرنگ/اعلان"
create_label "E7-reference"     "1d76db" "دادهٔ مرجع/PostGIS"
create_label "E8-frontend"      "1d76db" "فرانت‌اند"
create_label "E9-observability" "1d76db" "مانیتورینگ"

# =====================================================================
# ۲) Milestoneها  (از طریق API چون gh دستور مستقیم ندارد)
# =====================================================================
say "→ ساخت Milestoneها"
create_milestone() { # title desc
  local existing
  existing=$(gh api "repos/$REPO/milestones?state=all&per_page=100" --jq ".[] | select(.title==\"$1\") | .number" 2>/dev/null || true)
  if [ -n "$existing" ]; then
    printf '  = milestone «%s» موجود است (#%s)\n' "$1" "$existing"
  else
    run "gh api 'repos/$REPO/milestones' -f title='$1' -f description='$2' >/dev/null && echo '  + milestone: $1'"
  fi
}
create_milestone "M1 — پایهٔ مطمئن"   "E0,E1,E2 — ورود تضمین‌شدهٔ NOTAM بدون data-loss"
create_milestone "M2 — فهمِ دقیق"     "E3,E7 — دیکد/دسته‌بندی/امتیاز + دادهٔ مرجع و PostGIS"
create_milestone "M3 — بریفینگ پرواز" "E5,E8(پایه) — تعریف پرواز و بریفینگ مرتب‌شده"
create_milestone "M4 — اعتماد و زنده"  "E4,E6,E9 — اعلان بلادرنگ، gap/reconcile/consensus، مانیتورینگ"
create_milestone "M5 — پرداخت UX"      "E8(کامل) — نقشه، نوار اعتماد، اخطار بحرانی، سرعت و جذابیت"

milestone_number() { gh api "repos/$REPO/milestones?state=all&per_page=100" --jq ".[] | select(.title==\"$1\") | .number" 2>/dev/null || true; }

# =====================================================================
# ۳) Issueها
# =====================================================================
say "→ ساخت Issueها"

# فهرست عنوان‌های موجود برای جلوگیری از تکرار
EXISTING_TITLES="$(gh issue list --repo "$REPO" --state all --limit 500 --json title --jq '.[].title' 2>/dev/null || true)"

new_issue() { # title  labels(comma)  milestone-title  body
  local title="$1" labels="$2" ms="$3" body="$4"
  if printf '%s\n' "$EXISTING_TITLES" | grep -Fxq "$title"; then
    printf '  = issue موجود است: %s\n' "$title"
    return
  fi
  local ms_flag=""
  if [ -n "$ms" ]; then ms_flag="--milestone \"$ms\""; fi
  # gh issue create با milestone بر اساس عنوان کار می‌کند
  run "gh issue create --repo '$REPO' --title \"$title\" --label '$labels' $ms_flag --body \"$body\" >/dev/null && echo '  + $title'"
}

# ---------- E0 · پایه و امن‌سازی (Milestone M1) ----------
new_issue "[E0-1] خارج‌کردن credentialها از کد و compose به env/secret" "phase-1,epic,P0,size/S,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · پایه و امن‌سازی\n**اولویت:** P0 · **اندازه:** S\n\nخارج‌کردن رمزهای Solace/DB از main.go و docker-compose به env/secret فایل on-prem.\n\n**معیار پذیرش:** هیچ رمزی در گیت نیست؛ اجرا فقط با env.\n\nمرجع: docs/phase1/TASKS.md#E0"
new_issue "[E0-2] جایگزینی auth ثابت با JWT کوتاه‌عمر + refresh" "phase-1,P0,size/M,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** M\n\nحذف توکن ثابت جعل‌پذیر؛ JWT با کلید امضای واقعی، انقضا و refresh.\n\n**معیار پذیرش:** توکن جعل‌ناپذیر؛ انقضا و refresh کار می‌کند."
new_issue "[E0-3] جدول users واقعی + هش رمز + نقش‌ها" "phase-1,P0,size/M,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** M\n\nجدول users + هش (bcrypt/argon2) + نقش‌ها (viewer/operator/admin).\n\n**معیار پذیرش:** ورود با کاربر دیتابیسی؛ نقش در claims."
new_issue "[E0-4] بازچینش پکیج‌ها به ساختار ماژولار هدف" "phase-1,P0,size/M,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** M\n\nبازچینش به ingest/pipeline/reference/... مطابق ARCHITECTURE.md.\n\n**معیار پذیرش:** build سبز؛ مرزهای پکیج مطابق مستند."
new_issue "[E0-5] راه‌اندازی کلاینت Redis Streams" "phase-1,P0,size/M,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** M\n\nکلاینت produce/consume/ack/claim در data/stream.\n\n**معیار پذیرش:** تست دود XADD/XREADGROUP/XACK/XAUTOCLAIM."
new_issue "[E0-6] افزودن PostGIS به postgres" "phase-1,P0,size/S,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** S\n\nimage + extension در migration.\n\n**معیار پذیرش:** CREATE EXTENSION postgis موفق."
new_issue "[E0-7] چارچوب تست (unit + integration با testcontainers)" "phase-1,P0,size/M,E0-foundation" "M1 — پایهٔ مطمئن" \
"**Epic:** E0 · **اولویت:** P0 · **اندازه:** M\n\n**معیار پذیرش:** CI تست‌ها را اجرا می‌کند."

# ---------- E1 · Ingest تضمین‌شده (M1) ----------
new_issue "[E1-1] interface SourceAdapter + RawNotamMessage + Normalizer" "phase-1,P0,size/S,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P0 · **اندازه:** S\n\n**معیار پذیرش:** آداپتورها فقط از این قرارداد استفاده می‌کنند."
new_issue "[E1-2] بازنویسی آداپتور Solace با client-ack (حذف auto-ack)" "phase-1,P0,size/M,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P0 · **اندازه:** M\n\nack پس از XADD موفق؛ رفع ریسک data-loss فعلی.\n\n**معیار پذیرش:** crash قبل از ذخیره → پیام گم نمی‌شود (تست)."
new_issue "[E1-3] reconnect با backoff + ثبت بازهٔ قطعی" "phase-1,P1,size/M,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** قطع/وصل بدون ازدست‌رفتن؛ بازهٔ قطعی ثبت."
new_issue "[E1-4] ثبت provenance/sighting هنگام ورود" "phase-1,P0,size/S,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P0 · **اندازه:** S\n\nsource, seen_at, raw_hash.\n\n**معیار پذیرش:** هر پیام منبعش ثبت می‌شود."
new_issue "[E1-5] placeholder آداپتور FAA REST (backfill/reconcile)" "phase-1,P1,size/M,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** pull بازهٔ زمانی/فرودگاه کار می‌کند."
new_issue "[E1-6] طراحی placeholder آداپتور AFTN" "phase-1,P2,size/S,E1-ingest" "M1 — پایهٔ مطمئن" \
"**Epic:** E1 · **اولویت:** P2 · **اندازه:** S\n\n**معیار پذیرش:** interface آماده؛ مستند نحوهٔ افزودن."

# ---------- E2 · Pipeline (M1) ----------
new_issue "[E2-1] جداکردن parser XML/ICAO از آداپتور Solace" "phase-1,P0,size/M,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P0 · **اندازه:** M\n\n**معیار پذیرش:** تست با نمونه‌های واقعی FAA؛ استقلال از Solace."
new_issue "[E2-2] پردازندهٔ استریم (consumer group) + XACK پس از commit" "phase-1,P0,size/M,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P0 · **اندازه:** M\n\n**معیار پذیرش:** at-least-once؛ idempotent."
new_issue "[E2-3] canonical_key + UPSERT idempotent" "phase-1,P0,size/M,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P0 · **اندازه:** M\n\nجایگزین کلید message_id.\n\n**معیار پذیرش:** پیام تکراری/چندمنبعی رکورد یکتا می‌سازد."
new_issue "[E2-4] ساخت area (PostGIS) از مختصات+شعاع خط Q" "phase-1,P1,size/M,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** نقطه/دایره در DB؛ کوئری فضایی کار می‌کند."
new_issue "[E2-5] مدیریت NOTAMR/NOTAMC روی canonical_key" "phase-1,P1,size/M,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** کنسل/جایگزین درست اعمال می‌شود."
new_issue "[E2-6] DLQ برای خطای parse/panic + متریک" "phase-1,P1,size/S,E2-pipeline" "M1 — پایهٔ مطمئن" \
"**Epic:** E2 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** پیام خطادار به DLQ می‌رود، گم نمی‌شود."

# ---------- E3 · تحلیل (M2) ----------
new_issue "[E3-1] جدول کامل Q-code + دیکدر" "phase-1,P1,size/L,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** L\n\nsubject/condition به‌صورت داده‌محور.\n\n**معیار پذیرش:** نمونه‌های شناخته‌شده درست دیکد می‌شوند."
new_issue "[E3-2] fallback تحلیل متن E) وقتی Q-code نیست/XX" "phase-1,P1,size/M,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** FICON/RWY CLSD/ILS U/S تشخیص داده می‌شود."
new_issue "[E3-3] نگاشت به category + flightPhase[] + tags[]" "phase-1,P1,size/M,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** دسته‌بندی صحیح روی مجموعهٔ آزمون."
new_issue "[E3-4] موتور امتیاز پایه با جدول وزن نسخه‌دار" "phase-1,P1,size/L,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** L\n\n**معیار پذیرش:** امتیاز قابل‌توضیح؛ خروجی سطح درست."
new_issue "[E3-5] مستند جدول وزن‌ها برای بازبینی کارشناس" "phase-1,P1,size/S,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** تغییر وزن بدون تغییر کد."
new_issue "[E3-6] مجموعهٔ آزمون طلایی (golden set)" "phase-1,P1,size/M,E3-analysis" "M2 — فهمِ دقیق" \
"**Epic:** E3 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** تست رگرسیون دقت روی هر تغییر."

# ---------- E7 · دادهٔ مرجع (M2 / برخی M1) ----------
new_issue "[E7-1] مدل Airport/Runway/Navaid/FIR + مهاجرت PostGIS" "phase-1,P0,size/M,E7-reference" "M2 — فهمِ دقیق" \
"**Epic:** E7 · **اولویت:** P0 · **اندازه:** M\n\n**معیار پذیرش:** جداول + ایندکس GiST."
new_issue "[E7-2] ETL فرودگاه/باند/ناوید (OurAirports/NASR)" "phase-1,P1,size/L,E7-reference" "M2 — فهمِ دقیق" \
"**Epic:** E7 · **اولویت:** P1 · **اندازه:** L\n\n**معیار پذیرش:** داده‌های کامل بارگذاری می‌شوند."
new_issue "[E7-3] ورود مرزهای FIR (GeoJSON) به PostGIS" "phase-1,P1,size/M,E7-reference" "M2 — فهمِ دقیق" \
"**Epic:** E7 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** ST_Intersects کار می‌کند."
new_issue "[E7-4] نسخه‌بندی + changelog + تشخیص تغییر + هشدار" "phase-1,P1,size/M,E7-reference" "M2 — فهمِ دقیق" \
"**Epic:** E7 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** تغییر باند فرودگاه → ثبت + اطلاع."
new_issue "[E7-5] endpoint autocomplete فرودگاه/waypoint (کش‌شده)" "phase-1,P1,size/S,E7-reference" "M2 — فهمِ دقیق" \
"**Epic:** E7 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** جستجوی فوری در فرم پرواز."

# ---------- E5 · موتور بریفینگ (M3) ----------
new_issue "[E5-1] مدل FlightPlan + FlightAirport + API ساخت/ذخیره پرواز" "phase-1,P1,size/M,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** CRUD پرواز کار می‌کند."
new_issue "[E5-2] تطبیق مکانی aerodrome" "phase-1,P1,size/S,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** NOTAMهای فرودگاه‌های پرواز برمی‌گردند."
new_issue "[E5-3] تطبیق enroute با PostGIS + fallback FIR" "phase-1,P1,size/L,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** L\n\n**معیار پذیرش:** NOTAMهای مسیر درست انتخاب می‌شوند."
new_issue "[E5-4] فیلتر زمانی دقیق + بررسی schedule بند D" "phase-1,P1,size/M,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** فقط NOTAMهای فعال در پنجرهٔ پرواز."
new_issue "[E5-5] امتیاز کانتکستی + match_reason" "phase-1,P1,size/M,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** امتیاز وابسته به پرواز؛ دلیل انتخاب ثبت."
new_issue "[E5-6] گروه‌بندی + رتبه‌بندی + خلاصهٔ بحرانی" "phase-1,P1,size/M,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** خروجی مطابق بخش ۷ ANALYSIS."
new_issue "[E5-7] کش بریفینگ در Redis + invalidation" "phase-1,P2,size/M,E5-briefing" "M3 — بریفینگ پرواز" \
"**Epic:** E5 · **اولویت:** P2 · **اندازه:** M\n\n**معیار پذیرش:** بریفینگ گرم < ~۳۰۰ms."

# ---------- E8 · Frontend (M3/M5) ----------
new_issue "[E8-1] افزودن TanStack Query + Zustand + ساختار state" "phase-1,P1,size/M,E8-frontend" "M3 — بریفینگ پرواز" \
"**Epic:** E8 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** server-state مدیریت‌شده."
new_issue "[E8-2] فرم «پرواز جدید» با autocomplete" "phase-1,P1,size/L,E8-frontend" "M3 — بریفینگ پرواز" \
"**Epic:** E8 · **اولویت:** P1 · **اندازه:** L\n\nADEP/ADES/ALTN/روت + زمان.\n\n**معیار پذیرش:** ساخت پرواز روان و سریع."
new_issue "[E8-3] نمای بریفینگ: خلاصهٔ بحرانی + گروه‌ها + سطوح رنگی" "phase-1,P1,size/L,E8-frontend" "M3 — بریفینگ پرواز" \
"**Epic:** E8 · **اولویت:** P1 · **اندازه:** L\n\n**معیار پذیرش:** مطابق بخش ۷ ANALYSIS."
new_issue "[E8-4] اتصال WebSocket + toast/بنر + اخطار بحرانی" "phase-1,P1,size/M,E8-frontend" "M4 — اعتماد و زنده" \
"**Epic:** E8 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** اعلان زنده در UI."
new_issue "[E8-5] نوار اعتماد (وضعیت منابع + تازگی داده + بنر هشدار)" "phase-1,P1,size/M,E8-frontend" "M4 — اعتماد و زنده" \
"**Epic:** E8 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** مطابق بخش ۱۰ RELIABILITY."
new_issue "[E8-6] نقشهٔ روت با MapLibre + نقاط NOTAM" "phase-1,P2,size/L,E8-frontend" "M5 — پرداخت UX" \
"**Epic:** E8 · **اولویت:** P2 · **اندازه:** L\n\n**معیار پذیرش:** روت و NOTAMها روی نقشه."
new_issue "[E8-7] صیقل UX/سرعت (skeleton، انیمیشن، RTL، a11y)" "phase-1,P2,size/M,E8-frontend" "M5 — پرداخت UX" \
"**Epic:** E8 · **اولویت:** P2 · **اندازه:** M\n\n**معیار پذیرش:** تجربهٔ سریع و جذاب."

# ---------- E4 · اعتمادپذیری (M4) ----------
new_issue "[E4-1] جدول notam_series_watermark + تشخیص gap" "phase-1,P1,size/M,E4-reliability" "M4 — اعتماد و زنده" \
"**Epic:** E4 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** شمارهٔ جاافتاده تشخیص و ثبت می‌شود."
new_issue "[E4-2] backfill هدف‌دار gap + ارتقای مشکوک→تأییدشده" "phase-1,P1,size/M,E4-reliability" "M4 — اعتماد و زنده" \
"**Epic:** E4 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** gap پر یا پس از N تلاش هشدار می‌دهد."
new_issue "[E4-3] job reconcile: pull کامل + diff" "phase-1,P1,size/L,E4-reliability" "M4 — اعتماد و زنده" \
"**Epic:** E4 · **اولویت:** P1 · **اندازه:** L\n\nadd/stale/verify.\n\n**معیار پذیرش:** جاافتاده‌ها backfill؛ stale علامت‌گذاری."
new_issue "[E4-4] منطق اجماع چندمنبعی + confidence" "phase-1,P1,size/M,E4-reliability" "M4 — اعتماد و زنده" \
"**Epic:** E4 · **اولویت:** P1 · **اندازه:** M\n\nsingle/corroborated/conflicting.\n\n**معیار پذیرش:** تعارض پرچم می‌شود."
new_issue "[E4-5] backfill بازهٔ قطعی پس از بازگشت منبع" "phase-1,P1,size/M,E4-reliability" "M4 — اعتماد و زنده" \
"**Epic:** E4 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** بازهٔ down پوشش داده می‌شود."

# ---------- E6 · بلادرنگ (M4) ----------
new_issue "[E6-1] WebSocket hub + احراز هویت اتصال" "phase-1,P1,size/M,E6-realtime" "M4 — اعتماد و زنده" \
"**Epic:** E6 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** کلاینت متصل و پیام می‌گیرد."
new_issue "[E6-2] مدل subscription روی پرواز فعال" "phase-1,P1,size/M,E6-realtime" "M4 — اعتماد و زنده" \
"**Epic:** E6 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** فقط NOTAMهای مرتبط push می‌شوند."
new_issue "[E6-3] dispatcher: NOTAM جدید → match → push" "phase-1,P1,size/M,E6-realtime" "M4 — اعتماد و زنده" \
"**Epic:** E6 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** تحویل بلادرنگ (< چند ثانیه)."
new_issue "[E6-4] اخطار بحرانی (CRITICAL) با صوت/نشانه متمایز" "phase-1,P1,size/S,E6-realtime" "M4 — اعتماد و زنده" \
"**Epic:** E6 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** مورد بحرانی اخطار اجباری می‌دهد."
new_issue "[E6-5] حذف polling فعلی و جایگزینی با WS" "phase-1,P1,size/S,E6-realtime" "M4 — اعتماد و زنده" \
"**Epic:** E6 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** فشار سرور کم؛ تأخیر پایین."

# ---------- E9 · Observability (M4) ----------
new_issue "[E9-1] متریک‌های Prometheus" "phase-1,P1,size/M,E9-observability" "M4 — اعتماد و زنده" \
"**Epic:** E9 · **اولویت:** P1 · **اندازه:** M\n\nطبق جدول RELIABILITY §۹.\n\n**معیار پذیرش:** /metrics کامل."
new_issue "[E9-2] /health تجمیعی (DB/Redis/منابع/lag/gap)" "phase-1,P1,size/S,E9-observability" "M4 — اعتماد و زنده" \
"**Epic:** E9 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** ۲۰۰/۵۰۳ با جزئیات."
new_issue "[E9-3] source liveness state machine + هشدار قطعی" "phase-1,P1,size/M,E9-observability" "M4 — اعتماد و زنده" \
"**Epic:** E9 · **اولویت:** P1 · **اندازه:** M\n\n**معیار پذیرش:** قطع منبع سریع تشخیص."
new_issue "[E9-4] لاگ ساختاریافته + correlation id سرتاسری" "phase-1,P1,size/S,E9-observability" "M4 — اعتماد و زنده" \
"**Epic:** E9 · **اولویت:** P1 · **اندازه:** S\n\n**معیار پذیرش:** ردیابی یک NOTAM سرتاسر."
new_issue "[E9-5] داشبورد Grafana پایه (اختیاری)" "phase-1,P2,size/M,E9-observability" "M4 — اعتماد و زنده" \
"**Epic:** E9 · **اولویت:** P2 · **اندازه:** M\n\n**معیار پذیرش:** نمای عملیات."

say "✓ تمام شد."
