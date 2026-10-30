import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  CheckCircle2,
  Download,
  Loader2,
  Presentation,
  RotateCcw,
  XCircle,
} from 'lucide-react'
import { Fragment, useEffect, useRef } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { type MarcadorSlide, type SlideGeracao, api, comQuery } from '@/lib/api'
import { data as formatarData, dataHora, duracao, numero, paraMes } from '@/lib/format'

const TITULO_SECAO: Record<string, string> = {
  ID: 'Identificação e meses',
  T1: 'T1 - Histórias vs Test Bugs  (H cards, T test bugs, P %, FP/FA/FE fluxos)',
  T2: 'T2 - Horas por Conta de Esforço  (S sustentação, E evolução, O overhead, T total)',
  T3: 'T3 - Vazão de Incidentes  (C criados, R resolvidos, E estoque, V vazão %)',
}

function definirParam(params: URLSearchParams, chave: string, valor: string) {
  const next = new URLSearchParams(params)
  if (valor) next.set(chave, valor)
  else next.delete(chave)
  return next
}

function rotuloGrupo(m: MarcadorSlide) {
  if (m.grupo === 'TODOS') return 'TODOS - soma de todos os Espaços'
  if (m.grupo === 'GERAIS') return m.grupo_titulo || 'GERAIS'
  return m.grupo_titulo.replace(/^ESPAÇO:\s*/, '') || m.grupo
}

function agrupar(ms: MarcadorSlide[]) {
  const out: {
    chave: string
    titulo: string
    secoes: { titulo: string; itens: MarcadorSlide[] }[]
  }[] = []
  for (const m of ms) {
    let g = out.at(-1)
    if (!g || g.chave !== m.grupo) {
      g = { chave: m.grupo, titulo: m.grupo_titulo, secoes: [] }
      out.push(g)
    }
    const titulo = TITULO_SECAO[m.secao] ?? ''
    let sec = g.secoes.at(-1)
    if (!sec || sec.titulo !== titulo) {
      sec = { titulo, itens: [] }
      g.secoes.push(sec)
    }
    sec.itens.push(m)
  }
  return out
}

function StatusBadge({ status }: { status: SlideGeracao['status'] }) {
  if (status === 'ok')
    return (
      <Badge variant="secondary" className="gap-1 text-emerald-600 dark:text-emerald-400">
        <CheckCircle2 className="size-3" /> Concluída
      </Badge>
    )
  if (status === 'erro')
    return (
      <Badge variant="destructive" className="gap-1">
        <XCircle className="size-3" /> Erro
      </Badge>
    )
  return (
    <Badge variant="secondary" className="gap-1">
      <Loader2 className="size-3 animate-spin" /> Em andamento
    </Badge>
  )
}

