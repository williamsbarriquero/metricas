import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Search, Trash2, UserPlus, Wand2 } from 'lucide-react'
import { Fragment, useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { SeletorEspacos } from '@/components/seletor-espacos'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { type MembroEquipe, type MembroEquipeIn, type ResultadoPadrao, api } from '@/lib/api'
import { SEM_COORDENADOR, opcoesCoordenador, secoesDe } from '@/lib/coordenadores'

const schema = z
  .object({
    nome: z.string(),
    account_id: z.string(),
    espacos: z.string(),
    incluir: z.boolean(),
    observacao: z.string(),
    coordenador: z.string(),
  })
  .refine((v) => v.nome.trim() !== '' || v.account_id.trim() !== '', {
    message: 'Informe o e-mail/nome ou o Account ID.',
    path: ['nome'],
  })

type Valores = z.infer<typeof schema>

const VAZIO: Valores = {
  nome: '',
  account_id: '',
  espacos: '',
  incluir: true,
  observacao: '',
  coordenador: '',
}

function avisoPadrao(r: ResultadoPadrao) {
  if (r.adicionados > 0 && r.coordenadores_atualizados > 0) {
    return `${r.adicionados} pessoa(s) da lista padrão adicionada(s). ${r.coordenadores_atualizados} coordenador(es) atualizado(s).`
  }
  if (r.adicionados > 0) return `${r.adicionados} pessoa(s) da lista padrão adicionada(s).`
  if (r.coordenadores_atualizados > 0)
    return `${r.coordenadores_atualizados} coordenador(es) atualizado(s).`
  return ''
}

function FormularioMembro({
  membro,
  aberto,
  coordenadores,
  onFechar,
}: {
  membro: MembroEquipe | 'novo' | null
  aberto: boolean
  coordenadores: string[]
  onFechar: () => void
}) {
  const qc = useQueryClient()
  const editando = membro && membro !== 'novo' ? membro : null
  const sugestoes =
    editando?.coordenador && !coordenadores.includes(editando.coordenador)
      ? [...coordenadores, editando.coordenador]
      : coordenadores
  const form = useForm<Valores>({
    resolver: zodResolver(schema),
    values: editando
      ? {
          nome: editando.nome,
          account_id: editando.account_id,
          espacos: editando.espacos,
          incluir: editando.incluir,
          observacao: editando.observacao,
          coordenador: editando.coordenador ?? '',
        }
      : VAZIO,
  })

  const salvar = useMutation({
    mutationFn: (v: MembroEquipeIn) =>
      editando ? api.atualizarMembro(editando.id, v) : api.criarMembro(v),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['equipe'] })
      void qc.invalidateQueries({ queryKey: ['metricas'] })
      toast.success(editando ? 'Pessoa atualizada.' : 'Pessoa adicionada.')
      onFechar()
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const erro = form.formState.errors.nome?.message

  return (
    <Dialog open={aberto} onOpenChange={(a) => !a && onFechar()}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={form.handleSubmit((v) => salvar.mutate(v))} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{editando ? 'Editar pessoa' : 'Adicionar pessoa'}</DialogTitle>
            <DialogDescription>
              Só as horas de quem está na Equipe (com “Incluir” ligado) entram na métrica T2.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-1.5">
            <Label htmlFor="nome">E-mail ou nome (como no Jira)</Label>
            <Input id="nome" placeholder="pessoa@rpe.tech" {...form.register('nome')} />
            {erro && <p className="text-xs text-destructive">{erro}</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="coordenador">Coordenador</Label>
            <Input
              id="coordenador"
              list="coordenadores-equipe"
              placeholder="Sem coordenador"
              {...form.register('coordenador')}
            />
            <datalist id="coordenadores-equipe">
              {sugestoes.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </datalist>
            <p className="text-xs text-muted-foreground">
              Só organiza a lista. Não muda quem entra nas horas.
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="account_id">Account ID (opcional)</Label>
            <Input id="account_id" placeholder="712020:abc..." {...form.register('account_id')} />
            <p className="text-xs text-muted-foreground">
              Útil quando o e-mail está oculto no Jira. Use “Resolver Account IDs” para preencher
              automaticamente.
            </p>
          </div>
          <div className="space-y-1.5">
            <Label>Espaços (opcional)</Label>
            <Controller
              control={form.control}
              name="espacos"
              render={({ field }) => (
                <SeletorEspacos value={field.value} onChange={field.onChange} />
              )}
            />
            <p className="text-xs text-muted-foreground">
              Sem seleção, as horas valem em todos os Espaços.
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="observacao">Observação</Label>
            <Textarea id="observacao" rows={2} {...form.register('observacao')} />
          </div>
          <Controller
            control={form.control}
            name="incluir"
            render={({ field }) => (
              <div className="flex items-center gap-2 text-sm">
                <Switch
                  id="incluir-horas"
                  aria-labelledby="incluir-horas-rotulo"
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
                <span id="incluir-horas-rotulo">Incluir nas horas</span>
              </div>
            )}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onFechar}>
              Cancelar
            </Button>
            <Button type="submit" disabled={salvar.isPending}>
              {salvar.isPending ? 'Salvando…' : 'Salvar'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export default function EquipePage() {
  const qc = useQueryClient()
  const [busca, setBusca] = useState('')
  const [coord, setCoord] = useState('todos')
  const [editando, setEditando] = useState<MembroEquipe | 'novo' | null>(null)
  const [excluir, setExcluir] = useState<MembroEquipe | null>(null)

  const { data: equipe, isLoading, error } = useQuery({ queryKey: ['equipe'], queryFn: api.equipe })
  const { data: canon = [] } = useQuery({
    queryKey: ['equipe-coordenadores'],
    queryFn: api.coordenadoresEquipe,
  })

  const invalidar = () => {
    void qc.invalidateQueries({ queryKey: ['equipe'] })
    void qc.invalidateQueries({ queryKey: ['metricas'] })
  }

  const alternar = useMutation({
    mutationFn: (m: MembroEquipe) => api.atualizarMembro(m.id, { ...m, incluir: !m.incluir }),
    onSuccess: invalidar,
    onError: (e: Error) => toast.error(e.message),
  })
  const remover = useMutation({
    mutationFn: (m: MembroEquipe) => api.removerMembro(m.id),
    onSuccess: () => {
      invalidar()
      setExcluir(null)
      toast.success('Pessoa removida.')
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const completar = useMutation({
    mutationFn: api.completarPadrao,
    onSuccess: (r) => {
      invalidar()
      const aviso = avisoPadrao(r)
      if (aviso) toast.success(aviso)
      else toast.info('Toda a lista padrão já está na Equipe.')
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const resolver = useMutation({
    mutationFn: api.resolverAccountIds,
    onSuccess: (r) => {
      invalidar()
      const total = Object.keys(r).length
      const ok = Object.values(r).filter((v) => v.startsWith('ok')).length
      if (!total) toast.info('Todas as pessoas já têm Account ID.')
      else
        toast.success(`${ok} de ${total} Account IDs resolvidos.`, {
          description:
            Object.entries(r)
              .filter(([, v]) => !v.startsWith('ok'))
              .map(([n, v]) => `${n}: ${v}`)
              .join('\n') || undefined,
        })
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const opcoes = useMemo(
    () =>
      opcoesCoordenador(
        canon,
        (equipe ?? []).map((m) => m.coordenador),
      ),
    [canon, equipe],
  )
  const filtrada = useMemo(() => {
    const q = busca.trim().toLowerCase()
    return (equipe ?? []).filter((m) => {
      const grupo = m.coordenador ?? ''
      const texto = `${m.nome} ${m.account_id} ${m.espacos} ${m.observacao} ${grupo}`
      if (q && !texto.toLowerCase().includes(q)) return false
      if (coord === SEM_COORDENADOR) return grupo === ''
      if (coord !== 'todos') return grupo === coord
      return true
    })
  }, [equipe, busca, coord])
  const secoes = useMemo(() => secoesDe(filtrada, canon, (m) => m.coordenador), [filtrada, canon])
  const ativos = (equipe ?? []).filter((m) => m.incluir).length

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Equipe</CardTitle>
          <CardDescription>
            Pessoas que entram nas horas. Histórias e Test Bugs não usam esta lista. A Vazão traz
            todo o Espaço. Incluir alguém novo exige sincronizar de novo.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative">
              <Search className="absolute top-2 left-2.5 size-4 text-muted-foreground" />
              <Input
                className="w-64 pl-8"
                placeholder="Buscar pessoa…"
                value={busca}
                onChange={(e) => setBusca(e.target.value)}
              />
            </div>
            <Select value={coord} onValueChange={(v) => v && setCoord(v)} items={opcoes}>
              <SelectTrigger className="w-72" aria-label="Coordenador">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {opcoes.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Badge variant="secondary">
              {ativos} incluída(s) de {equipe?.length ?? 0}
            </Badge>
            <div className="ml-auto flex gap-2">
              <Button
                variant="outline"
                onClick={() => completar.mutate()}
                disabled={completar.isPending}
              >
                <UserPlus /> {completar.isPending ? 'Atualizando…' : 'Completar com a lista padrão'}
              </Button>
              <Button
                variant="outline"
                onClick={() => resolver.mutate()}
                disabled={resolver.isPending}
              >
                <Wand2 /> {resolver.isPending ? 'Resolvendo…' : 'Resolver Account IDs'}
              </Button>
              <Button onClick={() => setEditando('novo')}>
                <Plus /> Adicionar pessoa
              </Button>
            </div>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{(error as Error).message}</AlertDescription>
            </Alert>
          )}

          {isLoading ? (
            <Skeleton className="h-48" />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Incluir</TableHead>
                  <TableHead>E-mail ou nome</TableHead>
                  <TableHead>Coordenador</TableHead>
                  <TableHead>Account ID</TableHead>
                  <TableHead>Espaços</TableHead>
                  <TableHead>Observação</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtrada.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="py-10 text-center text-muted-foreground">
                      {equipe?.length
                        ? 'Nenhuma pessoa encontrada.'
                        : 'Ninguém cadastrado ainda. Adicione as pessoas para liberar a métrica de horas.'}
                    </TableCell>
                  </TableRow>
                )}
                {secoes.map((secao) => (
                  <Fragment key={secao.chave}>
                    <TableRow className="bg-muted/50 hover:bg-muted/50">
                      <TableCell colSpan={7} className="font-semibold">
                        {secao.rotulo}
                        <span className="ml-2 font-normal text-muted-foreground">
                          {secao.itens.length}
                        </span>
                      </TableCell>
                    </TableRow>
                    {secao.itens.map((m) => (
                      <TableRow key={m.id} className={m.incluir ? '' : 'opacity-60'}>
                        <TableCell>
                          <Switch
                            checked={m.incluir}
                            onCheckedChange={() => alternar.mutate(m)}
                            aria-label="Incluir nas horas"
                          />
                        </TableCell>
                        <TableCell className="font-medium">{m.nome || '—'}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {m.coordenador || '—'}
                        </TableCell>
                        <TableCell
                          className="max-w-48 truncate font-mono text-xs text-muted-foreground"
                          title={m.account_id}
                        >
                          {m.account_id || '—'}
                        </TableCell>
                        <TableCell>
                          {m.espacos ? (
                            <div className="flex flex-wrap gap-1">
                              {m.espacos.split(',').map((e) => (
                                <Badge key={e} variant="outline">
                                  {e.trim()}
                                </Badge>
                              ))}
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">Todos</span>
                          )}
                        </TableCell>
                        <TableCell
                          className="max-w-64 truncate text-muted-foreground"
                          title={m.observacao}
                        >
                          {m.observacao}
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="Editar"
                              onClick={() => setEditando(m)}
                            >
                              <Pencil />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="Remover"
                              onClick={() => setExcluir(m)}
                            >
                              <Trash2 />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </Fragment>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <FormularioMembro
        membro={editando}
        aberto={editando !== null}
        coordenadores={canon}
        onFechar={() => setEditando(null)}
      />

      <Dialog open={excluir !== null} onOpenChange={(a) => !a && setExcluir(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Remover pessoa?</DialogTitle>
            <DialogDescription>
              {excluir?.nome || excluir?.account_id} deixa de contar nas horas. Para manter o
              cadastro, desligue “Incluir”.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setExcluir(null)}>
              Cancelar
            </Button>
            <Button
              variant="destructive"
              disabled={remover.isPending}
              onClick={() => excluir && remover.mutate(excluir)}
            >
              Remover
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
