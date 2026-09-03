# Cube efficiency is measured per cube state, and read at the root until gammonNet says otherwise

## Status

accepted — decided 2026-09-03. Closes #192 (plan 2026-09b, fiche C.5). Lands on ADR-0022 (the
curve whose blend coefficient this record is about) and on ADR-0023 (the decision that made
every leaf of the search consume that coefficient). **Changes no behaviour**: it records a
divergence, measures it, and says where the change that would remove it has to be written.

Under the rule ADR-0011 and `CLAUDE.md` state — *"a shared optimisation is decided upstream, in
gammonNet"*, and `cube.go`'s own header, *"a discrepancy from `gn_cube.c` is a bug, never an
improvisation"* — the amendment this record calls for is gammonNet's to make. What is delivered
here is the decision, the measurement, and the proposed upstream patch.

## Context

The fiche was written on the suspicion of a port hole. It is not one; the C was read.

**What the code does.** `DefaultEfficiency` (`engine/gammonnet/cube.go`) returns three values,
one per cube state — owned **0.566**, centred **0.688**, opponent **0.687**. They are not three
guesses: gammonNet fitted each of them by least squares against a *different column* of its
exact two-sided bearoff table (`docs/mesures/2026-08-07-T34-ajustement.md`, 10 000 positions,
seed 20260807), as its own spec §3 requires — *"ajusté … contre les équités cubeful exactes de
la table bilatérale (les trois colonnes : possédé, centré, adverse), jamais repris d'un autre
moteur"*. **They are three branch coefficients, not a property of the position.**

Two places then read one of them where the model asks for another:

1. **`search.go`, every leaf.** `ConfigForPosition` sets `cfg.CubeX = DefaultEfficiency(owner)`
   at the **root**; `nodeValue` prices every leaf with that value, while the *owner* is mirrored
   (Owned ↔ Opponent) at every ply, in the same calls that swap the match state. So on a root
   whose cube is turned, one leaf in two is valued on the opponent branch with the coefficient
   fitted for the owned branch, or the reverse. (A centred root is exempt: `Mirror(Centred) ==
   Centred`.)
2. **`cube.go`, `Decide`.** `eDT` is the branch where the *opponent* holds the doubled cube —
   `janowskiEquity(…, CubeOpponent, efficiency)` in money, `levelBlend(&levels[1], …,
   CubeOpponent, efficiency)` at a score — but `efficiency` is the caller's, which is the
   **current** owner's. The reported take point (`levelSolve(&levels[1], CubeOpponent, …)`) has
   the same shape.

**The C does exactly the same.** `gn_search.c:299` and `:740` pass `config->cube_x` — one
double, fixed by `gn_search_use_cube` from the root — while passing the mirrored owner beside
it; `gn_cube.c:754`, `:790` and `:810` price the `e_dt` branch at the caller's `efficiency`.
The Go is faithful line for line. **This is a model question, not a port hole**, and the fiche's
first branch ("if the C mirrors `cube_x`, it is a port hole → fix it here") does not apply.

**The spec is ambiguous, and that is where the question really lives.** §3 fits `x` per cube
state; §4 then writes `E_nd = c · E(x; état courant)` and `E_dt = 2c · E(x; adversaire possède)`
— *one* `x`, two states. §8 step 2 says a leaf is valued by the model "hors domaine" and, inside
the two-sided table's domain, by `gn_bearoff_equities`, which is indexed by the **local** owner.
So upstream's own exact path already reads the branch matching the leaf's local cube state,
while its model path reads the root's coefficient. **The two paths disagree about which branch
the leaf is on**, and only the model path can be wrong about it.

**What gnubg does, for the record.** `docs/recherche/P6-videau-janowski.md` establishes it from
the gnubg manual and the bug-gnubg list: gnubg's `rCubeX` comes from `EvalEfficiency` and
depends on the **position class** (contact/crashed 0.68, one-sided bearoff 0.6, race
interpolated 0.6 → 0.7 on pip count 40 → 120), never on the cube owner; the owner only selects
the branch inside `Cl2CfMoney`. XG's take points match gnubg's on money, and no public source
shows an owner-dependent efficiency in either engine. blunderDB's scheme is therefore a
divergence from both — a deliberate one, and one made against an exact oracle rather than
against another engine's published constants.

### The measurement

`pkg/blunderdb/engine/gammonnet/cube_efficiency_measure_test.go`, behind
`BLUNDERDB_MEASURE_CUBEX`. Corpus: the **669 real analysed decisions** of the integration
gate's own fixtures (two gnubg SGF + one XG). Measured 2026-09-03, 16 cores.

