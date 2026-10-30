import type { ReactNode } from 'react'

import type { Bloco } from '@/lib/api'
import { estiloPaleta, visualDoBloco } from '@/lib/paleta'
import { cn } from '@/lib/utils'

/** Aplica a paleta da BU (ou de Todos) aos gráficos filhos via --chart-1..5. */
export function PaletaEspaco({
  bloco,
  className,
  children,
}: {
  bloco?: Bloco | null
  className?: string
  children: ReactNode
}) {
  return (
    <div className={cn(className)} style={estiloPaleta(visualDoBloco(bloco))}>
      {children}
    </div>
  )
}

export function TituloEspaco({
  bloco,
  className,
  tituloClassName,
}: {
  bloco: Bloco
  className?: string
  /** Classes do título (padrão: text-lg no dashboard; text-base na página Métricas). */
  tituloClassName?: string
}) {
  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      {bloco.cor && (
        <span
          className="inline-block size-3 shrink-0 rounded-full ring-1 ring-foreground/15"
          style={{ backgroundColor: bloco.cor }}
          title={bloco.bu_nome || bloco.cor}
          aria-hidden
        />
      )}
      <h2 className={cn('text-lg font-semibold', tituloClassName)}>{bloco.rotulo}</h2>
      {bloco.bu_nome && (
        <span className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground">
          {bloco.bu_nome}
        </span>
      )}
    </div>
  )
}
