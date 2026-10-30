import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RotateCcw, Save } from 'lucide-react'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'sonner'

import { SeletorEspacos } from '@/components/seletor-espacos'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { type ConfigItem, api } from '@/lib/api'

type Valores = Record<string, string>

const GRUPOS: { titulo: string; descricao: string; chaves: string[] }[] = [
  {
    titulo: 'Período',
    descricao:
      'Define o que é coletado do Jira e os meses exibidos (de 3 a no máximo 12 meses). Histórias e Test Bugs usam o dia 1 do mês seguinte à data fim. Os gráficos de T1 só batem com o Jira depois de uma nova sincronização.',
    chaves: ['inicio', 'fim', 'incluirDiaFinal'],
  },
  {
    titulo: 'Espaços (projetos) por métrica',
    descricao:
      'Histórias, Test Bugs e Incidentes usam estas listas na coleta. A lista de Horas é a dos espaços mapeados no dashboard.',
    chaves: ['projetosHistorias', 'projetosIncidentes', 'projetosHoras'],
  },
  {
    titulo: 'Filtros do Jira',
    descricao:
      'Tipos e status dos incidentes (filtros 29828 e 29812). A JQL de Histórias e Test Bugs é fixa e está na tela JQL.',
    chaves: ['tiposIncidentes', 'statusExcluidosEstoque'],
  },
  {
    titulo: 'Contas de Esforço',
    descricao: 'Nome exato das opções do campo “Conta de Esforço” no Jira.',
    chaves: ['contaSustentacao', 'contaEvolucao', 'contaRecesso'],
  },
  {
    titulo: 'Coleta de horas',
    descricao:
      'Com a busca em todos os espaços ligada, a coleta de horas não filtra por projeto. O dashboard continua nos espaços mapeados até a opção da própria tela ser ligada.',
    chaves: ['horasTodosEspacos', 'horasModo'],
  },
  {
    titulo: 'Slide',
    descricao: 'O modelo não é alterado: cada geração faz uma cópia e preenche os marcadores.',
    chaves: ['slideModelo', 'slidePasta', 'slideNome'],
  },
]

const MODOS_HORAS = [
  { value: 'bulk', label: 'Em massa (rápido, recomendado)' },
  { value: 'por_issue', label: 'Uma chamada por issue (mais lento)' },
]

