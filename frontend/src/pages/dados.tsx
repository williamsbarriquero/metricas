import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Download } from 'lucide-react'
import { Link, useParams, useSearchParams } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { api, comQuery } from '@/lib/api'
import { numero } from '@/lib/format'

const ABAS = [
  { id: 'historias', rotulo: 'Histórias' },
  { id: 'testbugs', rotulo: 'Test Bugs' },
  { id: 'criados', rotulo: 'Criados' },
  { id: 'resolvidos', rotulo: 'Resolvidos' },
  { id: 'estoque', rotulo: 'Estoque' },
  { id: 'horas', rotulo: 'Horas' },
]

const FLUXOS = [
  { value: 'todos', label: 'Todos' },
  { value: 'Principal', label: 'Principal' },
  { value: 'Alternativo', label: 'Alternativo' },
  { value: 'Exceção', label: 'Exceção' },
]

function definir(params: URLSearchParams, chave: string, valor: string) {
  const next = new URLSearchParams(params)
  if (valor) next.set(chave, valor)
  else next.delete(chave)
  if (chave !== 'pagina') next.delete('pagina')
  return next
}

export default function DadosPage() {
  const { aba = 'historias' } = useParams()
  const [params, setParams] = useSearchParams()
  const espaco = params.get('espaco') || ''
  const mes = params.get('mes') || ''
  const q = params.get('q') || ''
  const fluxo = params.get('fluxo') || ''
  const conta = params.get('conta') || ''
  const autor = params.get('autor') || ''
  const tipo = params.get('tipo') || ''
  const equipe = params.get('equipe') === '1'
  const pagina = params.get('pagina') || ''

  const espacos = useQuery({ queryKey: ['espacos'], queryFn: api.espacos })
  const consulta = useQuery({
    queryKey: ['dados', aba, espaco, mes, q, fluxo, conta, autor, tipo, equipe, pagina],
    queryFn: () =>
      api.dados(aba, {
        espaco: espaco || undefined,
        mes: mes || undefined,
        q: q || undefined,
        fluxo: fluxo || undefined,
        conta: conta || undefined,
        autor: autor || undefined,
        tipo: tipo || undefined,
        equipe: equipe ? '1' : undefined,
        pagina: pagina || undefined,
      }),
    placeholderData: (anterior) => anterior,
  })

  const opcoesEspaco = [
    { value: 'todos', label: 'Todos os Espaços' },
    ...(espacos.data ?? []).map((e) => ({ value: e.chave, label: `${e.chave} — ${e.nome}` })),
  ]
  const filtrosCsv = {
    espaco: espaco || undefined,
    mes: mes || undefined,
    q: q || undefined,
    fluxo: fluxo || undefined,
    conta: conta || undefined,
    autor: autor || undefined,
    tipo: tipo || undefined,
    equipe: equipe ? '1' : undefined,
  }
  const buscaSemPagina = (() => {
    const next = new URLSearchParams(params)
    next.delete('pagina')
    const s = next.toString()
    return s ? `?${s}` : ''
  })()

  return (
    <div className="space-y-4">
      <nav className="flex flex-wrap gap-2" aria-label="Abas de dados">
        {ABAS.map((item) => (
          <Link
            key={item.id}
            to={{ pathname: `/dados/${item.id}`, search: buscaSemPagina }}
            className={buttonVariants({
              variant: item.id === aba ? 'default' : 'outline',
              size: 'sm',
            })}
          >
            {item.rotulo}
          </Link>
        ))}
      </nav>

      <div className="flex flex-wrap items-end gap-4">
        <div className="space-y-1.5">
          <Label>Espaço</Label>
          <Select
            value={espaco || 'todos'}
            onValueChange={(v) =>
              setParams(definir(params, 'espaco', !v || v === 'todos' ? '' : v))
            }
            items={opcoesEspaco}
          >
            <SelectTrigger className="w-80">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="max-h-96">
              {opcoesEspaco.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="mes">Mês</Label>
          <Input
            id="mes"
            type="month"
            className="w-40"
            value={mes}
            onChange={(e) => setParams(definir(params, 'mes', e.target.value))}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="q">Buscar</Label>
          <Input
            id="q"
            type="search"
            className="w-56"
            placeholder="chave, resumo..."
            value={q}
            onChange={(e) => setParams(definir(params, 'q', e.target.value))}
          />
        </div>
        {aba === 'testbugs' && (
          <div className="space-y-1.5">
            <Label>Fluxo</Label>
            <Select
              value={fluxo || 'todos'}
              onValueChange={(v) =>
                setParams(definir(params, 'fluxo', !v || v === 'todos' ? '' : v))
              }
              items={FLUXOS}
            >
              <SelectTrigger className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FLUXOS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {aba === 'horas' && (
          <>
            <div className="space-y-1.5">
              <Label htmlFor="conta">Conta</Label>
              <Input
                id="conta"
                className="w-44"
                placeholder="Sustentação..."
                value={conta}
                onChange={(e) => setParams(definir(params, 'conta', e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="autor">Autor</Label>
              <Input
                id="autor"
                className="w-44"
                value={autor}
                onChange={(e) => setParams(definir(params, 'autor', e.target.value))}
              />
            </div>
            <Label className="mb-2 flex items-center gap-2">
              <Checkbox
                checked={equipe}
                onCheckedChange={(v) => setParams(definir(params, 'equipe', v === true ? '1' : ''))}
              />
              Só Equipe
            </Label>
          </>
        )}
        {(aba === 'criados' ||
          aba === 'resolvidos' ||
          aba === 'estoque' ||
          aba === 'historias') && (
          <div className="space-y-1.5">
            <Label htmlFor="tipo">Tipo</Label>
            <Input
              id="tipo"
              className="w-48"
              placeholder="História, Incident..."
              value={tipo}
              onChange={(e) => setParams(definir(params, 'tipo', e.target.value))}
            />
          </div>
        )}
        {consulta.data && (
          <p className="ml-auto text-right text-xs text-muted-foreground">
            {numero(consulta.data.total)} linha(s)
            {consulta.data.paginas > 1
              ? ` · página ${numero(consulta.data.pagina)} de ${numero(consulta.data.paginas)}`
              : ''}
            <br />
            <a
              href={comQuery(`/csv/dados/${aba}`, filtrosCsv)}
              className="inline-flex items-center gap-1 font-medium underline"
            >
              <Download className="size-3" /> CSV
            </a>
          </p>
        )}
      </div>

      {consulta.isLoading ? (
        <Skeleton className="h-80" />
      ) : consulta.isError || !consulta.data ? (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertTitle>Não foi possível carregar os dados</AlertTitle>
          <AlertDescription>
            {consulta.error instanceof Error ? consulta.error.message : 'Erro desconhecido.'}
          </AlertDescription>
        </Alert>
      ) : (
        <Card>
          <CardContent className="pt-6">
            <Table>
              <TableHeader>
                <TableRow>
                  {consulta.data.colunas.map((c) => (
                    <TableHead
                      key={c.titulo}
                      className={c.classe === 'n' ? 'text-right' : undefined}
                    >
                      {c.titulo}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {consulta.data.linhas.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={consulta.data.colunas.length}
                      className="text-muted-foreground"
                    >
                      Nenhuma linha com esse filtro. Rode a coleta na tela Sincronização se ainda
                      não houver dados.
                    </TableCell>
                  </TableRow>
                ) : (
                  consulta.data.linhas.map((linha, i) => (
                    <TableRow key={i} className={linha.apagada ? 'opacity-60' : undefined}>
                      {linha.celulas.map((cel, j) => (
                        <TableCell
                          key={j}
                          className={
                            cel.classe === 'n'
                              ? 'text-right tabular-nums'
                              : cel.classe === 'quebra'
                                ? 'max-w-md whitespace-normal'
                                : undefined
                          }
                        >
                          {cel.link ? (
                            <a
                              href={cel.link}
                              target="_blank"
                              rel="noopener"
                              className="font-medium underline"
                            >
                              {cel.texto}
                            </a>
                          ) : (
                            cel.texto
                          )}
                          {cel.selo && (
                            <Badge variant="outline" className="ml-2">
                              {cel.selo}
                            </Badge>
                          )}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
            {consulta.data.paginas > 1 && (
              <div className="mt-4 flex items-center gap-3">
                <Button
                  variant="outline"
                  disabled={consulta.data.pagina <= 1}
                  onClick={() =>
                    setParams(definir(params, 'pagina', String(consulta.data!.pagina - 1)))
                  }
                >
                  Anterior
                </Button>
                <span className="text-sm text-muted-foreground">
                  {numero(consulta.data.pagina)} / {numero(consulta.data.paginas)}
                </span>
                <Button
                  variant="outline"
                  disabled={consulta.data.pagina >= consulta.data.paginas}
                  onClick={() =>
                    setParams(definir(params, 'pagina', String(consulta.data!.pagina + 1)))
                  }
                >
                  Próxima
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
