// Cliente da API (o Vite faz proxy de /api para o backend em desenvolvimento).

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(`/api${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!resp.ok) {
    let msg = `Erro ${resp.status}`
    try {
      const corpo = await resp.json()
      if (typeof corpo?.detail === 'string') msg = corpo.detail
      else if (typeof corpo?.erro === 'string') msg = corpo.erro
      else if (Array.isArray(corpo?.detail))
        msg = corpo.detail.map((d: { msg: string }) => d.msg).join('; ')
    } catch {
      // resposta sem JSON
    }
    throw new ApiError(resp.status, msg)
  }
  if (resp.status === 204) return undefined as T
  return resp.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Tipos
// ---------------------------------------------------------------------------

export interface LinhaT1 {
  historias: number
  test_bugs: number
  razao: number | null
  fluxo_principal: number
  fluxo_alternativo: number
  fluxo_excecao: number
}

export interface LinhaT2 {
  sustentacao: number
  evolucao: number
  overhead: number
  sem_conta: number
  recesso: number
  total: number
  pct_sustentacao: number | null
  pct_evolucao: number | null
  pct_overhead: number | null
}

export interface LinhaT3 {
  criados: number
  resolvidos: number
  estoque: number
  vazao: number | null
}

export type Mensal<L> = L & { mes: string }

export interface Tabela<L> {
  meses: Mensal<L>[]
  total: L
}

export interface Bloco {
  chave: string
  nome: string
  rotulo: string
  tem_dados: boolean
  bu?: string
  bu_nome?: string
  cor?: string
  paleta?: string[]
  t1: Tabela<LinhaT1>
  t2: Tabela<LinhaT2> & { overhead_detalhe: { conta: string; horas: number }[] }
  t3: Tabela<LinhaT3>
}

export interface Metricas {
  periodo: { inicio: string; fim: string; meses: string[] }
  atualizado_em: string | null
  equipe_ativa: boolean
  avisos: string[]
  espacos: Bloco[]
  todos: Bloco | null
}

export interface MembroEquipe {
  id: number
  nome: string
  account_id: string
  espacos: string
  incluir: boolean
  observacao: string
  coordenador: string
}
export type MembroEquipeIn = Omit<MembroEquipe, 'id'>

export interface ResultadoPadrao {
  adicionados: number
  coordenadores_atualizados: number
}

export interface ConfigItem {
  chave: string
  rotulo: string
  tipo: 'data' | 'simnao' | 'lista' | 'texto'
  padrao: string
  nota: string
  valor: string
}

export interface ConfigResposta {
  itens: ConfigItem[]
  jira: { url: string; email: string; configurado: boolean }
  google: { configurado: boolean; conectado: boolean; conta: string; demo: boolean }
}

export interface SyncRun {
  id: number
  bloco: string
  inicio: string
  fim: string | null
  status: 'executando' | 'ok' | 'erro'
  linhas: number
  progresso: number
  etapa: string
  mensagem: string
  periodo_inicio: string
  periodo_fim: string
}

export interface SyncStatus {
  em_andamento: boolean
  ultima: SyncRun | null
  historico: SyncRun[]
  jira_configurado: boolean
  demo?: boolean
}

export interface Espaco {
  chave: string
  nome: string
  ordem: number
  bu?: string
  bu_nome?: string
  cor?: string
  paleta?: string[]
}

export interface CelulaDados {
  texto: string
  classe?: string
  link?: string
  selo?: string
}

export interface LinhaDados {
  celulas: CelulaDados[]
  apagada: boolean
}

export interface DadosPagina {
  aba: string
  total: number
  pagina: number
  por_pagina: number
  paginas: number
  colunas: { titulo: string; classe?: string }[]
  linhas: LinhaDados[]
}

export interface MarcadorSlide {
  marcador: string
  valor: string
  grupo: string
  grupo_titulo: string
  secao: string
  origem: string
}

export interface MapaSlide {
  periodo: Metricas['periodo']
  aviso: string
  marcadores: MarcadorSlide[] | null
}

export interface SlideGeracao {
  id: number
  inicio: string
  fim: string | null
  periodo_inicio: string
  periodo_fim: string
  status: 'executando' | 'ok' | 'erro'
  url: string
  mensagem: string
  avisos: string
  progresso: number
  etapa: string
}

export interface SlidePainel {
  google_configurado: boolean
  conectado: boolean
  demo: boolean
  pode_gerar: boolean
  conta: string
  modelo_id: string
  modelo_url: string
  motivo: string
  em_andamento: boolean
  atual: { etapa: string; progresso: number; status: string } | null
  historico: SlideGeracao[]
}

export interface Glossario {
  espacos: {
    chave: string
    nome: string
    bu?: string
    bu_nome?: string
    cor?: string
    historias: boolean
    incidentes: boolean
    horas: boolean
  }[]
  tiers: {
    id: string
    titulo: string
    descricao: string
    campos: { cod: string; nome: string; formato: string; regra: string; exemplo?: string }[]
  }[]
  gerais?: { chave: string; descricao: string }[]
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

export function comQuery(path: string, params: Record<string, string | undefined>) {
  const q = new URLSearchParams()
  for (const [chave, valor] of Object.entries(params)) {
    if (valor) q.set(chave, valor)
  }
  const qs = q.toString()
  return qs ? `${path}?${qs}` : path
}

export interface ConsultaJQL {
  id: string
  titulo: string
  jql: string
}

export interface GrupoJQL {
  chave: string
  nome: string
  consultas: ConsultaJQL[]
}

export interface PainelJQL {
  inicio: string
  fim: string
  limite_exclusivo: string
  espacos: GrupoJQL[]
}

export interface SubtotalHoras {
  rotulo: string
  horas: number
}

export interface LancamentoHoras {
  data: string
  espaco: string
  chave: string
  resumo: string
  conta: string
  horas: number
  link?: string
}

export interface PessoaHoras {
  nome: string
  account_id: string
  coordenador: string
  total_horas: number
  meses: SubtotalHoras[]
  contas: SubtotalHoras[]
  lancamentos: LancamentoHoras[]
}

export interface GrupoHoras {
  rotulo: string
  pessoas: PessoaHoras[]
}

export interface HorasEquipe {
  inicio: string
  fim: string
  pessoas: PessoaHoras[]
  grupos: GrupoHoras[]
}

export const api = {
  metricas: (inicio?: string, fim?: string, horasFora = false) => {
    const q = new URLSearchParams()
    if (inicio) q.set('inicio', inicio)
    if (fim) q.set('fim', fim)
    if (horasFora) q.set('horasFora', '1')
    const qs = q.toString()
    return request<Metricas>(`/metricas${qs ? `?${qs}` : ''}`)
  },
  espacos: () => request<Espaco[]>('/espacos'),
  glossario: () => request<Glossario>('/glossario'),
  jql: () => request<PainelJQL>('/jql'),
  horasEquipe: (horasFora = false) =>
    request<HorasEquipe>(`/horas-equipe${horasFora ? '?horasFora=1' : ''}`),

  config: () => request<ConfigResposta>('/config'),
  salvarConfig: (valores: Record<string, string>) =>
    request<ConfigResposta>('/config', { method: 'PUT', body: JSON.stringify({ valores }) }),

  equipe: () => request<MembroEquipe[]>('/equipe'),
  coordenadoresEquipe: () => request<string[]>('/equipe/coordenadores'),
  criarMembro: (m: MembroEquipeIn) =>
    request<MembroEquipe>('/equipe', { method: 'POST', body: JSON.stringify(m) }),
  atualizarMembro: (id: number, m: MembroEquipeIn) =>
    request<MembroEquipe>(`/equipe/${id}`, { method: 'PUT', body: JSON.stringify(m) }),
  removerMembro: (id: number) => request<void>(`/equipe/${id}`, { method: 'DELETE' }),
  completarPadrao: () => request<ResultadoPadrao>('/equipe/padrao', { method: 'POST' }),
  resolverAccountIds: () =>
    request<Record<string, string>>('/equipe/resolver-account-ids', { method: 'POST' }),

  syncStatus: () => request<SyncStatus>('/sync/status'),
  iniciarSync: (bloco: string) =>
    request<SyncRun>('/sync', { method: 'POST', body: JSON.stringify({ bloco }) }),

  mapaSlide: (de?: string, ate?: string) =>
    request<MapaSlide>(comQuery('/mapa-slide', { de, ate })),
  dados: (aba: string, params: Record<string, string | undefined>) =>
    request<DadosPagina>(comQuery(`/dados/${aba}`, params)),
  slide: () => request<SlidePainel>('/slide'),
  gerarSlide: (de?: string, ate?: string) =>
    request<SlidePainel>('/slide', {
      method: 'POST',
      body: JSON.stringify({ de: de ?? '', ate: ate ?? '' }),
    }),
  desconectarGoogle: () => request<ConfigResposta>('/google/desconectar', { method: 'POST' }),
}
