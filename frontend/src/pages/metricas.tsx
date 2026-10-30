import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Download, Info, RotateCcw } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { PaletaEspaco, TituloEspaco } from '@/components/dashboard/paleta-espaco'
import { SecaoT1, SecaoT2, SecaoT3 } from '@/components/dashboard/secoes'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { type Bloco, api, comQuery } from '@/lib/api'
import { data as formatarData, dataHora, fimDoMes, inicioDoMes, paraMes } from '@/lib/format'

function definirParam(params: URLSearchParams, chave: string, valor: string) {
  const next = new URLSearchParams(params)
  if (valor) next.set(chave, valor)
  else next.delete(chave)
  return next
}

function Blocos({
  blocos,
  vazio,
  acoes,
  children,
}: {
  blocos: Bloco[]
  vazio: string
  acoes: (bloco: Bloco) => { to: string; rotulo: string }[]
  children: (bloco: Bloco) => ReactNode
}) {
  if (blocos.length === 0) return <p className="text-sm text-muted-foreground">{vazio}</p>
  return (
    <div className="space-y-6">
      {blocos.map((bloco) => (
        <PaletaEspaco key={bloco.chave} bloco={bloco} className="space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <TituloEspaco bloco={bloco} tituloClassName="text-base" />
            <div className="flex flex-wrap gap-2">
              {acoes(bloco).map((a) => (
                <Link
                  key={a.to}
                  to={a.to}
                  className={buttonVariants({ variant: 'outline', size: 'sm' })}
                >
                  {a.rotulo}
                </Link>
              ))}
            </div>
          </div>
          {children(bloco)}
        </PaletaEspaco>
      ))}
    </div>
  )
}

export default function MetricasPage() {
  const [params, setParams] = useSearchParams()
  const de = params.get('de') || undefined
  const ate = params.get('ate') || undefined

  const metricas = useQuery({
    queryKey: ['metricas', de, ate],
    queryFn: () => api.metricas(de && inicioDoMes(de), ate && fimDoMes(ate)),
    placeholderData: (anterior) => anterior,
  })
  const glossario = useQuery({ queryKey: ['glossario'], queryFn: api.glossario })

  if (metricas.isLoading || glossario.isLoading) return <Skeleton className="h-96" />
  if (metricas.isError || !metricas.data) {
    return (
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertTitle>Não foi possível carregar as métricas</AlertTitle>
        <AlertDescription>
          {metricas.error instanceof Error ? metricas.error.message : 'Erro desconhecido.'}
        </AlertDescription>
      </Alert>
    )
  }

  const m = metricas.data
  const flags = new Map((glossario.data?.espacos ?? []).map((e) => [e.chave, e]))
  const doTier = (
    aceita: (f: { historias: boolean; incidentes: boolean; horas: boolean }) => boolean,
  ) => {
    const lista = m.espacos.filter((b) => {
      const f = flags.get(b.chave)
      return !f || aceita(f)
    })
    if (m.todos) lista.push(m.todos)
    return lista
  }
  const espacoDe = (chave: string) => (chave === 'TODOS' ? undefined : chave)
  const mesInicio = de ?? paraMes(m.periodo.inicio)
  const mesFim = ate ?? paraMes(m.periodo.fim)

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end gap-4">
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
        <a
          className={buttonVariants({ variant: 'outline', size: 'sm' })}
          href={comQuery('/csv/metricas', { de, ate })}
        >
          <Download /> CSV (todas as seções)
        </a>
        <p className="ml-auto text-right text-xs text-muted-foreground">
          Período: {formatarData(m.periodo.inicio)} a {formatarData(m.periodo.fim)}
          <br />
          {m.atualizado_em
            ? `Última coleta: ${dataHora(m.atualizado_em)}`
            : 'Nenhuma coleta concluída'}
        </p>
      </div>

      {m.avisos.map((a) => (
        <Alert key={a}>
          <Info />
          <AlertDescription>{a}</AlertDescription>
        </Alert>
      ))}

      <Tabs defaultValue="t1">
        <TabsList>
          <TabsTrigger value="t1">1. Histórias vs Test Bugs</TabsTrigger>
          <TabsTrigger value="t2">2. Horas por Conta</TabsTrigger>
          <TabsTrigger value="t3">3. Vazão de Incidentes</TabsTrigger>
        </TabsList>
        <TabsContent value="t1" className="pt-4">
          <Blocos
            blocos={doTier((f) => f.historias)}
            vazio="Nenhum Espaço habilitado para Histórias e Test Bugs."
            acoes={(b) => [
              {
                to: comQuery('/dados/historias', { espaco: espacoDe(b.chave) }),
                rotulo: 'Ver histórias',
              },
            ]}
          >
            {(b) => <SecaoT1 bloco={b} />}
          </Blocos>
        </TabsContent>
        <TabsContent value="t2" className="pt-4">
          <Blocos
            blocos={doTier((f) => f.horas)}
            vazio="Nenhum Espaço com horas no período."
            acoes={(b) => [
              {
                to: comQuery('/dados/horas', { espaco: espacoDe(b.chave), equipe: '1' }),
                rotulo: 'Ver worklogs',
              },
            ]}
          >
            {(b) => <SecaoT2 bloco={b} equipeAtiva={m.equipe_ativa} />}
          </Blocos>
        </TabsContent>
        <TabsContent value="t3" className="pt-4">
          <Blocos
            blocos={doTier((f) => f.incidentes)}
            vazio="Nenhum Espaço habilitado para Incidentes."
            acoes={(b) => {
              const espaco = espacoDe(b.chave)
              return [
                { to: comQuery('/dados/criados', { espaco }), rotulo: 'Criados' },
                { to: comQuery('/dados/resolvidos', { espaco }), rotulo: 'Resolvidos' },
                { to: comQuery('/dados/estoque', { espaco }), rotulo: 'Estoque' },
              ]
            }}
          >
            {(b) => <SecaoT3 bloco={b} />}
          </Blocos>
        </TabsContent>
      </Tabs>
    </div>
  )
}
