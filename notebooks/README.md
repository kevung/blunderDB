# Notebooks

`blunderdb-analyse.ipynb` reads the CSV exports of a blunderDB database and
draws the three figures a player actually asks for: the Performance Rating over
time, the distribution of error magnitudes, and the worst decisions.

It exists so that "your data is yours" is a fact rather than a claim: the
notebook uses nothing but `blunderdb list --format csv` and pandas, so anything
it does can be done to your own database, and anything it cannot do you can
write yourself in the next cell.

## Running it

```bash
blunderdb list --db your.db --type positions --format csv > positions.csv
blunderdb list --db your.db --type moves     --format csv > moves.csv
blunderdb list --db your.db --type analyses  --format csv > analyses.csv
jupyter lab blunderdb-analyse.ipynb
```

The notebook expects those three files beside it, and says so rather than
failing on a stack trace when they are missing.

## Why it is executed in CI

A notebook that is not run is a notebook that has stopped working without
anybody noticing — the column it reads was renamed, the CLI flag moved. The
`notebook` job of `.github/workflows/nightly.yml` builds a small database from
`testdata/`, exports the three CSVs and executes every cell. It asserts
nothing about the numbers (they are a fixture's, not a claim); it asserts that
the notebook still runs against today's export.

The CSV columns are a **contract** for exactly that reason: they are added at
the end, never renamed and never reordered. See `internal/cli/cli_list_export.go`.
