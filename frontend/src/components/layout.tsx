import { useQuery } from '@tanstack/react-query'
import {
  BarChart3,
  BookOpen,
  Clock,
  Code,
  Database,
  Moon,
  Presentation,
  RefreshCw,
  Settings,
  Sun,
  Table,
  Users,
} from 'lucide-react'
import { useTheme } from 'next-themes'
import { Link, Outlet, useLocation } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { api } from '@/lib/api'
import { dataHora } from '@/lib/format'

const NAVEGACAO = [
  { to: '/', titulo: 'Dashboard', icone: BarChart3 },
  { to: '/metricas', titulo: 'Métricas', icone: Table },
  { to: '/mapa-slide', titulo: 'MapaSlide', icone: Presentation },
  { to: '/dados', titulo: 'Dados', icone: Database },
  { to: '/equipe', titulo: 'Equipe', icone: Users },
  { to: '/horas-equipe', titulo: 'Horas da Equipe', icone: Clock },
  { to: '/configuracoes', titulo: 'Configurações', icone: Settings },
  { to: '/sincronizacao', titulo: 'Sincronização', icone: RefreshCw },
  { to: '/glossario', titulo: 'Glossário', icone: BookOpen },
  { to: '/jql', titulo: 'Glossário de JQL', icone: Code },
]

function AlternarTema() {
  const { resolvedTheme, setTheme } = useTheme()
  const escuro = resolvedTheme === 'dark'
  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label={escuro ? 'Usar tema claro' : 'Usar tema escuro'}
      onClick={() => setTheme(escuro ? 'light' : 'dark')}
    >
      {escuro ? <Sun /> : <Moon />}
    </Button>
  )
}

function StatusSync() {
  const { data } = useQuery({
    queryKey: ['sync-status'],
    queryFn: api.syncStatus,
    refetchInterval: (q) => (q.state.data?.em_andamento ? 2000 : 30_000),
  })
  if (!data) return null
  if (data.em_andamento)
    return (
      <Badge variant="secondary" className="gap-1">
        <RefreshCw className="size-3 animate-spin" /> Coletando… {data.ultima?.progresso ?? 0}%
      </Badge>
    )
  const ok = [...data.historico].find((r) => r.status === 'ok')
  return (
    <span className="hidden text-xs text-muted-foreground sm:inline">
      {ok?.fim ? `Dados de ${dataHora(ok.fim)}` : 'Sem dados coletados'}
    </span>
  )
}

export function Layout() {
  const { pathname } = useLocation()
  const atual = NAVEGACAO.find((n) => (n.to === '/' ? pathname === '/' : pathname.startsWith(n.to)))

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader className="px-4 py-3">
          <div className="flex items-center gap-2">
            <div className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <BarChart3 className="size-4" />
            </div>
            <div className="leading-tight">
              <p className="text-sm font-semibold">Métricas RPE</p>
              <p className="text-xs text-muted-foreground">Coordenadores</p>
            </div>
          </div>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Menu</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {NAVEGACAO.map((item) => (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      isActive={atual?.to === item.to}
                      render={<Link to={item.to} />}
                      tooltip={item.titulo}
                    >
                      <item.icone />
                      <span>{item.titulo}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter className="px-4 py-3 text-xs text-muted-foreground">
          T1 Histórias × Test Bugs · T2 Horas · T3 Vazão
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="sticky top-0 z-10 flex h-14 items-center gap-2 border-b bg-background/95 px-4 backdrop-blur">
          <SidebarTrigger />
          <Separator orientation="vertical" className="mx-1 h-5" />
          <h1 className="text-base font-semibold">{atual?.titulo ?? 'Métricas'}</h1>
          <div className="ml-auto flex items-center gap-2">
            <StatusSync />
            <AlternarTema />
          </div>
        </header>
        <main className="mx-auto w-full max-w-[92rem] flex-1 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
