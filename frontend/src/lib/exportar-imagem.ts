import { toPng } from 'html-to-image'

export type TemaImagem = 'claro' | 'escuro'

export function temaAtual(): TemaImagem {
  return document.documentElement.classList.contains('dark') ? 'escuro' : 'claro'
}

function esperarFrames(n = 2) {
  return new Promise<void>((resolve) => {
    const passo = (restante: number) => {
      if (restante <= 0) resolve()
      else requestAnimationFrame(() => passo(restante - 1))
    }
    passo(n)
  })
}

function mostrarOverlayExportacao() {
  const el = document.createElement('div')
  el.setAttribute('data-export-ignore', '')
  el.setAttribute('aria-live', 'polite')
  el.style.cssText =
    'position:fixed;inset:0;z-index:2147483646;display:flex;align-items:center;justify-content:center;' +
    'background:color-mix(in oklch, canvas 82%, transparent);backdrop-filter:blur(2px);' +
    'font:500 14px/1.4 system-ui,sans-serif;color:CanvasText'
  el.textContent = 'Gerando imagem…'
  document.body.appendChild(el)
  return () => el.remove()
}

type EstiloOverflow = {
  el: HTMLElement
  overflow: string
  overflowX: string
  overflowY: string
}

/** Libera overflows para a captura incluir o conteúdo inteiro, sem barras de rolagem. */
function liberarOverflow(raiz: HTMLElement): () => void {
  const alvos: EstiloOverflow[] = []
  const nos = [raiz, ...Array.from(raiz.querySelectorAll<HTMLElement>('*'))]
  for (const el of nos) {
    const computed = getComputedStyle(el)
    if (
      computed.overflow === 'visible' &&
      computed.overflowX === 'visible' &&
      computed.overflowY === 'visible'
    ) {
      continue
    }
    alvos.push({
      el,
      overflow: el.style.overflow,
      overflowX: el.style.overflowX,
      overflowY: el.style.overflowY,
    })
    el.style.overflow = 'visible'
    el.style.overflowX = 'visible'
    el.style.overflowY = 'visible'
  }
  return () => {
    for (const a of alvos) {
      a.el.style.overflow = a.overflow
      a.el.style.overflowX = a.overflowX
      a.el.style.overflowY = a.overflowY
    }
  }
}

/** Tira foco/hover do gráfico para o PNG não sair com tooltip. */
function limparInteracao(raiz: HTMLElement) {
  if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
  raiz.dispatchEvent(new MouseEvent('pointerleave', { bubbles: true }))
  raiz.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }))
  for (const no of raiz.querySelectorAll('.recharts-tooltip-wrapper')) {
    ;(no as HTMLElement).style.visibility = 'hidden'
    ;(no as HTMLElement).style.opacity = '0'
  }
}

/** Captura o cartão no tema atual da tela, sem barras nem tooltip de hover. */
export async function baixarPngElemento(el: HTMLElement, nome?: string) {
  const tema = temaAtual()
  const arquivo = (nome ?? 'cartao').endsWith('.png')
    ? (nome ?? 'cartao')
    : `${nome ?? 'cartao'}-${tema}.png`

  const html = document.documentElement
  const fecharOverlay = mostrarOverlayExportacao()
  html.classList.add('exportando-imagem')
  el.classList.add('exportando-cartao')
  limparInteracao(el)
  const restaurarOverflow = liberarOverflow(el)

  try {
    if (document.fonts?.ready) await document.fonts.ready
    await esperarFrames(2)
    await new Promise((r) => setTimeout(r, 80))
    limparInteracao(el)

    const bg = getComputedStyle(el).backgroundColor
    const fundo =
      bg && bg !== 'rgba(0, 0, 0, 0)' && bg !== 'transparent'
        ? bg
        : tema === 'escuro'
          ? '#1a1a1a'
          : '#ffffff'

    const dataUrl = await toPng(el, {
      pixelRatio: Math.max(3, window.devicePixelRatio || 1),
      cacheBust: true,
      backgroundColor: fundo,
      width: Math.ceil(el.scrollWidth),
      height: Math.ceil(el.scrollHeight),
      style: { overflow: 'visible' },
      filter: (no) => {
        if (!(no instanceof HTMLElement)) return true
        if (no.closest('[data-export-ignore]')) return false
        if (no.classList.contains('recharts-tooltip-wrapper')) return false
        if (no.classList.contains('recharts-active-bar')) return false
        if (no.classList.contains('recharts-active-dot')) return false
        if (no.classList.contains('recharts-tooltip-cursor')) return false
        return true
      },
    })

    const a = document.createElement('a')
    a.href = dataUrl
    a.download = arquivo
    a.click()
  } finally {
    restaurarOverflow()
    el.classList.remove('exportando-cartao')
    html.classList.remove('exportando-imagem')
    fecharOverlay()
  }
}
