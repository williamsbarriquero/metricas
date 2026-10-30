import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Loader2,
  XCircle,
} from 'lucide-react'
import { Fragment, useId, useState } from 'react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { type SyncRun, api } from '@/lib/api'
import { data as formatarData, dataHora, duracao, numero } from '@/lib/format'

const BLOCOS = [
  { id: 'historias', rotulo: 'Histórias' },
  { id: 'testbugs', rotulo: 'Test Bugs' },
  { id: 'incidentes', rotulo: 'Incidentes' },
  { id: 'horas', rotulo: 'Horas' },
]
const NOME_BLOCO: Record<string, string> = {
  tudo: 'Tudo',
  ...Object.fromEntries(BLOCOS.map((b) => [b.id, b.rotulo])),
}

function StatusBadge({ status }: { status: SyncRun['status'] }) {
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

export default function SincronizacaoPage() {
  const qc = useQueryClient()
  const idForcar = useId()
  const [aberta, setAberta] = useState<number | null>(null)
  const [forcar, setForcar] = useState(false)
  const [confirmando, setConfirmando] = useState<string | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['sync-status'],
    queryFn: api.syncStatus,
    refetchInterval: (q) => (q.state.data?.em_andamento ? 1500 : false),
  })

  const iniciar = useMutation({
    mutationFn: ({ bloco, forcar: f }: { bloco: string; forcar: boolean }) =>
      api.iniciarSync(bloco, f),
    onSuccess: (r, vars) => {
      toast.info(
        vars.forcar
          ? `Sincronia completa iniciada: ${NOME_BLOCO[r.bloco]}.`
          : `Coleta iniciada: ${NOME_BLOCO[r.bloco]}.`,
      )
      void qc.invalidateQueries({ queryKey: ['sync-status'] })
    },
    onError: (e: Error) => toast.error('Não foi possível iniciar', { description: e.message }),
  })

  const disparar = (bloco: string) => {
    if (forcar) {
      setConfirmando(bloco)
      return
    }
    iniciar.mutate({ bloco, forcar: false })
  }

  // quando uma coleta termina, atualiza as métricas
  const ultima = data?.ultima
  const [vista, setVista] = useState<number | null>(null)
  if (ultima && ultima.status !== 'executando' && vista !== ultima.id) {
    setVista(ultima.id)
    void qc.invalidateQueries({ queryKey: ['metricas'] })
    if (vista !== null) {
      if (ultima.status === 'ok')
        toast.success('Coleta concluída.', {
          description: `${numero(ultima.linhas)} linhas gravadas.`,
        })
      else toast.error('A coleta falhou.', { description: ultima.mensagem.slice(0, 200) })
    }
  }

  if (isLoading) return <Skeleton className="h-96" />
  if (error || !data)
    return (
      <Alert variant="destructive">
        <AlertDescription>
          {(error as Error | null)?.message ?? 'Erro ao carregar o status.'}
        </AlertDescription>
      </Alert>
    )

  const ocupado = data.em_andamento || iniciar.isPending

  return (
    <div className="space-y-4">
      {data.demo && (
        <Alert>
          <AlertTitle>Modo demonstração</AlertTitle>
          <AlertDescription>
            Os dados são fictícios. A coleta no Jira fica desligada.
          </AlertDescription>
        </Alert>
      )}
      {!data.jira_configurado && !data.demo && (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertTitle>Credenciais do Jira não configuradas</AlertTitle>
          <AlertDescription>
            Preencha <code>JIRA_EMAIL</code> e <code>JIRA_TOKEN</code> no <code>.env</code> da pasta
            spa e reinicie o servidor.
          </AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Atualizar dados do Jira</CardTitle>
          <CardDescription>
            A coleta normal só busca o que ainda não foi sincronizado. Use o interruptor para
            rebuscar o período inteiro quando houver ajuste no Jira. Roda em segundo plano.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <Switch
              id={idForcar}
              checked={forcar}
              onCheckedChange={setForcar}
              disabled={ocupado || !data.jira_configurado}
            />
            <Label htmlFor={idForcar} className="text-sm font-normal">
              Forçar período completo
            </Label>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button disabled={ocupado || !data.jira_configurado} onClick={() => disparar('tudo')}>
              {data.em_andamento ? <Loader2 className="animate-spin" /> : null} Atualizar tudo
            </Button>
            {BLOCOS.map((b) => (
              <Button
                key={b.id}
                variant="outline"
                disabled={ocupado || !data.jira_configurado}
                onClick={() => disparar(b.id)}
              >
                {b.rotulo}
              </Button>
            ))}
          </div>

          {data.em_andamento && ultima && (
            <div className="space-y-2 rounded-lg border p-4">
              <div className="flex items-center justify-between text-sm">
                <span className="font-medium">
                  {NOME_BLOCO[ultima.bloco]}: {ultima.etapa}
                </span>
                <span className="tabular-nums text-muted-foreground">{ultima.progresso}%</span>
              </div>
              <Progress value={ultima.progresso} />
              <p className="text-xs text-muted-foreground">
                {numero(ultima.linhas)} linhas gravadas · {duracao(ultima.inicio, null)} · período{' '}
                {formatarData(ultima.periodo_inicio)} a {formatarData(ultima.periodo_fim)}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Histórico</CardTitle>
          <CardDescription>
            Últimas 20 execuções. Clique numa linha para ver os detalhes e avisos.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead>Início</TableHead>
                <TableHead>Bloco</TableHead>
                <TableHead>Período</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Linhas</TableHead>
                <TableHead className="text-right">Duração</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.historico.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="py-10 text-center text-muted-foreground">
                    Nenhuma coleta executada ainda.
                  </TableCell>
                </TableRow>
              )}
              {data.historico.map((r) => (
                <Fragment key={r.id}>
                  <TableRow
                    className="cursor-pointer"
                    onClick={() => setAberta(aberta === r.id ? null : r.id)}
                  >
                    <TableCell>
                      {aberta === r.id ? (
                        <ChevronDown className="size-4" />
                      ) : (
                        <ChevronRight className="size-4" />
                      )}
                    </TableCell>
                    <TableCell>{dataHora(r.inicio)}</TableCell>
                    <TableCell>{NOME_BLOCO[r.bloco] ?? r.bloco}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {r.periodo_inicio &&
                        `${formatarData(r.periodo_inicio)} a ${formatarData(r.periodo_fim)}`}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={r.status} />
                    </TableCell>
                    <TableCell className="text-right tabular-nums">{numero(r.linhas)}</TableCell>
                    <TableCell className="text-right tabular-nums">
                      {duracao(r.inicio, r.fim)}
                    </TableCell>
                  </TableRow>
                  {aberta === r.id && (
                    <TableRow className="hover:bg-transparent">
                      <TableCell />
                      <TableCell colSpan={6}>
                        <pre
                          className={`whitespace-pre-wrap py-2 text-xs ${r.status === 'erro' ? 'text-destructive' : 'text-muted-foreground'}`}
                        >
                          {r.mensagem || 'Sem mensagens.'}
                        </pre>
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={confirmando !== null} onOpenChange={(a) => !a && setConfirmando(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Forçar período completo?</DialogTitle>
            <DialogDescription>
              A consulta volta a percorrer o período configurado inteiro no Jira, inclusive meses já
              sincronizados, para refletir alteração feita depois da última coleta.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmando(null)}>
              Cancelar
            </Button>
            <Button
              disabled={iniciar.isPending}
              onClick={() => {
                if (!confirmando) return
                const bloco = confirmando
                setConfirmando(null)
                iniciar.mutate({ bloco, forcar: true })
              }}
            >
              Confirmar
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
