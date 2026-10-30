import {
  Bar,
  BarChart,
  CartesianGrid,
  LabelList,
  Line,
  LineChart,
  XAxis,
  YAxis,
  usePlotArea,
  useXAxisScale,
  useYAxisScale,
} from 'recharts'

import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart'
import type { Bloco, LinhaT1, LinhaT2, LinhaT3, Mensal } from '@/lib/api'
import { baixarCsv, horas, numero, percentual, percentualDireto } from '@/lib/format'

import { type Coluna, SecaoCard, SemDados, TabelaMensal } from './comum'

const slug = (b: Bloco) => b.chave.toLowerCase()

function classeGrade(impressao?: boolean) {
  // No painel: gráfico em cima e tabela embaixo (largura total, sem scroll lateral).
  // Na impressão do slide: lado a lado.
  return impressao ? 'grid grid-cols-2 gap-6' : 'grid gap-6'
}

type CaixaRotulo = {
  x?: unknown
  y?: unknown
  width?: unknown
  height?: unknown
  value?: unknown
  viewBox?: unknown
}

function medida(v: unknown) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function caixaDoRotulo(props: CaixaRotulo) {
  const vb = props.viewBox
  const caixa = vb && typeof vb === 'object' ? (vb as Record<string, unknown>) : undefined
  return {
    x: medida(props.x ?? caixa?.x),
    y: medida(props.y ?? caixa?.y),
    width: medida(props.width ?? caixa?.width),
    height: medida(props.height ?? caixa?.height),
  }
}

/** Número acima da barra. Some se a barra for estreita demais para o texto. */
function RotuloBarra(props: CaixaRotulo) {
  if (props.value == null || props.value === '' || Array.isArray(props.value)) return <g />
  const texto = numero(Number(props.value))
  const { x, y, width } = caixaDoRotulo(props)
  if (width > 0 && texto.length * 6.6 > width + 10) return <g />
  return (
    <text
      x={x + width / 2}
      y={y - 6}
      textAnchor="middle"
      fill="var(--foreground)"
      stroke="var(--background)"
      strokeWidth={3}
      paintOrder="stroke"
      fontSize={11}
      fontWeight={600}
      style={{ pointerEvents: 'none' }}
    >
      {texto}
    </text>
  )
}

/** Percentual no centro do trecho. Usa uma casa quando cabe; senão, inteiro. */
function RotuloPercentual(props: CaixaRotulo) {
  if (props.value == null || props.value === '' || Array.isArray(props.value)) return <g />
  const fracao = Number(props.value)
  if (!Number.isFinite(fracao) || fracao <= 0) return <g />
  const { x, y, width, height } = caixaDoRotulo(props)
  if (height < 12) return <g />
  const cheio = percentual(fracao)
  const curto = `${Math.round(fracao * 100)}%`
  const texto = width >= cheio.length * 6.4 + 8 ? cheio : curto
  if (width < texto.length * 6.2 + 4) return <g />
  return (
    <text
      x={x + width / 2}
      y={y + height / 2}
      textAnchor="middle"
      dominantBaseline="central"
      fill="oklch(0.2 0 0)"
      fontSize={11}
      fontWeight={600}
      style={{ pointerEvents: 'none' }}
    >
      {texto}
    </text>
  )
}

const SERIES_T3 = [
  { chave: 'criados', cor: 'var(--color-criados)' },
  { chave: 'resolvidos', cor: 'var(--color-resolvidos)' },
  { chave: 'estoque', cor: 'var(--color-estoque)' },
] as const

const ALTURA_ROTULO = 13

type RotuloColocado = { id: string; x: number; y: number; texto: string; cor: string }

