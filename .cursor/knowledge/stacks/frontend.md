# Frontend (React + Vite) — SSOT deste repositório

Conteúdo canônico da SPA de Métricas dos Coordenadores. Referenciado por `frontend-mastery` e `frontend-standards.mdc`.

O cálculo de T1, T2 e T3 fica no Go (`internal/metrics`). O browser só formata e exibe o que `/api` devolve.

## Perfil

- SPA cliente, compilada pelo Vite e embutida no binário `cmd/spa`. Sem SSR e sem runtime Node na imagem.
- Componentes funcionais, declarativos (SOLID, KISS, DRY).
- TypeScript com `strict: true` em `frontend/tsconfig.app.json` e `frontend/tsconfig.node.json`.
- Código legível antes de micro-otimização; early return e guard clauses.

## Stack deste repositório

| Papel | Ferramenta |
|-------|------------|
| UI | React 19 |
| Build | Vite 8 (`tsc -b && vite build`) |
| Linguagem | TypeScript |
| Estilo | Tailwind CSS 4, tokens em `src/index.css` |
| Componentes | shadcn/ui estilo `base-nova` (Base UI, não Radix) |
| Ícones | lucide-react |
| Roteamento | React Router 7 (`src/App.tsx`) |
| Estado de servidor | TanStack Query |
| Tabela | TanStack Table |
| Gráficos | Recharts |
| Formulários | React Hook Form + Zod (equipe e configurações) |
| Tema | `next-themes` (`class` no elemento raiz) |
| Lint | oxlint type-aware (`frontend/.oxlintrc.json`, `oxlint-tsgolint`) |
| Formato | oxfmt (`frontend/.oxfmtrc.json`: aspas simples, sem ponto e vírgula) |
| Testes | Vitest + Testing Library (`npm test`) |

Alias `@` → `frontend/src`. No dev, o Vite (`:5173`) faz proxy de `/api`, `/csv`, `/google` e `/static` para `127.0.0.1:8090`.

## Fora deste repositório

Não introduzir Next.js, Server Components, Server Actions, `useActionState`, `error.tsx`, Zustand, Radix como dependência nova, ESLint, Prettier, Framer Motion nem um segundo processo Node no pod. Estado global de cliente não tem store: dado de servidor vai para o Query Client; o resto fica em `useState` ou na URL. oxlint e oxfmt rodam só na máquina de desenvolvimento; a imagem final não os inclui.

## Arquitetura

```text
frontend/src/
  main.tsx          QueryClient, ThemeProvider, BrowserRouter
  App.tsx           rotas
  pages/            uma página por rota (export default)
  components/       UI compartilhada e blocos do dashboard
  components/ui/    shadcn (não tratar como pacote opaco; não reformatar em massa)
  lib/api.ts        cliente fetch de /api e tipos do contrato
  lib/format.ts     números, horas, datas e CSV no padrão pt-BR
  hooks/
```

Rotas: `/`, `/metricas`, `/mapa-slide`, `/dados/:aba`, `/equipe`, `/configuracoes`, `/sincronizacao`, `/glossario`, `/print`.

Páginas usam `export default`. Peças reutilizáveis usam named export (`Kpis`). Arquivos do app ficam em minúsculas (`dashboard.tsx`, `print-cartao.tsx`, `use-mobile.ts`). Não renomear o que já existe só para kebab-case.

oxfmt unifica o estilo: aspas simples, sem ponto e vírgula, inclusive em `components/ui`.

## TypeScript

- `strict: true` nos dois tsconfigs do front.
- Preferir `interface` para objetos e props extensíveis.
- Evitar `enum`; usar união de literais ou `as const`.
- Evitar `any`; usar `unknown` e estreitar.
- Props sempre tipadas. O contrato HTTP mora em `lib/api.ts` — não duplicar o formato de T1/T2/T3 na página.

## React

- Componentes funcionais. Lógica repetida vira hook `use*`.
- TanStack Query com query key hierárquica (`['metricas', de, ate]`). `retry: 1`, `staleTime` de 30s, sem refetch no foco (ver `main.tsx`).
- Efeito com cleanup. `useMemo` / `useCallback` / `React.memo` só com ganho medido.
- Key estável em lista. Índice só quando a lista é estática e sem reordenação.
- Mensagem ao usuário em português; detalhe técnico fica no Go (`slog`).

## Estado

| Tipo | Onde |
|------|------|
| Servidor / cache | TanStack Query |
| Formulário | React Hook Form |
| Filtro local de página | `useState` |
| URL | React Router (`/dados/:aba` e search params quando o filtro precisar ser linkável) |
| Tema | `next-themes` |

## Formulários

- Zod na equipe; mensagens claras.
- O servidor continua sendo quem valida config, equipe e período. O schema do cliente não substitui essa checagem.

## Testes

- Utilitários: Vitest, em `src/**/*.test.ts`.
- Componente: Testing Library, comportamento visível (`src/**/*.test.tsx`).
- Sem E2E neste pacote. Cypress e Playwright não estão instalados.

## Segurança

- Segredo de Jira e Google não entra no bundle. O cliente só fala com `/api` na mesma origem.
- Não montar HTML com dado da API via `dangerouslySetInnerHTML`.

## Verificação

Em `frontend/`, conforme `stack-baseline.mdc`:

- `npm run lint`
- `npm run fmt:check`
- `npx tsc -b`
- `npm test`
