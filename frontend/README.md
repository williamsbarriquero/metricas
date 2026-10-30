# Frontend do projeto spa

React, Vite, Tailwind e shadcn/ui. Quem serve a API é o Go do projeto `spa/`, não o Python.

O jeito de subir tudo está no [README do spa](../README.md). Daqui, com o Go já no ar em `127.0.0.1:8090`:

```bash
npm install
npm run dev      # http://localhost:5173
npm run build
```

Componentes novos: `npx shadcn@latest add <componente>`.
