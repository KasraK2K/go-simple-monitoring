# Horizon UI Next.js Dashboard

This package contains a Horizon UI + Chakra UI dashboard for the Go monitoring API.

## Getting started

```bash
cd dashboard
npm install
npm run dev
```

The app expects an API server that exposes the `/monitoring` endpoint (POST) and `/api/v1/server-config` endpoint (GET). Configure the base URL via `NEXT_PUBLIC_MONITORING_BASE_URL` in a `.env.local` file when the API is served from another origin.

The dashboard keeps feature parity with the legacy `web/dashboard.html` while adopting Horizon UI patterns:

- Real-time metric cards and charts
- Server switching and historical range filters
- Heartbeat health view and alerting with threshold tracking

## Scripts

- `npm run dev` – start the Next.js dev server
- `npm run build` – create a production build
- `npm run start` – run the production build
- `npm run lint` – run ESLint with the Next.js preset
