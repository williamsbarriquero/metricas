import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { PaletaEspaco } from '@/components/dashboard/paleta-espaco'
import { SecaoT1, SecaoT2, SecaoT3 } from '@/components/dashboard/secoes'
import type { Bloco } from '@/lib/api'
import { api } from '@/lib/api'

function esperarCartao(raiz: HTMLElement) {
  const limite = performance.now() + 5000
  return new Promise<void>((resolve) => {
    const passo = () => {
      const claro = !document.documentElement.classList.contains('dark')
      const grafico = raiz.querySelector('[data-slot="chart"]')
      const svg = grafico?.querySelector('svg')
      if (claro && !grafico) {
        resolve()
        return
      }
      if (claro && svg) {
        const sw = svg.getBoundingClientRect().width
        const bw = grafico!.getBoundingClientRect().width
        if (sw > 80 && bw > 80 && Math.abs(sw - bw) < 12) {
          resolve()
          return
        }
      }
      if (performance.now() > limite) {
        resolve()
        return
      }
      requestAnimationFrame(passo)
    }
    passo()
  })
}

export default function PrintCartaoPage() {
  const [params] = useSearchParams()
  const espaco = (params.get('espaco') || 'TODOS').toUpperCase()
  const tier = (params.get('tier') || 'T1').toUpperCase()
  const inicio = params.get('inicio') || undefined
  const fim = params.get('fim') || undefined
  const soGrafico = params.get('grafico') === '1'
  const [layoutOk, setLayoutOk] = useState(false)

  const consulta = useQuery({
    queryKey: ['print-cartao', inicio, fim],
    queryFn: () => api.metricas(inicio, fim),
  })

  useEffect(() => {
    const html = document.documentElement
    const tinhaEscuro = html.classList.contains('dark')
    html.classList.add('print-cartao')
    html.classList.remove('dark')
    const anteriorMargin = document.body.style.margin
    const anteriorFundo = document.body.style.background
    document.body.style.margin = '0'
    document.body.style.background = '#fff'
    const obs = new MutationObserver(() => {
      if (html.classList.contains('dark')) html.classList.remove('dark')
    })
    obs.observe(html, { attributes: true, attributeFilter: ['class'] })
    return () => {
      obs.disconnect()
      html.classList.remove('print-cartao')
      document.body.style.margin = anteriorMargin
      document.body.style.background = anteriorFundo
      if (tinhaEscuro) html.classList.add('dark')
    }
  }, [])

  const bloco =
    consulta.data == null
      ? undefined
      : espaco === 'TODOS'
        ? consulta.data.todos
        : consulta.data.espacos.find((e) => e.chave.toUpperCase() === espaco)

  const tierOk = tier === 'T1' || tier === 'T2' || tier === 'T3'
  const erro = !tierOk
    ? 'Métrica inválida no print.'
    : consulta.isError
      ? consulta.error instanceof Error
        ? consulta.error.message
        : 'Não foi possível carregar o dashboard.'
      : consulta.data && !bloco
        ? 'Espaço não encontrado no dashboard.'
        : ''

  const assinatura = `${erro}|${bloco?.chave ?? ''}|${tier}|${soGrafico ? 1 : 0}|${consulta.dataUpdatedAt}`
  const [assinaturaVista, setAssinaturaVista] = useState(assinatura)
  if (assinaturaVista !== assinatura) {
    setAssinaturaVista(assinatura)
    setLayoutOk(false)
  }

  useEffect(() => {
    if (!assinatura || erro || !bloco || !consulta.data) return () => {}
    const raiz = document.getElementById('print')
    if (!raiz) return () => {}
    let cancelado = false
    void (async () => {
      if (document.fonts?.ready) await document.fonts.ready
      await esperarCartao(raiz)
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
      if (!cancelado) setLayoutOk(true)
    })()
    return () => {
      cancelado = true
    }
  }, [assinatura, erro, bloco, consulta.data])

  const pronto = erro ? 'erro' : layoutOk ? 'ok' : ''

  return (
    <div className="bg-white text-foreground">
      <div
        id="print"
        className="w-full bg-white"
        data-print-ready={pronto || undefined}
        data-print-erro={erro || undefined}
      >
        {consulta.isLoading && (
          <p className="p-6 text-sm text-muted-foreground">Carregando o dashboard…</p>
        )}
        {erro && <p className="p-6 text-sm">{erro}</p>}
        {bloco &&
          consulta.data &&
          tierOk &&
          cartao(bloco, consulta.data.equipe_ativa, tier, soGrafico)}
      </div>
    </div>
  )
}

function cartao(bloco: Bloco, equipeAtiva: boolean, tier: string, soGrafico: boolean) {
  const secoes =
    tier === 'T2' ? (
      <SecaoT2 bloco={bloco} equipeAtiva={equipeAtiva} impressao soGrafico={soGrafico} />
    ) : tier === 'T3' ? (
      <SecaoT3 bloco={bloco} impressao soGrafico={soGrafico} />
    ) : (
      <SecaoT1 bloco={bloco} impressao soGrafico={soGrafico} />
    )
  return <PaletaEspaco bloco={bloco}>{secoes}</PaletaEspaco>
}
