package coleta

import (
	"fmt"

	"metricas-coordenadores/internal/params"
)

// JQLIncidentesEstoqueLacuna monta o predicado complementar de estoque para uma lacuna.
func JQLIncidentesEstoqueLacuna(cfg params.Config, pedido Intervalo, lac Intervalo, cobertura []Intervalo) string {
	base := params.ClausulaProjetos(params.Lista(cfg["projetosIncidentes"])) +
		fmt.Sprintf("issuetype IN %s AND ", params.JQLLista(params.Lista(cfg["tiposIncidentes"])))
	status := ""
	if excl := params.Lista(cfg["statusExcluidosEstoque"]); len(excl) > 0 {
		status = " AND status NOT IN " + params.JQLLista(excl)
	}
	switch ClassificarLacunaEstoque(lac, cobertura) {
	case EstoquePrefixo:
		// resolvidos na lacuna + criados antes do fim do pedido
		return base +
			fmt.Sprintf(`resolutiondate >= "%s" AND resolutiondate < "%s" AND created < "%s"`,
				lac.InicioStr(), lac.FimExclStr(), pedido.FimExclStr()) +
			status + " ORDER BY created DESC"
	case EstoqueSufixo:
		return base +
			fmt.Sprintf(`created >= "%s" AND created < "%s" AND (resolutiondate IS EMPTY OR resolutiondate >= "%s")`,
				lac.InicioStr(), lac.FimExclStr(), pedido.InicioStr()) +
			status + " ORDER BY created DESC"
	case EstoqueBuraco:
		return base +
			fmt.Sprintf(`created >= "%s" AND created < "%s" AND resolutiondate >= "%s" AND resolutiondate < "%s"`,
				lac.InicioStr(), lac.FimExclStr(), lac.InicioStr(), lac.FimExclStr()) +
			status + " ORDER BY created DESC"
	default:
		p := PeriodoDaLacuna(pedido)
		return JQLIncidentesEstoque(cfg, p)
	}
}