function Campo({ item, form }: { item: ConfigItem; form: ReturnType<typeof useForm<Valores>> }) {
  const { control, register } = form
  return (
    <div className="space-y-1.5">
      <Label htmlFor={item.chave}>{item.rotulo}</Label>
      {item.tipo === 'data' ? (
        <Input id={item.chave} type="date" className="w-48" {...register(item.chave)} />
      ) : item.tipo === 'simnao' ? (
        <Controller
          control={control}
          name={item.chave}
          render={({ field }) => (
            <div>
              <Switch
                id={item.chave}
                checked={!['não', 'nao'].includes((field.value ?? 'Sim').trim().toLowerCase())}
                onCheckedChange={(v) => field.onChange(v ? 'Sim' : 'Não')}
              />
            </div>
          )}
        />
      ) : item.chave.startsWith('projetos') ? (
        <Controller
          control={control}
          name={item.chave}
          render={({ field }) => (
            <SeletorEspacos value={field.value ?? ''} onChange={field.onChange} />
          )}
        />
      ) : item.chave === 'horasModo' ? (
        <Controller
          control={control}
          name={item.chave}
          render={({ field }) => (
            <Select
              value={field.value ?? 'bulk'}
              onValueChange={(v) => v && field.onChange(v)}
              items={MODOS_HORAS}
            >
              <SelectTrigger className="w-80">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MODOS_HORAS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
      ) : (
        <Input id={item.chave} {...register(item.chave)} />
      )}
      {item.nota && <p className="text-xs text-muted-foreground">{item.nota}</p>}
    </div>
  )
}

export default function ConfiguracoesPage() {
  const qc = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['config'], queryFn: api.config })
  const form = useForm<Valores>({
    values: data ? Object.fromEntries(data.itens.map((i) => [i.chave, i.valor])) : undefined,
  })

  const desconectar = useMutation({
    mutationFn: api.desconectarGoogle,
    onSuccess: (r) => {
      qc.setQueryData(['config'], r)
      toast.success('Acesso Google revogado e conta desconectada.')
    },
    onError: (e: Error) => toast.error('Não foi possível desconectar', { description: e.message }),
  })

  const salvar = useMutation({
    mutationFn: (v: Valores) => api.salvarConfig(v),
    onSuccess: (r) => {
      qc.setQueryData(['config'], r)
      void qc.invalidateQueries({ queryKey: ['metricas'] })
      void qc.invalidateQueries({ queryKey: ['glossario'] })
      void qc.invalidateQueries({ queryKey: ['jql'] })
      void qc.invalidateQueries({ queryKey: ['horas-equipe'] })
      toast.success('Configurações salvas.', {
        description: 'Se mudou período ou Espaços, rode uma nova coleta em Sincronização.',
      })
    },
    onError: (e: Error) => toast.error('Não foi possível salvar', { description: e.message }),
  })

  if (isLoading) return <Skeleton className="h-96" />
  if (error || !data)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {(error as Error | null)?.message ?? 'Erro ao carregar a configuração.'}
        </AlertDescription>
      </Alert>
    )

  const porChave = Object.fromEntries(data.itens.map((i) => [i.chave, i]))

  const google = data.google

  return (
    <div className="space-y-4">
      <form onSubmit={form.handleSubmit((v) => salvar.mutate(v))} className="space-y-4">
        {GRUPOS.map((g) => (
          <Card key={g.titulo}>
            <CardHeader>
              <CardTitle>{g.titulo}</CardTitle>
              <CardDescription>{g.descricao}</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-5 md:grid-cols-2">
              {g.chaves.map(
                (c) =>
                  porChave[c] && (
                    <div
                      key={c}
                      className={
                        c.startsWith('projetos') || c.startsWith('status') || c.startsWith('tipos')
                          ? 'md:col-span-2'
                          : ''
                      }
                    >
                      <Campo item={porChave[c]} form={form} />
                    </div>
                  ),
              )}
            </CardContent>
          </Card>
        ))}

        <div className="sticky bottom-4 flex justify-end gap-2 rounded-lg border bg-background/95 p-3 shadow-sm backdrop-blur">
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              form.reset(Object.fromEntries(data.itens.map((i) => [i.chave, i.padrao])), {
                keepDefaultValues: true,
              })
            }
          >
            <RotateCcw /> Restaurar padrões
          </Button>
          <Button type="submit" disabled={salvar.isPending}>
            <Save /> {salvar.isPending ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </form>

      <Card>
        <CardHeader>
          <CardTitle>Google</CardTitle>
          <CardDescription>
            Conta usada para copiar o Slide modelo e preencher os marcadores. O botão de gerar
            continua na tela MapaSlide.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          {data.jira.configurado ? (
            <p className="text-muted-foreground">Jira: {data.jira.email}</p>
          ) : (
            <p>
              Preencha JIRA_EMAIL e JIRA_TOKEN no arquivo .env da pasta spa e reinicie o servidor.
            </p>
          )}
          {google.demo ? (
            <p>
              Modo demonstração: a geração real do Slide fica desligada. O MapaSlide ainda mostra a
              pré-visualização.
            </p>
          ) : !google.configurado ? (
            <p>Preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env e reinicie o servidor.</p>
          ) : google.conectado ? (
            <div className="flex flex-wrap items-center gap-3">
              <p>Conectado{google.conta ? `: ${google.conta}` : ''}.</p>
              <Button
                type="button"
                variant="outline"
                disabled={desconectar.isPending}
                onClick={() => desconectar.mutate()}
              >
                Desconectar
              </Button>
            </div>
          ) : (
            <a href="/google/conectar" className={buttonVariants()}>
              Conectar conta Google
            </a>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
