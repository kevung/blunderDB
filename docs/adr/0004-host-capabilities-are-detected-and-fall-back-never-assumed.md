# Host capabilities are detected and fall back, never assumed

## Status

accepted

## Context

blunderDB ships as a single binary to machines nobody on the project controls.
A field report from one user's workstation (Arch Linux, KDE Plasma 6 on Wayland,
azerty keyboard, `fr_FR.UTF-8`) surfaced a cluster of failures that share one root
cause: the app silently assumed a facility the host happened not to provide.

- **Image clipboard.** Copying the board image failed with
  `no clipboard tool found (install xclip or wl-copy)`. Text copy worked on the same
  machine because it goes through the WebView's own clipboard
  (`navigator.clipboard.writeText`), but image copy shells out from Go to `xclip`/
  `wl-copy` (`internal/gui/app.go`), neither of which was installed.
- **CJK font.** Japanese UI text rendered as tofu. The `Noto Sans JP` font *is*
  embedded (`frontend/src/assets/fonts/NotoSansJP-Regular.ttf`, ~5.7 MB, declared via
  `@font-face` in `style.css`), but it is only wired to the `.japanese-text` class and
  a couple of components; the `body` font stack and four hard-coded component stacks
  end in bare `sans-serif`, so any Japanese text outside `.japanese-text` fell back to
  a host CJK font the machine did not have.
- **Last database not reopened.** On startup the app reads the remembered path and
  tries to open it; on *any* failure the `catch` clears the path
  (`SaveLastDatabasePath('')`, `frontend/src/App.svelte`). A transient condition —
  an unmounted disk, a slow network mount, a file locked by a second instance —
  therefore does not just fail once, it *forgets the database permanently*.

These are not bugs in one feature. They are one missing stance: the app treated
clipboard tools, installed fonts, and filesystem availability as guaranteed. They are
not. This ADR fixes the *stance*, not each symptom, so future host-facing code inherits
the right default. It deliberately does **not** cover the field report's logic bugs
(analysis click, comment filter, PR miscalculation, export, …), which fail identically
on any system and belong to a separate track.

## Decision

A **host capability** is a facility blunderDB consumes from the machine/OS/desktop but
does not own — its presence and shape vary per system. The stance is: **never assume a
host capability; detect it, fall back on an embedded or native substitute, and only
block the user's gesture when the capability is essential.** (Glossary terms in
`CONTEXT.md` under "The host environment".)

1. **Two tiers.**
   - *Essential* — exactly two: a **writable config/data directory** and the
     **WebView renderer**. Without either, blunderDB cannot store or search positions,
     so it **fails loud and early** with an actionable message rather than limping in a
     half-broken state.
   - *Optional* — image clipboard, CJK/Latin fonts, single-instance behaviour, expected
     keyboard layout, host locale. Absence must **never block** the core product: the
     app **degrades gracefully** and shows a non-blocking notice.

2. **Fallback ladder, in order.** Prefer a substitute blunderDB **ships** (an embedded
   font) → a **native** mechanism (the WebView's own clipboard) → an **external tool**
   (`xclip`/`wl-copy`) → a **non-blocking notice** explaining what is unavailable and how
   to restore it. Never jump straight to "install a tool."

3. **Each capability = a Capability probe + a Fallback policy.** The probe is a thin
   piece of code that reports raw *facts* about one capability's state as plain data
   (`{ HasXclip, HasWlCopy, SessionType }`) and decides nothing. The policy is a **pure
   function**, facts in / chosen rung out, no I/O. All the *risk* — which rung is right —
   lives in the pure policy, so it is exhaustively unit-testable with hand-written fact
   values, with no host to simulate (functional core / imperative shell).

4. **Validation = unit tests of every policy, plus one "hostile" image as a backstop.**
   The per-policy unit tests are the backbone. A single container image stripped of
   every optional capability (no clipboard tool, no system fonts, an exotic locale) runs
   as a pre-release smoke test to catch what the unit tests forgot to simulate. We do
   **not** try to reproduce each user's workstation faithfully.

5. **Non-goal: blunderDB never reads the host locale.** Formatting is already done with
   internal, hard-coded formats (`fr-FR`, `sv-SE`) rather than the host's `LANG`, which
   is what makes the app robust to the user's locale. This is affirmed as a deliberate
   non-goal, not a capability to probe.

Applied to the reported symptoms:

- **Image clipboard** gains a native-first rung (`navigator.clipboard.write` with a
  PNG `ClipboardItem`) before the external-tool rung, and a final **file-fallback**
  rung: when no clipboard path works, the PNG is written to a file and the notice says
  where, with the install hint grafted on — the user's goal (get the image) succeeds by
  another channel.
- **Fonts:** the embedded CJK font is moved to the **end of the global `body` stack**
  and the hard-coded per-component stacks are purged, so blunderDB never depends on a
  host CJK font.
- **Last database:** the remembered path is purged **only** on a *definitive* error
  (file absent / not a database); on any other error (I/O, lock, transient permission)
  the path is kept and a notice is shown.

## Considered options

- **Reproduce the environments (VMs/containers imitating each reported workstation).**
  Rejected as the primary strategy: heavy, brittle, and it can never enumerate every
  user's machine. Kept only in reduced form as the single "hostile" backstop image.
- **Inject a `HostEnvironment` interface mocked in tests.** Rejected: it still mixes the
  *decision* into code doing (mocked) I/O, and every test has to build a behaving mock.
  The thin-probe / pure-policy split puts all the tested risk in a pure function fed
  literal fact structs — no mock, no `PATH` manipulation.
- **Fail loud for optional capabilities too** (e.g. refuse to run without a clipboard
  tool). Rejected: it lets a missing convenience block the core product, which is the
  whole failure being fixed.
- **Require the user to install external tools / system fonts.** Rejected as the default:
  it pushes host variability back onto the user. Embedding and native mechanisms come
  first; the install hint is a last-rung notice, not the primary path.

## Consequences

- The ~5.7 MB CJK font is affirmed as an accepted binary-size cost, in exchange for
  never depending on a host CJK font.
- Image-clipboard copy no longer hard-fails when no clipboard tool exists; it degrades
  to a saved file plus an install hint.
- The last-database path survives transient host conditions and is forgotten only when
  it is genuinely, permanently invalid.
- New host-facing code has a default to follow: add a thin Capability probe and a pure
  Fallback policy, unit-test the policy, and wire the tiers/ladder above — rather than
  calling `exec.LookPath`/`os.Getenv` inline at the point of use.
- **Coupling noted, out of scope:** running two instances against the same database is
  the probable *cause* of the transient last-database failure. Decision 4 above protects
  against the state loss, but the same-database-twice guard itself is a data-safety /
  concurrency concern that fails identically on any system, so it is routed to the logic
  track — allow multiple instances, but forbid opening one database read-write twice.