**How often it can matter.** 369 of 669 decisions (**55.2 %**) are played with a cube that has
already been turned. On the other 300 the root is centred and the gap is exactly zero.

**1 — the error at one leaf** (`Value` at the local branch's coefficient vs the mirrored one):

| scale | mean | median | p95 | max | > 0.001 | > 0.01 |
|---|---:|---:|---:|---:|---:|---:|
| money, per unit of cube | 0.0227 | 0.0256 | 0.0430 | 0.0480 | 91.8 % | 78.6 % |
| match, normalised equity | 0.0050 | 0.0040 | 0.0137 | 0.0404 | 74.4 % | 10.2 % |

**2 — repricing `Decide`'s `eDT` on its own branch** (604 evaluable cube decisions at a score;
centred 235, owned 196, opponent 173): mean |Δ eDT| **0.00126 MWC** (≈ 0.0025 normalised),
max **0.0159 MWC** (≈ 0.032); the reported take point moves 0.0011 on average, 0.0069 at worst;
**0 verdicts flip out of 604.**

**3 — what the search actually returns.** 60 real positions with a turned cube, canonical 2-ply
k=12, at the position's own score:

- The **exact** branch-local variant (`nodeValue` patched to `DefaultEfficiency(owner)`, run and
  reverted): **0 of 60 best moves change**; |Δ equity| mean 0.0040, max 0.0084 normalised.
- The **bracket** (the same search at a uniform x = 0.566 and a uniform x = 0.687, the whole
  amplitude any per-leaf assignment can span): **0 of 60**; mean 0.0040, max 0.0089.

So the mis-read coefficient is frequent and visible in the *third decimal of a reported equity*,
and it has not been observed to change a single thing the tool tells the user.

### The gate's two red decisions, re-measured

The fiche's second half. `TestMeasureGateRedCasesAtDepth`
(`BLUNDERDB_MEASURE_GATE_DEPTH`) replays every score-[1,5] checker decision of the XG fixture at
three settings. The two that block the gate do not move:

| decision | 2-ply k=12 | 2-ply unpruned | 3-ply k=12 |
|---|---:|---:|---:|
| dice [4,3] — `21/17 bar/22` | 0.0552 | 0.0552 | 0.0552 |
| dice [1,1] — `10/9 13/12 6/5(2)` | 0.0738 | (move outside XG's net) | 0.0738 |

Same move chosen at every setting, same cost to the fourth decimal. And there is no "XG MET" to
try: `engine/met.go` **is** Kazaross-XG2, XG's own default table. So the ADR-0023 header's
closing hypothesis — *"what is left to explain them is depth and table"* — is refuted in the
same way its predecessor was: neither `use_cube`, nor cube efficiency, nor depth, nor pruning,
nor the MET moves these two. What is left is the network's own judgement on those two boards,
against XG's deeper analysis of them. (Noted in passing, since it cuts against the intuition
that pruning is the suspect: at dice [5,2], turning pruning **off** made the cost *worse*,
0.0000 → 0.0827.) The gate stays failing, not loosened; its header records this.

## Decision

1. **The cube efficiency stays indexed by CUBE STATE, and stays measured.** blunderDB does not
   adopt gnubg's position-class efficiency. gammonNet's spec §3 forbids a borrowed constant, the
   three values come from a least-squares fit against an exact oracle, and this port's job is to
   carry that model, not to arbitrate between it and another engine's. **The divergence from
   gnubg and XG is assumed, and named here so it is never rediscovered as a bug.**

2. **The coefficient belongs to the BRANCH the curve is on, and reading the root's is a defect
   of the model's statement — a small one, measured above.** There is no reading of the T34 fit
   under which the root's coefficient belongs at a mirrored leaf: each of the three numbers was
   fitted against one column of the exact table, and a leaf is on exactly one of those columns.
   Upstream's own exact path (`gn_bearoff_equities`, indexed by the local owner) already agrees;
   only the model fallback does not.

3. **The correction is written upstream first, or not at all.** It changes the shape of the
   algorithm, not an implementation detail, so `CLAUDE.md`'s upstream rule applies in full:
   `gn_cube.c`, `gn_search.c` and `docs/specs/t34-videau-spec.md` §3/§4/§8 move first, with
   their own measurement, then this port follows and `testdata/cube_gold.bin` and
   `testdata/search_cube_gold.bin` are regenerated from the fixed C. **Nothing in this repository
   changes today.** A local fix would manufacture exactly the divergence `cube.go`'s header
   forbids, and the gold gate would go red on the next run.

