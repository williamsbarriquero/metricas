import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Info, RotateCcw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

import { Comparativo } from '@/components/dashboard/comparativo'
import { Kpis } from '@/components/dashboard/kpis'
import { PaletaEspaco, TituloEspaco } from '@/components/dashboard/paleta-espaco'
import { SecaoT1, SecaoT2, SecaoT3 } from '@/components/dashboard/secoes'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api } from '@/lib/api'
import { data as formatarData, dataHora, fimDoMes, inicioDoMes, paraMes } from '@/lib/format'
import { agruparPorBU } from '@/lib/paleta'

export default function DashboardPage() {
  const [espaco, setEspaco] = useState('TODOS')
  const [de, setDe] = useState<string | undefined>()
  const [ate, setAte] = useState<string | undefined>()
  const [horasFora, setHorasFora] = useState(false)

  const consulta = useQuery({
    queryKey: ['metricas', de, ate, horasFora],
    queryFn: () => api.metricas(de && inicioDoMes(de), ate && fimDoMes(ate), horasFora),
    placeholderData: (anterior) => anterior,
  })

  const grupos = useMemo(
    () => (consulta.data ? agruparPorBU(consulta.data.espacos) : []),
    [consulta.data],
  )

  const espacoAtivo =
    espaco !== 'TODOS' && consulta.data && !consulta.data.espacos.some((e) => e.chave === espaco)
      ? 'TODOS'
      : espaco

  const itensSelect = useMemo(() => {
    const itens = [{ value: 'TODOS', label: 'Todos os Espaços' }]
    for (const g of grupos) {
      for (const e of g.itens) {
        itens.push({ value: e.chave, label: `${e.chave} - ${e.nome}` })
      }
    }
    return itens
  }, [grupos])

  if (consulta.isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full max-w-xl" />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
        <Skeleton className="h-80" />
      </div>
    )
  }

  if (consulta.isError || !consulta.data) {
    return (
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertTitle>Não foi possível carregar as métricas</AlertTitle>
        <AlertDescription>
          {consulta.error instanceof Error ? consulta.error.message : 'Erro desconhecido.'}{' '}
          Verifique se o servidor está rodando.
        </AlertDescription>
      </Alert>
    )
  }

  const m = consulta.data
  const bloco = espacoAtivo === 'TODOS' ? m.todos : m.espacos.find((e) => e.chave === espacoAtivo)
  const mesInicio = de ?? paraMes(m.periodo.inicio)
  const mesFim = ate ?? paraMes(m.periodo.fim)

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end gap-4">
        <div className="space-y-1.5">
          <Label>Espaço</Label>
          <Select value={espacoAtivo} onValueChange={(v) => v && setEspaco(v)} items={itensSelect}>
            <SelectTrigger className="w-80">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="max-h-96">
              <SelectItem value="TODOS">Todos os Espaços</SelectItem>
              {grupos.map((g) => (
                <SelectGroup key={g.bu || g.nome}>
                  <SelectLabel>{g.nome}</SelectLabel>
                  {g.itens.map((e) => (
                    <SelectItem key={e.chave} value={e.chave}>
                      {e.chave} - {e.nome}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex items-center gap-2 pb-2 text-sm">
          <Switch
            id="horas-fora"
            aria-labelledby="horas-fora-rotulo"
            checked={horasFora}
            onCheckedChange={setHorasFora}
          />
          <span id="horas-fora-rotulo">Ver horas fora dos espaços mapeados</span>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="de">De</Label>
          <Input
            id="de"
            type="month"
            className="w-40"
            value={mesInicio}
            onChange={(e) => e.target.value && setDe(e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="ate">Até</Label>
          <Input
            id="ate"
            type="month"
            className="w-40"
            value={mesFim}
            onChange={(e) => e.target.value && setAte(e.target.value)}
          />
        </div>
        {(de || ate) && (
          <Button
            variant="ghost"
            onClick={() => {
              setDe(undefined)
              setAte(undefined)
            }}
          >
            <RotateCcw /> Período da Config
          </Button>
        )}
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
          <AlertDescription>
            {a}{' '}
            {a.includes('Sincronização') && (
              <Link to="/sincronizacao" className="font-medium underline">
                Ir para Sincronização
              </Link>
            )}
            {a.includes('Equipe') && (
              <Link to="/equipe" className="font-medium underline">
                Ir para Equipe
              </Link>
            )}
          </AlertDescription>
        </Alert>
      ))}

      <Tabs defaultValue="espaco">
        <TabsList>
          <TabsTrigger value="espaco">Visão do Espaço</TabsTrigger>
          <TabsTrigger value="comparativo">Comparativo</TabsTrigger>
        </TabsList>

        <TabsContent value="espaco" className="space-y-6 pt-2">
          {bloco ? (
            <PaletaEspaco bloco={bloco} className="space-y-6">
              <div>
                <TituloEspaco bloco={bloco} />
                {!bloco.tem_dados && (
                  <p className="text-sm text-muted-foreground">
                    Este Espaço não tem dados coletados no período.
                  </p>
                )}
              </div>
              <Kpis bloco={bloco} />
              <SecaoT1 bloco={bloco} />
              <SecaoT2 bloco={bloco} equipeAtiva={m.equipe_ativa} />
              <SecaoT3 bloco={bloco} />
            </PaletaEspaco>
          ) : (
            <p className="text-sm text-muted-foreground">Espaço não encontrado.</p>
          )}
        </TabsContent>

        <TabsContent value="comparativo" className="pt-2">
          <Comparativo blocos={m.espacos} todos={m.todos} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
