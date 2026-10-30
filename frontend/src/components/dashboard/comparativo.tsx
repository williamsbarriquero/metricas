import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, Download } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Bar, BarChart, CartesianGrid, Cell, XAxis, YAxis } from 'recharts'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { Bloco } from '@/lib/api'
import { baixarCsv, horas, numero, percentual, percentualDireto } from '@/lib/format'

type Linha = {
  chave: string
  nome: string
  historias: number
  test_bugs: number
  razao: number | null
  horas_total: number
  pct_evolucao: number | null
  pct_sustentacao: number | null
  pct_overhead: number | null
  criados: number
  resolvidos: number
  vazao: number | null
}

const METRICAS = [
  { id: 'razao', rotulo: 'Test Bugs / Histórias', fator: 100, unidade: '%' },
  { id: 'pct_evolucao', rotulo: '% Evolução', fator: 100, unidade: '%' },
  { id: 'pct_sustentacao', rotulo: '% Sustentação', fator: 100, unidade: '%' },
  { id: 'vazao', rotulo: 'Vazão de incidentes', fator: 1, unidade: '%' },
] as const

const cabecalho = (titulo: string) =>
  function Cabecalho({
    column,
  }: {
    column: { getIsSorted: () => false | 'asc' | 'desc'; toggleSorting: (desc?: boolean) => void }
  }) {
    const s = column.getIsSorted()
    return (
      <Button
        variant="ghost"
        size="xs"
        className="-mx-2"
        onClick={() => column.toggleSorting(s === 'asc')}
      >
        {titulo}
        {s === 'asc' ? (
          <ArrowUp />
        ) : s === 'desc' ? (
          <ArrowDown />
        ) : (
          <ArrowUpDown className="opacity-50" />
        )}
      </Button>
    )
  }

const colunas: ColumnDef<Linha>[] = [
  {
    accessorKey: 'nome',
    header: cabecalho('Espaço'),
    cell: ({ row }) => (
      <div>
        <p className="font-medium">{row.original.nome}</p>
        <p className="text-xs text-muted-foreground">{row.original.chave}</p>
      </div>
    ),
  },
  {
    accessorKey: 'historias',
    header: cabecalho('Histórias'),
    cell: ({ getValue }) => numero(getValue<number>()),
  },
  {
    accessorKey: 'test_bugs',
    header: cabecalho('Test Bugs'),
    cell: ({ getValue }) => numero(getValue<number>()),
  },
  {
    accessorKey: 'razao',
    header: cabecalho('TB / Hist.'),
    cell: ({ getValue }) => percentual(getValue<number | null>()),
    sortUndefined: 'last',
  },
  {
    accessorKey: 'horas_total',
    header: cabecalho('Horas'),
    cell: ({ getValue }) => horas(getValue<number>()),
  },
  {
    accessorKey: 'pct_evolucao',
    header: cabecalho('% Evol.'),
    cell: ({ getValue }) => percentual(getValue<number | null>()),
    sortUndefined: 'last',
  },
  {
    accessorKey: 'pct_sustentacao',
    header: cabecalho('% Sust.'),
    cell: ({ getValue }) => percentual(getValue<number | null>()),
    sortUndefined: 'last',
  },
  {
    accessorKey: 'pct_overhead',
    header: cabecalho('% Over.'),
    cell: ({ getValue }) => percentual(getValue<number | null>()),
    sortUndefined: 'last',
  },
  {
    accessorKey: 'criados',
    header: cabecalho('Criados'),
    cell: ({ getValue }) => numero(getValue<number>()),
  },
  {
    accessorKey: 'resolvidos',
    header: cabecalho('Resolvidos'),
    cell: ({ getValue }) => numero(getValue<number>()),
  },
  {
    accessorKey: 'vazao',
    header: cabecalho('Vazão'),
    cell: ({ getValue }) => percentualDireto(getValue<number | null>()),
    sortUndefined: 'last',
  },
]

function paraLinha(b: Bloco): Linha {
  return {
    chave: b.chave,
    nome: b.nome,
    historias: b.t1.total.historias,
    test_bugs: b.t1.total.test_bugs,
    razao: b.t1.total.razao,
    horas_total: b.t2.total.total,
    pct_evolucao: b.t2.total.pct_evolucao,
    pct_sustentacao: b.t2.total.pct_sustentacao,
    pct_overhead: b.t2.total.pct_overhead,
    criados: b.t3.total.criados,
    resolvidos: b.t3.total.resolvidos,
    vazao: b.t3.total.vazao,
  }
}

