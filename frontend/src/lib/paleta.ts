import type { CSSProperties } from 'react'

import type { Bloco, Espaco } from '@/lib/api'

/** Ordem de agrupamento das BUs (espelha internal/espacos.BUs). */
export const ORDEM_BU = [
  'conecta',
  'contabiliza',
  'credito',
  'integra',
  'processamento',
  'valida',
  'valoriza',
] as const

export type VisualEspaco = {
  bu?: string
  bu_nome?: string
  cor?: string
  paleta?: string[]
}

/** Define --chart-1..5 a partir da paleta do Espaço (ou da visão Todos). */
export function estiloPaleta(visual?: VisualEspaco | null): CSSProperties | undefined {
  const p = visual?.paleta
  if (!p || p.length < 5) return undefined
  return {
    '--chart-1': p[0],
    '--chart-2': p[1],
    '--chart-3': p[2],
    '--chart-4': p[3],
    '--chart-5': p[4],
  } as CSSProperties
}

export function visualDoBloco(bloco?: Bloco | null): VisualEspaco | undefined {
  if (!bloco) return undefined
  return {
    bu: bloco.bu,
    bu_nome: bloco.bu_nome,
    cor: bloco.cor,
    paleta: bloco.paleta,
  }
}

export type GrupoBU<T extends { bu?: string; bu_nome?: string }> = {
  bu: string
  nome: string
  cor?: string
  itens: T[]
}

/** Agrupa itens pela BU, na ordem canônica; itens sem BU vão ao final. */
export function agruparPorBU<T extends { bu?: string; bu_nome?: string; cor?: string }>(
  itens: T[],
): GrupoBU<T>[] {
  const porBU = new Map<string, GrupoBU<T>>()
  const semBU: T[] = []
  for (const item of itens) {
    const bu = item.bu?.trim()
    if (!bu) {
      semBU.push(item)
      continue
    }
    let g = porBU.get(bu)
    if (!g) {
      g = { bu, nome: item.bu_nome || bu, cor: item.cor, itens: [] }
      porBU.set(bu, g)
    }
    g.itens.push(item)
  }
  const out: GrupoBU<T>[] = []
  for (const id of ORDEM_BU) {
    const g = porBU.get(id)
    if (g) out.push(g)
  }
  const conhecidas = new Set<string>(ORDEM_BU)
  for (const [id, g] of porBU) {
    if (!conhecidas.has(id)) out.push(g)
  }
  if (semBU.length) out.push({ bu: '', nome: 'Outros', itens: semBU })
  return out
}

export function corDoEspaco(chave: string, espacos: Espaco[]): string | undefined {
  return espacos.find((e) => e.chave === chave)?.cor
}
