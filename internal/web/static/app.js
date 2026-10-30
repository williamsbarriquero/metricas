// Métricas RPE — comportamentos mínimos: tema, menu móvel, filtros que se aplicam sozinhos,
// confirmação de exclusão e acompanhamento da coleta. O resto é HTML renderizado no servidor.
(function () {
  'use strict';

  var raiz = document.documentElement;

  // Tema claro/escuro
  document.querySelectorAll('[data-alternar-tema]').forEach(function (b) {
    b.addEventListener('click', function () {
      var novo = raiz.dataset.tema === 'escuro' ? 'claro' : 'escuro';
      raiz.dataset.tema = novo;
      try { localStorage.setItem('tema', novo); } catch (e) { /* modo privativo */ }
    });
  });

  // Menu lateral no celular
  var lateral = document.getElementById('lateral');
  document.querySelectorAll('[data-alternar-menu]').forEach(function (b) {
    b.addEventListener('click', function () { lateral.classList.toggle('aberta'); });
  });
  document.addEventListener('click', function (e) {
    if (lateral && lateral.classList.contains('aberta') && !lateral.contains(e.target) && !e.target.closest('[data-alternar-menu]')) {
      lateral.classList.remove('aberta');
    }
  });

  // Formulários de filtro: envia ao mudar um campo
  document.querySelectorAll('form[data-auto-submit]').forEach(function (f) {
    f.addEventListener('change', function () {
      if (typeof f.requestSubmit === 'function') f.requestSubmit(); else f.submit();
    });
  });

  // Confirmação antes de ações destrutivas
  document.querySelectorAll('form[data-confirm]').forEach(function (f) {
    f.addEventListener('submit', function (e) {
      if (!window.confirm(f.dataset.confirm)) e.preventDefault();
    });
  });

  // Marcadores: clique copia o texto para a área de transferência
  document.addEventListener('click', function (e) {
    var el = e.target.closest('[data-copiar]');
    if (!el || !navigator.clipboard) return;
    navigator.clipboard.writeText(el.textContent.trim()).then(function () {
      el.classList.add('copiado');
      setTimeout(function () { el.classList.remove('copiado'); }, 900);
    });
  });

  // Geração do Slide: atualiza a barra e recarrega a página quando terminar
  if (document.querySelector('[data-slide-poll]')) {
    var consultarSlide = function () {
      fetch('/api/slide/status', { headers: { Accept: 'application/json' } })
        .then(function (r) { return r.json(); })
        .then(function (s) {
          if (!s.em_andamento) { window.location.reload(); return; }
          var u = s.atual;
          if (u) {
            var etapa = document.getElementById('slide-etapa'); if (etapa) etapa.textContent = u.etapa;
            var p = document.getElementById('slide-pct'); if (p) p.textContent = u.progresso + '%';
            var barra = document.getElementById('slide-barra'); if (barra) barra.value = u.progresso;
          }
          setTimeout(consultarSlide, 1500);
        })
        .catch(function () { setTimeout(consultarSlide, 4000); });
    };
    setTimeout(consultarSlide, 1500);
  }

  // Acompanhamento da coleta: atualiza a barra e recarrega a página quando terminar
  if (document.querySelector('[data-sync-poll]')) {
    var texto = function (id, valor) { var el = document.getElementById(id); if (el) el.textContent = valor; };
    var consultar = function () {
      fetch('/api/sync/status', { headers: { Accept: 'application/json' } })
        .then(function (r) { return r.json(); })
        .then(function (s) {
          if (!s.em_andamento) { window.location.reload(); return; }
          var u = s.ultima;
          if (u) {
            texto('sync-etapa', u.etapa);
            texto('sync-pct', u.progresso + '%');
            texto('sync-linhas', u.linhas.toLocaleString('pt-BR'));
            var barra = document.getElementById('sync-barra');
            if (barra) barra.value = u.progresso;
          }
          setTimeout(consultar, 1500);
        })
        .catch(function () { setTimeout(consultar, 4000); });
    };
    setTimeout(consultar, 1500);
  }
})();
