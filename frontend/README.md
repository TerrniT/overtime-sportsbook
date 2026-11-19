# Овертайм — frontend

Vue 3 + TypeScript + Vite + Tailwind CSS v4 + shadcn-vue. The dev server proxies
`/api` to `http://127.0.0.1:8080`; set `VITE_API_PROXY` to override the target.

```bash
npm ci
npm run dev        # http://127.0.0.1:5173
npm run typecheck
npm run test
npm run build
```

The Docker image builds with `node:24-alpine` and serves the SPA from nginx,
proxying `/api` (including SSE) to the compose service `api:8080`.