export function Comparativo({ blocos, todos }: { blocos: Bloco[]; todos: Bloco | null }) {
  const [semDados, setSemDados] = useState(false)
  const [metrica, setMetrica] = useState<(typeof METRICAS)[number]['id']>('razao')
  const [ordem, setOrdem] = useState<SortingState>([])

  const linhas = useMemo(
    () => blocos.filter((b) => semDados || b.tem_dados).map(paraLinha),
    [blocos, semDados],
  )
  const tabela = useReactTable({
    data: linhas,
    columns: colunas,
    state: { sorting: ordem },
    onSortingChange: setOrdem,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  const m = METRICAS.find((x) => x.id === metrica)!
  const corPorChave = useMemo(() => {
    const cores = new Map<string, string>()
    for (const b of blocos) {
      if (b.cor) cores.set(b.chave, b.cor)
    }
    return cores
  }, [blocos])
  const dadosGrafico = linhas
    .filter((l) => l[m.id] != null)
    .map((l) => ({
      chave: l.chave,
      nome: l.nome,
      valor: Number(((l[m.id] as number) * m.fator).toFixed(2)),
      cor: corPorChave.get(l.chave) ?? 'var(--chart-1)',
    }))
    .sort((a, b) => b.valor - a.valor)
  const config = { valor: { label: m.rotulo, color: 'var(--chart-1)' } } satisfies ChartConfig

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Comparativo entre Espaços</CardTitle>
          <CardDescription>
            Totais do período para cada Espaço. Clique nos títulos para ordenar.
          </CardDescription>
          <CardAction className="flex items-center gap-2">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Switch
                id="mostrar-sem-dados"
                aria-labelledby="mostrar-sem-dados-rotulo"
                checked={semDados}
                onCheckedChange={setSemDados}
              />
              <span id="mostrar-sem-dados-rotulo">Mostrar Espaços sem dados</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                baixarCsv('comparativo-espacos.csv', [
                  [
                    'Chave',
                    'Espaço',
                    'Histórias',
                    'Test Bugs',
                    'TB/Histórias',
                    'Horas (sem Recesso)',
                    '% Evolução',
                    '% Sustentação',
                    '% Overhead',
                    'Criados',
                    'Resolvidos',
                    'Vazão (%)',
                  ],
                  ...[...linhas, ...(todos ? [paraLinha(todos)] : [])].map((l) => [
                    l.chave,
                    l.nome,
                    l.historias,
                    l.test_bugs,
                    l.razao,
                    l.horas_total,
                    l.pct_evolucao,
                    l.pct_sustentacao,
                    l.pct_overhead,
                    l.criados,
                    l.resolvidos,
                    l.vazao,
                  ]),
                ])
              }
            >
              <Download /> CSV
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <Table>
            <TableHeader>
              {tabela.getHeaderGroups().map((g) => (
                <TableRow key={g.id}>
                  {g.headers.map((h) => (
                    <TableHead key={h.id}>
                      {flexRender(h.column.columnDef.header, h.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {tabela.getRowModel().rows.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={colunas.length}
                    className="py-8 text-center text-muted-foreground"
                  >
                    Nenhum Espaço com dados no período.
                  </TableCell>
                </TableRow>
              )}
              {tabela.getRowModel().rows.map((r) => (
                <TableRow key={r.id}>
                  {r.getVisibleCells().map((c) => (
                    <TableCell key={c.id} className="tabular-nums">
                      {flexRender(c.column.columnDef.cell, c.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
              {todos && (
                <TableRow className="bg-muted/50 font-semibold hover:bg-muted/50">
                  {(() => {
                    const l = paraLinha(todos)
                    return (
                      <>
                        <TableCell>Todos os Espaços</TableCell>
                        <TableCell className="tabular-nums">{numero(l.historias)}</TableCell>
                        <TableCell className="tabular-nums">{numero(l.test_bugs)}</TableCell>
                        <TableCell className="tabular-nums">{percentual(l.razao)}</TableCell>
                        <TableCell className="tabular-nums">{horas(l.horas_total)}</TableCell>
                        <TableCell className="tabular-nums">{percentual(l.pct_evolucao)}</TableCell>
                        <TableCell className="tabular-nums">
                          {percentual(l.pct_sustentacao)}
                        </TableCell>
                        <TableCell className="tabular-nums">{percentual(l.pct_overhead)}</TableCell>
                        <TableCell className="tabular-nums">{numero(l.criados)}</TableCell>
                        <TableCell className="tabular-nums">{numero(l.resolvidos)}</TableCell>
                        <TableCell className="tabular-nums">{percentualDireto(l.vazao)}</TableCell>
                      </>
                    )
                  })()}
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Ranking por indicador</CardTitle>
          <CardDescription>{m.rotulo} no período, por Espaço.</CardDescription>
          <CardAction>
            <Select
              value={metrica}
              onValueChange={(v) => setMetrica(v as typeof metrica)}
              items={METRICAS.map((x) => ({ value: x.id, label: x.rotulo }))}
            >
              <SelectTrigger className="w-52">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {METRICAS.map((x) => (
                  <SelectItem key={x.id} value={x.id}>
                    {x.rotulo}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </CardAction>
        </CardHeader>
        <CardContent>
          {dadosGrafico.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              Sem dados para este indicador.
            </p>
          ) : (
            <ChartContainer
              config={config}
              className="w-full"
              style={{ height: Math.max(dadosGrafico.length * 34 + 40, 160) }}
            >
              <BarChart data={dadosGrafico} layout="vertical" margin={{ left: 8, right: 24 }}>
                <CartesianGrid horizontal={false} />
                <YAxis
                  dataKey="chave"
                  type="category"
                  tickLine={false}
                  axisLine={false}
                  width={56}
                />
                <XAxis
                  type="number"
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v) => `${v}${m.unidade}`}
                />
                <ChartTooltip
                  content={
                    <ChartTooltipContent
                      hideLabel
                      formatter={(valor, _n, item) => (
                        <div className="flex w-full justify-between gap-4">
                          <span className="text-muted-foreground">
                            {(item.payload as { nome: string }).nome}
                          </span>
                          <span className="font-mono font-medium">
                            {Number(valor).toLocaleString('pt-BR')}
                            {m.unidade}
                          </span>
                        </div>
                      )}
                    />
                  }
                />
                <Bar dataKey="valor" fill="var(--color-valor)" radius={4}>
                  {dadosGrafico.map((d) => (
                    <Cell key={d.chave} fill={d.cor} />
                  ))}
                </Bar>
              </BarChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
