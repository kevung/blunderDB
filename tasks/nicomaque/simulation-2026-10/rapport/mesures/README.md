# Meters des passages à l'échelle 100 %

`T2-meter-100.jsonl.gz` et `T3-meter-100.jsonl.gz` : une ligne JSON par action (champs en tête de
`outils/e2e/meter.js`), passages T2 et T3 du 2026-10-04 rejoués avec `UIScale: 100`. Les
tailles de cibles de T2.md, T3.md et ecarts.md (E17, E18) en sont tirées :
`zcat T3-meter-100.jsonl.gz | grep -o 'cible [0-9]*×[0-9]* px < 24 : [^"]*' | sort | uniq -c`.
