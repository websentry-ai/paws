# Billing, plans, jobs, catalog

The Billing page and Pro limits, the scheduled jobs, the public catalog and refused requests. Stripe
checkout itself needs real Stripe test keys, so here billing is driven with Stripe not configured and the
Pro plan is set in the DB.

## Sub-features

- `billing-off` `/portal/settings/billing` shows Free and `Try Pro · $10 per user / month`; `POST /billing/checkout` → 503; an unsigned webhook is refused.
- `pro` plan `pro` → `/me` retention 90/30 and the Billing page shows Pro.
- `purge` `manage.py purge_paws_events` drops records past retention.
- `weekly-email-job` `manage.py send_weekly_email` with no Resend key logs `no_email_provider`; `/email/weekly` renders.
- `catalog` `/catalog` lists skills; `GET /catalog` works without auth.
- `auth` no token → 401; a bad device token → 401.

## How to get to it (user POV)

- Settings → `Billing`; the public `/catalog`; the jobs run from cron in production.

## Driving it with portal.sh

Preconditions:

- `onboard` passed (the account exists).

- **Billing.** Run `portal.sh run billing-jobs`. The page shows `Free`; checkout 503; webhook 503/400.
- **Pro.** `update paws_portal_pawsaccount set plan='pro', subscription_status='active'`; `GET /me` 90/30; Billing shows `Pro`; set back to free.
- **Purge.** Insert a 200-day-old event, run the job, the row is gone.
- **Weekly email.** Run the job; output `no_email_provider`; `/email/weekly` has content.
- **Catalog and auth.** `/catalog` lists `humanizer` and `secret-scan`; `GET /catalog` without auth has 40+ skills; `/skills` without a token and `/config` with `pdt_nope` are 401. Screenshots `40-billing` … `44-catalog`.

## Gotchas

- Real checkout, the customer portal and signed webhooks need `PAWS_STRIPE_*` test keys and `stripe listen`; until then they are covered only by paws-backend's mocked tests.
- The weekly email job sends real mail when `RESEND_API_KEY` is set; leave it unset here.
