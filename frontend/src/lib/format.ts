const nf0 = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 0 })
const nf1 = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const nf2 = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

export const VAZIO = '–'

export function numero(v: number | null | undefined): string {
  return v == null ? VAZIO : nf0.format(v)
}

/** Fração (0.476) -> "47,6%". */
export function percentual(v: number | null | undefined): string {
  return v == null ? VAZIO : `${nf1.format(v * 100)}%`
}

/** Valor já em % (102.82) -> "102,82%". */
export function percentualDireto(v: number | null | undefined): string {
  return v == null ? VAZIO : `${nf2.format(v)}%`
}

/** Horas decimais (455.3333) -> "455:20:00" (mesmo formato dos marcadores de Slide). */
export function horas(v: number | null | undefined): string {
  if (v == null) return VAZIO
  const total = Math.round(v * 3600)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

/** Horas decimais curtas para eixos/cartões: "455 h". */
export function horasCurto(v: number | null | undefined): string {
  return v == null ? VAZIO : `${nf0.format(v)} h`
}

export function dataHora(iso: string | null | undefined): string {
  if (!iso) return VAZIO
  const d = new Date(iso)
  return d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}

export function data(iso: string): string {
  const [a, m, d] = iso.split('-')
  return `${d}/${m}/${a}`
}

export function duracao(inicio: string, fim: string | null): string {
  const ms = (fim ? new Date(fim).getTime() : Date.now()) - new Date(inicio).getTime()
  const s = Math.max(Math.round(ms / 1000), 0)
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}min ${s % 60}s`
}

/** "2026-07-01" -> "2026-07" (input type=month). */
export const paraMes = (iso: string) => iso.slice(0, 7)

/** "2026-09" -> "2026-09-30" (último dia do mês). */
export function fimDoMes(mes: string): string {
  const [a, m] = mes.split('-').map(Number)
  const d = new Date(a, m, 0)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export const inicioDoMes = (mes: string) => `${mes}-01`

export function baixarCsv(nome: string, linhas: (string | number | null)[][]) {
  const esc = (c: string | number | null) => {
    const t = c == null ? '' : String(c)
    return /[;"\n]/.test(t) ? `"${t.replace(/"/g, '""')}"` : t
  }
  const csv = '\ufeff' + linhas.map((l) => l.map(esc).join(';')).join('\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = nome
  a.click()
  URL.revokeObjectURL(url)
}
