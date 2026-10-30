import { Download, ImageDown, Loader2 } from 'lucide-react'
import { useRef, useState, type ReactNode } from 'react'
import { toast } from 'sonner'

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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { baixarPngElemento } from '@/lib/exportar-imagem'

export function SecaoCard({
  codigo,
  titulo,
  descricao,
  onCsv,
  nomeImagem,
  children,
}: {
  codigo: string
  titulo: string
  descricao: string
  onCsv?: () => void
  /** Prefixo do arquivo PNG (sem extensão). Se omitido, não oferece download de imagem. */
  nomeImagem?: string
  children: ReactNode
}) {
  const ref = useRef<HTMLDivElement>(null)
  const [exportando, setExportando] = useState(false)
  const temAcoes = Boolean(onCsv || nomeImagem)

  async function baixarImagem() {
    if (!ref.current || !nomeImagem) return
    setExportando(true)
    try {
      await baixarPngElemento(ref.current, nomeImagem)
      toast.success('Imagem baixada.')
    } catch (e) {
      toast.error('Não foi possível gerar a imagem.', {
        description: e instanceof Error ? e.message : undefined,
      })
    } finally {
      setExportando(false)
    }
  }

  return (
    <Card ref={ref}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span className="rounded-md bg-primary/10 px-1.5 py-0.5 text-xs font-semibold text-primary">
            {codigo}
          </span>
          {titulo}
        </CardTitle>
        <CardDescription>{descricao}</CardDescription>
        {temAcoes && (
          <CardAction data-export-ignore className="flex flex-wrap items-center justify-end gap-2">
            {onCsv && (
              <Button variant="outline" size="sm" onClick={onCsv} disabled={exportando}>
                <Download /> CSV
              </Button>
            )}
            {nomeImagem && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void baixarImagem()}
                disabled={exportando}
              >
                {exportando ? <Loader2 className="animate-spin" /> : <ImageDown />}
                Imagem
              </Button>
            )}
          </CardAction>
        )}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  )
}

export interface Coluna<L> {
  titulo: string
  valor: (linha: L) => string
}

/** Tabela mensal com linha de total do período. */
export function TabelaMensal<L>({
  colunas,
  meses,
  total,
  compacta,
}: {
  colunas: Coluna<L>[]
  meses: (L & { mes: string })[]
  total: L
  /** Menos padding / fonte menor — útil no T2 com muitas colunas. */
  compacta?: boolean
}) {
  const cab = compacta ? 'h-8 px-1.5 text-xs' : undefined
  const cel = compacta ? 'px-1.5 py-1.5 text-xs' : undefined
  return (
    <Table className={compacta ? 'text-xs' : undefined} containerClassName="overflow-visible">
      <TableHeader>
        <TableRow>
          <TableHead className={cab}>Mês</TableHead>
          {colunas.map((c) => (
            <TableHead key={c.titulo} className={['text-right', cab].filter(Boolean).join(' ')}>
              {c.titulo}
            </TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {meses.map((m) => (
          <TableRow key={m.mes}>
            <TableCell className={['font-medium', cel].filter(Boolean).join(' ')}>
              {m.mes}
            </TableCell>
            {colunas.map((c) => (
              <TableCell
                key={c.titulo}
                className={['text-right tabular-nums', cel].filter(Boolean).join(' ')}
              >
                {c.valor(m)}
              </TableCell>
            ))}
          </TableRow>
        ))}
        <TableRow className="bg-muted/50 font-semibold hover:bg-muted/50">
          <TableCell className={cel}>Período</TableCell>
          {colunas.map((c) => (
            <TableCell
              key={c.titulo}
              className={['text-right tabular-nums', cel].filter(Boolean).join(' ')}
            >
              {c.valor(total)}
            </TableCell>
          ))}
        </TableRow>
      </TableBody>
    </Table>
  )
}

export function SemDados({ texto = 'Sem dados neste período.' }: { texto?: string }) {
  return <p className="py-8 text-center text-sm text-muted-foreground">{texto}</p>
}
