package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"math"
	"strings"
	"sync"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/slide"
)

// fonteTabelaPx é o tamanho do texto da tabela do dashboard:
// html { font-size: 15px } e .tabela { font-size: 0.88rem }.
const fonteTabelaPx = 0.88 * 15

const cssPrint = `
html, body { background: #fff; margin: 0; }
#print { background: #fff; display: block; width: 100%; }
.cartao { width: 100%; box-shadow: none; }
/* print: duas colunas mesmo em caixa estreita */
@media (max-width: 1100px) {
  .duas-colunas { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
`

type cartaoDados struct {
	CSS                                     template.CSS
	Selo, Titulo, Descricao, SemDados, Tier string
	Grafico                                 template.HTML
	SoGrafico                               bool
	PaletaCSS                               template.CSS
	Bloco                                   metrics.Bloco
}

var (
	cartaoOnce sync.Once
	cartaoTmpl *template.Template
	cartaoCSS  template.CSS
	cartaoErr  error
)

func prepararCartao() {
	cartaoOnce.Do(func() {
		raw, err := fs.ReadFile(staticFS, "static/app.css")
		if err != nil {
			cartaoErr = err
			return
		}
		cartaoCSS = template.CSS(string(raw) + cssPrint)
		cartaoTmpl, cartaoErr = template.New("cartao_print.html").Funcs(funcoesCartao()).ParseFS(templatesFS, "templates/cartao_print.html")
	})
}

func funcoesCartao() template.FuncMap {
	return template.FuncMap{
		"numero":    numero,
		"pct":       pct,
		"pctDireto": pctDireto,
		"horas":     horas,
		"pctDe": func(parte, total float64) string {
			if total == 0 {
				return vazio
			}
			v := parte / total
			return pct(&v)
		},
	}
}

func mensagemVazia(tier string, equipeAtiva bool) string {
	if strings.EqualFold(tier, "T2") {
		if equipeAtiva {
			return "Sem horas da Equipe neste período."
		}
		return "Sem horas: cadastre as pessoas na tela Equipe (Incluir = Sim)."
	}
	return "Sem dados neste período. Rode a coleta na tela Sincronização."
}

// htmlDoCartao monta o mesmo cartão da visão do Espaço. temTabela é falso no
// recorte só do gráfico e quando não há dados.
func htmlDoCartao(ped slide.PedidoPrint) (html string, temTabela bool, err error) {
	prepararCartao()
	if cartaoErr != nil {
		return "", false, cartaoErr
	}
	d := cartaoDados{CSS: cartaoCSS, Bloco: ped.Bloco, SoGrafico: ped.SoGrafico}
	if v, ok := espacos.VisualDaChave(ped.Bloco.Chave); ok && v.Cor != "" {
		d.PaletaCSS = template.CSS(fmt.Sprintf(
			"--chart-1:%s;--chart-2:%s;--chart-3:%s;--chart-4:%s;--chart-5:%s",
			v.Paleta[0], v.Paleta[1], v.Paleta[2], v.Paleta[3], v.Paleta[4],
		))
	}
	switch strings.ToUpper(ped.Tier) {
	case "T2":
		d.Tier = "T2"
		d.Selo, d.Titulo = "T2", "Horas por Conta de Esforço"
		d.Descricao = "Distribuição das horas da Equipe entre Evolução, Sustentação e Overhead (Recesso fica de fora)."
		d.Grafico = graficoT2(&ped.Bloco)
	case "T3":
		d.Tier = "T3"
		d.Selo, d.Titulo = "T3", "Vazão de Incidentes"
		d.Descricao = "Incidentes criados e resolvidos por mês. Estoque = abertos no fim do mês. Vazão = Resolvidos ÷ Criados × 100."
		d.Grafico = graficoT3(&ped.Bloco)
	default:
		d.Tier = "T1"
		d.Selo, d.Titulo = "T1", "Histórias vs Test Bugs"
		d.Descricao = "Histórias entregues e Test Bugs resolvidos por mês, com a razão e a quebra por fluxo."
		d.Grafico = graficoT1(&ped.Bloco)
	}
	if d.Grafico == "" {
		d.SemDados = mensagemVazia(d.Tier, ped.EquipeAtiva)
	}
	var buf bytes.Buffer
	if err := cartaoTmpl.Execute(&buf, d); err != nil {
		return "", false, err
	}
	return buf.String(), !ped.SoGrafico && d.Grafico != "", nil
}

// avisoCaixaPequena avisa quando o texto da tabela, já encaixado na caixa do
// marcador, ficaria abaixo de 11 pt. Não corta coluna nem redesenha.
func avisoCaixaPequena(marcador string, boxW, boxH, cssW, cssH, fontePx float64) string {
	fonte := fonteTabelaPt(boxW, boxH, cssW, cssH, fontePx)
	if fonte == 0 || fonte >= 11 {
		return ""
	}
	return fmt.Sprintf("%s: a caixa do marcador deixa o texto da tabela abaixo de 11 pt (%.0f pt). Aumente o retângulo no modelo.",
		strings.TrimSpace(marcador), fonte)
}

func fonteTabelaPt(boxW, boxH, cssW, cssH, fontePx float64) float64 {
	if boxW <= 0 || boxH <= 0 || cssW <= 0 || cssH <= 0 {
		return 0
	}
	if fontePx <= 0 {
		fontePx = fonteTabelaPx
	}
	escala := math.Min(boxW/cssW, boxH/cssH)
	return fontePx * escala
}