/** Criados e resolvidos ficam lado a lado, acima das duas linhas. Estoque fica junto da própria linha. */
function RotulosT3({ meses }: { meses: Mensal<LinhaT3>[] }) {
  const escalaX = useXAxisScale()
  const escalaY = useYAxisScale()
  const area = usePlotArea()
  if (!escalaX || !escalaY || !area || area.height < 20) return null

  const topo = area.y + 8
  const base = area.y + area.height - 8
  const colocados: RotuloColocado[] = []
  const limitarX = (x: number, metade: number) =>
    Math.min(Math.max(x, area.x + metade + 1), area.x + area.width - metade - 1)

  for (const mes of meses) {
    const xMes = escalaX(mes.mes, { position: 'middle' })
    if (xMes == null) continue
    const pontos = SERIES_T3.flatMap((serie) => {
      const y = escalaY(mes[serie.chave])
      if (y == null) return []
      return [
        {
          id: `${mes.mes}-${serie.chave}`,
          chave: serie.chave,
          x: xMes,
          y,
          texto: numero(mes[serie.chave]),
          cor: serie.cor,
        },
      ]
    })
    const par = pontos.filter((p) => p.chave !== 'estoque')
    const estoque = pontos.find((p) => p.chave === 'estoque')
    let yPar = topo
    if (par.length > 0) {
      const larguras = par.map((p) => Math.max(p.texto.length * 6.6, 10))
      const total = larguras.reduce((s, w) => s + w, 0) + 10 * (par.length - 1)
      let cursor = limitarX(xMes - total / 2 + larguras[0] / 2, larguras[0] / 2) - larguras[0] / 2
      if (cursor + total > area.x + area.width - 1) cursor = area.x + area.width - 1 - total
      if (cursor < area.x + 1) cursor = area.x + 1
      yPar = Math.max(topo, Math.min(...par.map((p) => p.y)) - 18)
      par.forEach((p, i) => {
        colocados.push({
          id: p.id,
          x: cursor + larguras[i] / 2,
          y: yPar,
          texto: p.texto,
          cor: p.cor,
        })
        cursor += larguras[i] + 10
      })
    }
    if (estoque) {
      const metade = estoque.texto.length * 3.3
      let y = estoque.y - 14
      if (y < topo || Math.abs(y - yPar) < 22) y = estoque.y + 16
      if (y > base) y = Math.min(base, yPar + ALTURA_ROTULO + 4)
      colocados.push({
        id: estoque.id,
        x: limitarX(estoque.x, metade),
        y,
        texto: estoque.texto,
        cor: estoque.cor,
      })
    }
  }

  return (
    <g>
      {colocados.map((r) => (
        <text
          key={r.id}
          x={r.x}
          y={r.y}
          textAnchor="middle"
          dominantBaseline="central"
          fill={r.cor}
          stroke="var(--background)"
          strokeWidth={3}
          paintOrder="stroke"
          fontSize={11}
          fontWeight={600}
          style={{ pointerEvents: 'none' }}
        >
          {r.texto}
        </text>
      ))}
    </g>
  )
}

// ---------------------------------------------------------------------------
// T1 - Histórias vs Test Bugs
// ---------------------------------------------------------------------------

const COLUNAS_T1: Coluna<LinhaT1>[] = [
  { titulo: 'Histórias', valor: (l) => numero(l.historias) },
  { titulo: 'Test Bugs', valor: (l) => numero(l.test_bugs) },
  { titulo: 'TB / H', valor: (l) => percentual(l.razao) },
  { titulo: 'Principal', valor: (l) => numero(l.fluxo_principal) },
  { titulo: 'Alternativo', valor: (l) => numero(l.fluxo_alternativo) },
  { titulo: 'Exceção', valor: (l) => numero(l.fluxo_excecao) },
]

export function SecaoT1({
  bloco,
  impressao,
  soGrafico,
}: {
  bloco: Bloco
  impressao?: boolean
  soGrafico?: boolean
}) {
  const { meses, total } = bloco.t1
  const config = {
    historias: { label: 'Histórias', color: 'var(--chart-1)' },
    test_bugs: { label: 'Test Bugs', color: 'var(--chart-4)' },
  } satisfies ChartConfig
  const vazio = total.historias === 0 && total.test_bugs === 0
  const grafico = (
    <ChartContainer config={config} className="h-64 w-full">
      <BarChart data={meses} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="mes" tickLine={false} axisLine={false} />
        <YAxis
          tickLine={false}
          axisLine={false}
          allowDecimals={false}
          width={32}
          padding={{ top: 18 }}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} />
        <Bar
          dataKey="historias"
          fill="var(--color-historias)"
          radius={4}
          isAnimationActive={!impressao}
        >
          <LabelList content={RotuloBarra} />
        </Bar>
        <Bar
          dataKey="test_bugs"
          fill="var(--color-test_bugs)"
          radius={4}
          isAnimationActive={!impressao}
        >
          <LabelList content={RotuloBarra} />
        </Bar>
      </BarChart>
    </ChartContainer>
  )

  return (
    <SecaoCard
      codigo="T1"
      titulo="Histórias vs Test Bugs"
      descricao="Histórias entregues e Test Bugs resolvidos por mês, com a razão e a quebra por fluxo."
      nomeImagem={impressao ? undefined : `t1-historias-test-bugs-${slug(bloco)}`}
      onCsv={
        impressao
          ? undefined
          : () =>
              baixarCsv(`t1-historias-test-bugs-${slug(bloco)}.csv`, [
                [
                  'Mês',
                  'Histórias',
                  'Test Bugs',
                  'Test Bugs / Histórias',
                  'Fluxo Principal',
                  'Fluxo Alternativo',
                  'Fluxo Exceção',
                ],
                ...[...meses, { mes: 'Período', ...total }].map((m) => [
                  m.mes,
                  m.historias,
                  m.test_bugs,
                  m.razao,
                  m.fluxo_principal,
                  m.fluxo_alternativo,
                  m.fluxo_excecao,
                ]),
              ])
      }
    >
      {vazio ? (
        <SemDados />
      ) : soGrafico ? (
        grafico
      ) : (
        <div className={classeGrade(impressao)}>
          {grafico}
          <TabelaMensal colunas={COLUNAS_T1} meses={meses} total={total} />
        </div>
      )}
    </SecaoCard>
  )
}

