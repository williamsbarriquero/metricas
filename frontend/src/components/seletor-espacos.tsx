import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import { agruparPorBU } from '@/lib/paleta'

const para = (texto: string) =>
  texto
    .split(',')
    .map((s) => s.trim().toUpperCase())
    .filter(Boolean)

/** Marca/desmarca as chaves dos Espaços; o valor é a lista separada por vírgula (como na Config). */
export function SeletorEspacos({
  value,
  onChange,
}: {
  value: string
  onChange: (v: string) => void
}) {
  const { data: espacos = [] } = useQuery({
    queryKey: ['espacos'],
    queryFn: api.espacos,
    staleTime: Infinity,
  })
  const grupos = useMemo(() => agruparPorBU(espacos), [espacos])
  const selecionadas = para(value)
  const alternar = (chave: string) => {
    const novas = selecionadas.includes(chave)
      ? selecionadas.filter((c) => c !== chave)
      : [...selecionadas, chave]
    // mantém a ordem do glossário; chaves fora dele ficam no fim
    const ordem = espacos.map((e) => e.chave)
    novas.sort((a, b) => {
      const ia = ordem.indexOf(a)
      const ib = ordem.indexOf(b)
      return (ia < 0 ? 999 : ia) - (ib < 0 ? 999 : ib)
    })
    onChange(novas.join(', '))
  }
  const extras = selecionadas.filter((c) => !espacos.some((e) => e.chave === c))

  return (
    <div className="space-y-3">
      {grupos.map((g) => (
        <div key={g.bu || g.nome} className="space-y-1.5">
          <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
            {g.cor && (
              <span
                className="inline-block size-2.5 rounded-full ring-1 ring-foreground/15"
                style={{ backgroundColor: g.cor }}
                aria-hidden
              />
            )}
            {g.nome}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {g.itens.map((e) => {
              const ativo = selecionadas.includes(e.chave)
              return (
                <button
                  type="button"
                  key={e.chave}
                  onClick={() => alternar(e.chave)}
                  title={e.nome}
                  aria-pressed={ativo}
                >
                  <Badge
                    variant={ativo ? 'default' : 'outline'}
                    className="cursor-pointer"
                    style={
                      ativo && e.cor
                        ? { backgroundColor: e.cor, borderColor: e.cor, color: '#111' }
                        : e.cor
                          ? { borderColor: e.cor }
                          : undefined
                    }
                  >
                    {e.chave}
                  </Badge>
                </button>
              )
            })}
          </div>
        </div>
      ))}
      {extras.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {extras.map((c) => (
            <button
              type="button"
              key={c}
              onClick={() => alternar(c)}
              title="Chave fora do glossário (clique para remover)"
            >
              <Badge variant="secondary" className="cursor-pointer">
                {c}
              </Badge>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
