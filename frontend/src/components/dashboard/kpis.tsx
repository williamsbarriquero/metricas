import { Activity, Bug, Clock, Gauge } from 'lucide-react'
import type { ComponentType, ReactNode } from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { Bloco } from '@/lib/api'
import { horas, numero, percentual, percentualDireto } from '@/lib/format'

function Kpi({
  titulo,
  valor,
  detalhe,
  icone: Icone,
}: {
  titulo: string
  valor: string
  detalhe: ReactNode
  icone: ComponentType<{ className?: string }>
}) {
  return (
    <Card size="sm">
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm font-medium text-muted-foreground">{titulo}</CardTitle>
        <Icone className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold tabular-nums">{valor}</p>
        <p className="mt-1 text-xs text-muted-foreground">{detalhe}</p>
      </CardContent>
    </Card>
  )
}

export function Kpis({ bloco }: { bloco: Bloco }) {
  const t1 = bloco.t1.total
  const t2 = bloco.t2.total
  const t3 = bloco.t3.total
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Kpi
        titulo="Test Bugs / Histórias"
        valor={percentual(t1.razao)}
        detalhe={`${numero(t1.test_bugs)} test bugs em ${numero(t1.historias)} histórias`}
        icone={Bug}
      />
      <Kpi
        titulo="Evolução do total de horas"
        valor={percentual(t2.pct_evolucao)}
        detalhe={`${horas(t2.evolucao)} de ${horas(t2.total)}`}
        icone={Activity}
      />
      <Kpi
        titulo="Sustentação do total de horas"
        valor={percentual(t2.pct_sustentacao)}
        detalhe={`${horas(t2.sustentacao)} · Overhead ${percentual(t2.pct_overhead)}`}
        icone={Clock}
      />
      <Kpi
        titulo="Vazão de incidentes"
        valor={percentualDireto(t3.vazao)}
        detalhe={`${numero(t3.resolvidos)} resolvidos de ${numero(t3.criados)} criados · estoque ${numero(t3.estoque)}`}
        icone={Gauge}
      />
    </div>
  )
}