// ---------------------------------------------------------------------------
// T2 - Horas por Conta de Esforço
// ---------------------------------------------------------------------------

const COLUNAS_T2: Coluna<LinhaT2>[] = [
  { titulo: 'Sust.', valor: (l) => horas(l.sustentacao) },
  { titulo: 'Evol.', valor: (l) => horas(l.evolucao) },
  { titulo: 'Over.', valor: (l) => horas(l.overhead) },
  { titulo: 'S/conta', valor: (l) => horas(l.sem_conta) },
  { titulo: 'Total', valor: (l) => horas(l.total) },
  { titulo: '% Sust.', valor: (l) => percentual(l.pct_sustentacao) },
  { titulo: '% Evol.', valor: (l) => percentual(l.pct_evolucao) },
  { titulo: '% Over.', valor: (l) => percentual(l.pct_overhead) },
]

export function SecaoT2({
  bloco,
  equipeAtiva,
  impressao,
  soGrafico,
}: {
  bloco: Bloco
  equipeAtiva: boolean
  impressao?: boolean
  soGrafico?: boolean
}) {
  const { meses, total } = bloco.t2
  const config = {
    evolucao: { label: 'Evolução', color: 'var(--chart-1)' },
    sustentacao: { label: 'Sustentação', color: 'var(--chart-2)' },
    overhead: { label: 'Overhead', color: 'var(--chart-3)' },
  } satisfies ChartConfig
  const vazio = total.total === 0
  const grafico = (
    <ChartContainer config={config} className="h-64 w-full">
      <BarChart data={meses} layout="vertical" stackOffset="expand">
        <CartesianGrid horizontal={false} />
        <YAxis dataKey="mes" type="category" tickLine={false} axisLine={false} width={64} />
        <XAxis
          type="number"
          tickLine={false}
          axisLine={false}
          tickFormatter={(v) => `${Math.round(v * 100)}%`}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(valor, nome) => (
                <div className="flex w-full justify-between gap-4">
                  <span className="text-muted-foreground">
                    {config[nome as keyof typeof config]?.label}
                  </span>
                  <span className="font-mono font-medium">{horas(Number(valor))}</span>
                </div>
              )}
            />
          }
        />
        <ChartLegend content={<ChartLegendContent />} />
        <Bar
          dataKey="evolucao"
          stackId="h"
          fill="var(--color-evolucao)"
          isAnimationActive={!impressao}
        >
          <LabelList dataKey="pct_evolucao" content={RotuloPercentual} />
        </Bar>
        <Bar
          dataKey="sustentacao"
          stackId="h"
          fill="var(--color-sustentacao)"
          isAnimationActive={!impressao}
        >
          <LabelList dataKey="pct_sustentacao" content={RotuloPercentual} />
        </Bar>
        <Bar
          dataKey="overhead"
          stackId="h"
          fill="var(--color-overhead)"
          radius={[0, 4, 4, 0]}
          isAnimationActive={!impressao}
        >
          <LabelList dataKey="pct_overhead" content={RotuloPercentual} />
        </Bar>
      </BarChart>
    </ChartContainer>
  )

  return (
    <SecaoCard
      codigo="T2"
      titulo="Horas por Conta de Esforço"
      descricao="Distribuição das horas da Equipe entre Evolução, Sustentação e Overhead (Recesso fica de fora)."
      nomeImagem={impressao ? undefined : `t2-horas-${slug(bloco)}`}
      onCsv={
        impressao
          ? undefined
          : () =>
              baixarCsv(`t2-horas-${slug(bloco)}.csv`, [
                [
                  'Mês',
                  'Sustentação (h)',
                  'Evolução (h)',
                  'Overhead (h)',
                  'Sem conta (h)',
                  'Recesso (h)',
                  'Total sem Recesso (h)',
                  '% Sustentação',
                  '% Evolução',
                  '% Overhead',
                ],
                ...[...meses, { mes: 'Período', ...total }].map((m) => [
                  m.mes,
                  m.sustentacao,
                  m.evolucao,
                  m.overhead,
                  m.sem_conta,
                  m.recesso,
                  m.total,
                  m.pct_sustentacao,
                  m.pct_evolucao,
                  m.pct_overhead,
                ]),
              ])
      }
    >
      {vazio ? (
        <SemDados
          texto={
            equipeAtiva
              ? 'Sem horas da Equipe neste período.'
              : 'Sem horas: cadastre as pessoas na tela Equipe (Incluir = Sim).'
          }
        />
      ) : soGrafico ? (
        grafico
      ) : (
        <div className={classeGrade(impressao)}>
          {grafico}
          <TabelaMensal colunas={COLUNAS_T2} meses={meses} total={total} compacta />
        </div>
      )}
    </SecaoCard>
  )
}