function PainelSlide({ de, ate }: { de?: string; ate?: string }) {
  const qc = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ['slide'],
    queryFn: api.slide,
    refetchInterval: (q) => (q.state.data?.em_andamento ? 1500 : false),
  })

  const gerar = useMutation({
    mutationFn: () => api.gerarSlide(de, ate),
    onSuccess: () => {
      toast.info(data?.demo ? 'Pré-visualização iniciada.' : 'Geração do Slide iniciada.')
      void qc.invalidateQueries({ queryKey: ['slide'] })
    },
    onError: (e: Error) =>
      toast.error('Não foi possível gerar o Slide', { description: e.message }),
  })

  const desconectar = useMutation({
    mutationFn: api.desconectarGoogle,
    onSuccess: () => {
      toast.success('Acesso Google revogado e conta desconectada.')
      void qc.invalidateQueries({ queryKey: ['slide'] })
      void qc.invalidateQueries({ queryKey: ['config'] })
    },
    onError: (e: Error) => toast.error('Não foi possível desconectar', { description: e.message }),
  })

  const ultima = data?.historico[0]
  const visto = useRef<{ id: number; status: string } | null>(null)
  useEffect(() => {
    if (!ultima) return
    const anterior = visto.current
    visto.current = { id: ultima.id, status: ultima.status }
    if (
      !anterior ||
      anterior.id !== ultima.id ||
      anterior.status !== 'executando' ||
      ultima.status === 'executando'
    )
      return
    if (ultima.status === 'ok')
      toast.success('Slide gerado.', { description: 'A apresentação está pronta para abrir.' })
    else toast.error('A geração do Slide falhou.', { description: ultima.mensagem.slice(0, 200) })
  }, [ultima])

  if (isLoading) return <Skeleton className="h-40" />
  if (error || !data)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {(error as Error | null)?.message ?? 'Erro ao carregar a geração do Slide.'}
        </AlertDescription>
      </Alert>
    )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2">
          Gerar Slide
          {data.demo ? (
            <Badge variant="secondary">Modo demonstração</Badge>
          ) : data.conectado ? (
            <Badge variant="secondary" className="text-emerald-600 dark:text-emerald-400">
              Google: {data.conta || 'conectado'}
            </Badge>
          ) : (
            <Badge variant="destructive">Google não conectado</Badge>
          )}
        </CardTitle>
        <CardDescription>
          Faz uma cópia do Slide modelo (o modelo não é alterado) e preenche os marcadores.{' '}
          <code>{'{{HPAY_T1}}'}</code>, <code>{'{{HPAY_T2}}'}</code> e <code>{'{{HPAY_T3}}'}</code>{' '}
          desenham a métrica inteira dentro da caixa de texto. <code>{'{{HPAY_T1_GRAFICO}}'}</code>{' '}
          desenha só o gráfico.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm">
          <span className="text-muted-foreground">Slide modelo: </span>
          {data.modelo_url ? (
            <a
              href={data.modelo_url}
              target="_blank"
              rel="noopener"
              className="font-medium underline"
            >
              {data.modelo_id}
            </a>
          ) : (
            <span className="text-muted-foreground">não informado (Configurações)</span>
          )}
        </p>
        {data.motivo && (
          <Alert>
            <AlertTriangle />
            <AlertDescription>
              {data.motivo}{' '}
              {data.motivo.includes('Configurações') && (
                <Link to="/configuracoes" className="font-medium underline">
                  Ir para Configurações
                </Link>
              )}
            </AlertDescription>
          </Alert>
        )}
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={!data.pode_gerar || data.em_andamento || gerar.isPending}
            onClick={() => gerar.mutate()}
          >
            {data.em_andamento || gerar.isPending ? (
              <Loader2 className="animate-spin" />
            ) : (
              <Presentation />
            )}
            {data.demo ? 'Pré-visualizar' : 'Gerar Slide'}
          </Button>
          {!data.demo &&
            data.google_configurado &&
            (data.conectado ? (
              <>
                <a href="/google/conectar" className={buttonVariants({ variant: 'outline' })}>
                  Reconectar Google
                </a>
                <Button
                  variant="ghost"
                  disabled={desconectar.isPending}
                  onClick={() => desconectar.mutate()}
                >
                  Revogar acesso
                </Button>
              </>
            ) : (
              <a href="/google/conectar" className={buttonVariants({ variant: 'outline' })}>
                Conectar conta Google
              </a>
            ))}
        </div>

        {data.em_andamento && (
          <div className="space-y-2 rounded-lg border p-4">
            <div className="flex items-center justify-between text-sm">
              <span className="font-medium">{data.atual?.etapa || 'Gerando…'}</span>
              <span className="tabular-nums text-muted-foreground">
                {data.atual?.progresso ?? 0}%
              </span>
            </div>
            <Progress value={data.atual?.progresso ?? 0} />
          </div>
        )}

        {data.historico.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">Últimas gerações</h3>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Início</TableHead>
                  <TableHead>Período</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Resultado</TableHead>
                  <TableHead className="text-right">Duração</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.historico.map((h) => {
                  const texto = [h.mensagem, h.avisos].filter(Boolean).join('\n')
                  return (
                    <Fragment key={h.id}>
                      <TableRow>
                        <TableCell>{dataHora(h.inicio)}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatarData(h.periodo_inicio)} a {formatarData(h.periodo_fim)}
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={h.status} />
                        </TableCell>
                        <TableCell>
                          {h.url ? (
                            <a
                              href={h.url}
                              target="_blank"
                              rel="noopener"
                              className="font-medium underline"
                            >
                              Abrir apresentação
                            </a>
                          ) : (
                            <span className="text-muted-foreground">–</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {duracao(h.inicio, h.fim)}
                        </TableCell>
                      </TableRow>
                      {texto && (
                        <TableRow>
                          <TableCell colSpan={5}>
                            <details open={h.status === 'erro'}>
                              <summary className="cursor-pointer text-sm">
                                {h.status === 'erro' ? 'Ver erro' : 'Ver resumo'}
                              </summary>
                              <pre
                                className={`mt-2 whitespace-pre-wrap text-xs ${h.status === 'erro' ? 'text-destructive' : 'text-muted-foreground'}`}
                              >
                                {texto}
                              </pre>
                            </details>
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function MapaSlidePage() {
  const [params, setParams] = useSearchParams()
  const de = params.get('de') || undefined
  const ate = params.get('ate') || undefined
  const grupo = params.get('grupo') || ''
  const busca = params.get('q') || ''

  const consulta = useQuery({
    queryKey: ['mapa-slide', de, ate],
    queryFn: () => api.mapaSlide(de, ate),
    placeholderData: (anterior) => anterior,
  })

  if (consulta.isLoading) return <Skeleton className="h-96" />
  if (consulta.isError || !consulta.data) {
    return (
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertTitle>Não foi possível montar o MapaSlide</AlertTitle>
        <AlertDescription>
          {consulta.error instanceof Error ? consulta.error.message : 'Erro desconhecido.'}
        </AlertDescription>
      </Alert>
    )
  }

  const mapa = consulta.data
  const marcadores = mapa.marcadores ?? []
  const opcoes = [{ value: 'todos', label: 'Todos os grupos' }]
  const vistos = new Set<string>()
  for (const m of marcadores) {
    if (vistos.has(m.grupo)) continue
    vistos.add(m.grupo)
    opcoes.push({ value: m.grupo, label: rotuloGrupo(m) })
  }
  const q = busca.trim().toLowerCase()
  const filtrados = marcadores.filter((m) => {
    if (grupo && m.grupo !== grupo) return false
    if (!q) return true
    return `${m.marcador} ${m.valor} ${m.origem}`.toLowerCase().includes(q)
  })
  const semValor = filtrados.filter((m) => !m.valor).length
  const grupos = agrupar(filtrados)
  const mesInicio = de ?? paraMes(mapa.periodo.inicio)
  const mesFim = ate ?? paraMes(mapa.periodo.fim)
  const csv = comQuery('/csv/mapa-slide', {
    de,
    ate,
    grupo: grupo || undefined,
    q: busca || undefined,
  })

  async function copiar(texto: string) {
    try {
      await navigator.clipboard.writeText(texto)
      toast.success('Marcador copiado.', { description: texto })
    } catch {
      toast.error('Não foi possível copiar o marcador.')
    }
  }

  return (
    <div className="space-y-6">
      <PainelSlide de={de} ate={ate} />

      {mapa.aviso && (
        <Alert>
          <AlertTriangle />
          <AlertDescription>{mapa.aviso}</AlertDescription>
        </Alert>
      )}

      <div className="flex flex-wrap items-end gap-4">
        <div className="space-y-1.5">
          <Label>Grupo</Label>
          <Select
            value={grupo || 'todos'}
            onValueChange={(v) =>
              setParams(definirParam(params, 'grupo', !v || v === 'todos' ? '' : v))
            }
            items={opcoes}
          >
            <SelectTrigger className="w-80">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="max-h-96">
              {opcoes.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="q">Buscar</Label>
          <Input
            id="q"
            type="search"
            className="w-56"
            placeholder="HPAY_T1, 78,6%..."
            value={busca}
            onChange={(e) => setParams(definirParam(params, 'q', e.target.value))}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="de">De</Label>
          <Input
            id="de"
            type="month"
            className="w-40"
            value={mesInicio}
            onChange={(e) =>
              e.target.value && setParams(definirParam(params, 'de', e.target.value))
            }
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="ate">Até</Label>
          <Input
            id="ate"
            type="month"
            className="w-40"
            value={mesFim}
            onChange={(e) =>
              e.target.value && setParams(definirParam(params, 'ate', e.target.value))
            }
          />
        </div>
        {(de || ate) && (
          <Button
            variant="ghost"
            onClick={() => setParams(definirParam(definirParam(params, 'de', ''), 'ate', ''))}
          >
            <RotateCcw /> Período das Configurações
          </Button>
        )}
        <p className="ml-auto text-right text-xs text-muted-foreground">
          {numero(filtrados.length)} de {numero(marcadores.length)} marcadores
          {semValor > 0 ? ` · ${numero(semValor)} sem valor (viram “-” no Slide)` : ''}
          <br />
          <a href={csv} className="inline-flex items-center gap-1 font-medium underline">
            <Download className="size-3" /> CSV
          </a>
        </p>
      </div>

      {grupos.length === 0 ? (
        <p className="text-sm text-muted-foreground">Nenhum marcador encontrado com esse filtro.</p>
      ) : (
        grupos.map((g) => (
          <Card key={g.chave}>
            <CardHeader>
              <CardTitle>{g.titulo}</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Marcador (clique para copiar)</TableHead>
                    <TableHead className="text-right">Valor atual</TableHead>
                    <TableHead>Origem</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {g.secoes.map((sec) => (
                    <Fragment key={`${g.chave}-${sec.titulo}`}>
                      {sec.titulo && (
                        <TableRow className="bg-muted/40 hover:bg-muted/40">
                          <TableCell colSpan={3} className="font-medium">
                            {sec.titulo}
                          </TableCell>
                        </TableRow>
                      )}
                      {sec.itens.map((item) => (
                        <TableRow key={item.marcador}>
                          <TableCell>
                            <button
                              type="button"
                              title="Clique para copiar"
                              className="font-mono text-xs underline-offset-2 hover:underline"
                              onClick={() => copiar(item.marcador)}
                            >
                              {item.marcador}
                            </button>
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            {item.valor || <span className="text-muted-foreground">–</span>}
                          </TableCell>
                          <TableCell className="text-muted-foreground">{item.origem}</TableCell>
                        </TableRow>
                      ))}
                    </Fragment>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  )
}