4. **What the upstream change is** (delivered as a proposal, not applied):

   - `GnSearchConfig.cube_x` becomes `double cube_x[3]`, indexed by `GnCubeOwner`;
     `gn_search_use_cube(config, owner, const double x[3])`. Passing the same value in all three
     slots reproduces today's behaviour bit for bit, which is how the gold stays a control.
   - `node_value` (`gn_search.c:299`) and the batch path (`:740`) index it by the **local**
     owner they already pass beside it — the same index `gn_bearoff_equities` uses two lines
     above.
   - `gn_cube_decide` prices `e_nd` at `x[owner]` and `e_dt`, plus the `level_solve` that reports
     the take point, at `x[GN_CUBE_OPPONENT]` (`gn_cube.c:752-754`, `:789-790`, `:810`).
   - Spec §4 is rewritten as `E_nd = c · E(x_courant; état courant)` and
     `E_dt = 2c · E(x_adverse; adversaire possède)` — today it writes one `x` for two states,
     which is where the ambiguity §3 does not have was introduced. §8 step 2 says the leaf takes
     the coefficient of its **local** cube state.
   - It is a Configuration change, so `EngineVersion` names the gammonNet tag that carries it and
     every stored gammonNet analysis is stale as a whole (`AnalyzeStaleGammonNet`).

5. **Until then, the divergence is commented where a reader would otherwise "fix" it** —
   at `SearchConfig.CubeX`, at `DefaultEfficiency`, and at `Decide`'s `eDT` line — each pointing
   here. The measurement instrument is committed, not the fix.

## Considered options

**Correct `cube.go`/`search.go` here and regenerate the gold.** Rejected on the invariant, not
on the merits: the gold file exists precisely to catch a port that has drifted from the C, and
"regenerating the gold" to accommodate a local model change is the one use of it that destroys
what it measures. The measurement above is what makes the upstream case; it is not a licence to
skip upstream.

**Adopt gnubg's position-class efficiency (contact 0.68, bearoff 0.6, race interpolated).**
Rejected. It is forbidden by gammonNet's spec §3 (*"jamais repris d'un autre moteur"*); it
would replace three numbers fitted against an exact oracle with three read off a manual; and
this port has no position classifier of gnubg's kind to hang them on. P6 documents the scheme
so that the divergence is a decision, which is what this record makes it.

**Re-fit the three coefficients locally, against blunderDB's own two-sided table**
(`engine/race`, ADR-0009). Deferred, unchanged from issue #122's reasoning: the values were
fitted against *gammonNet's* oracle, and a first port keeps the upstream measurement rather than
blocking on a second one. It is a separate piece of work from *which* coefficient a given leaf
reads, and doing it would not answer this record's question.

**Add a `CubeXMirror` flag so the variant can be measured in production code.** Rejected: a
knob on a model gammonNet has not adopted is a second model with no owner. The variant was
measured by patching `nodeValue`, recording the numbers here, and reverting — and the committed
bracket measurement reproduces the same figures without any production surface.

**Leave the fiche open and measure nothing.** Rejected: this is the third time the same two
gate decisions have been explained by a hypothesis that a measurement then refuted (ADR-0016's
`use_cube`, ADR-0023's depth/MET, and now cube efficiency). The point of measuring was to stop
the sequence, and it does: the remaining explanation is the network's judgement on two boards,
which is not a configuration question at all.

## Consequences

- **No behavioural change, no schema change, no `EngineVersion` change.** Stored analyses stay
  valid; the gold files are untouched.
- The divergence from gnubg/XG on cube efficiency is now a written decision rather than an
  undocumented difference, with P6 as its evidence file.
- `cube_efficiency_measure_test.go` is the instrument: three measurements behind
  `BLUNDERDB_MEASURE_CUBEX`, plus the gate's depth replay behind `BLUNDERDB_MEASURE_GATE_DEPTH`.
  Re-run them after any upstream change to see the same numbers move.
- When gammonNet lands the branch-local reading, the port follows: both gold corpora are
  regenerated, `EngineVersion` names the new tag, and every stored gammonNet analysis is stale.
  Expected size of the change, from the measurement: a few thousandths of equity on 55 % of
  positions, and — on this corpus — no change to a single move or verdict.
- `TestIntegrationGate`'s header records the depth replay, so the next reader does not spend
  another run on the hypothesis this one closed.