// ---------------------------------------------------------------------------
// T3 - Vazão de incidentes
// ---------------------------------------------------------------------------

const COLUNAS_T3: Coluna<LinhaT3>[] = [
  { titulo: 'Criados', valor: (l) => numero(l.criados) },
  { titulo: 'Resolvidos', valor: (l) => numero(l.resolvidos) },
  { titulo: 'Estoque', valor: (l) => numero(l.estoque) },
  { titulo: 'Vazão', valor: (l) => percentualDireto(l.vazao) },
]

export function SecaoT3({
  bloco,
  impressao,
  soGrafico,
}: {
  bloco: Bloco
  impressao?: boolean
  soGrafico?: boolean
}) {
  const { meses, total } = bloco.t3
  const config = {
    criados: { label: 'Criados', color: 'var(--chart-4)' },
    resolvidos: { label: 'Resolvidos', color: 'var(--chart-5)' },
    estoque: { label: 'Estoque', color: 'var(--chart-3)' },
  } satisfies ChartConfig
  const vazio = total.criados === 0 && total.resolvidos === 0 && total.estoque === 0
  const grafico = (
    <ChartContainer config={config} className="h-72 w-full">
      <LineChart data={meses} margin={{ left: 4, right: 16, top: 8, bottom: 4 }}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="mes" tickLine={false} axisLine={false} />
        <YAxis
          tickLine={false}
          axisLine={false}
          allowDecimals={false}
          width={36}
          padding={{ top: 22 }}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} />
        <Line
          dataKey="criados"
          type="monotone"
          stroke="var(--color-criados)"
          strokeWidth={2}
          dot
          isAnimationActive={!impressao}
        />
        <Line
          dataKey="resolvidos"
          type="monotone"
          stroke="var(--color-resolvidos)"
          strokeWidth={2}
          dot
          isAnimationActive={!impressao}
        />
        <Line
          dataKey="estoque"
          type="monotone"
          stroke="var(--color-estoque)"
          strokeWidth={2}
          dot
          isAnimationActive={!impressao}
        />
        <RotulosT3 meses={meses} />
      </LineChart>
    </ChartContainer>
  )

  return (
    <SecaoCard
      codigo="T3"
      titulo="Vazão de Incidentes"
      descricao="Incidentes criados, resolvidos e estoque no fim do mês. Vazão = Resolvidos ÷ Criados × 100. O total do estoque é o do último mês."
      nomeImagem={impressao ? undefined : `t3-vazao-${slug(bloco)}`}
      onCsv={
        impressao
          ? undefined
          : () =>
              baixarCsv(`t3-vazao-${slug(bloco)}.csv`, [
                ['Mês', 'Criados', 'Resolvidos', 'Estoque', 'Vazão (%)'],
                ...[...meses, { mes: 'Período', ...total }].map((m) => [
                  m.mes,
                  m.criados,
                  m.resolvidos,
                  m.estoque,
                  m.vazao,
                ]),
              ])
      }
    >
      {vazio ? (
        <SemDados />
      ) : soGrafico ? (
        grafico
      ) : (
        <div className={classeGrade(impressao)}>
          {grafico}
          <TabelaMensal colunas={COLUNAS_T3} meses={meses} total={total} />
        </div>
      )}
    </SecaoCard>
  )
}
