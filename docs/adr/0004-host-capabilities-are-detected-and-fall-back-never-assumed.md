# Host capabilities are detected and fall back, never assumed

Status: accepted.

## Context

blunderDB ships as one binary to machines nobody on the project controls. Clipboard tools,
installed fonts, keyboard layout, locale and filesystem availability vary per host. Code that
assumes one of them fails on the host that lacks it — image copy shelling out to an absent
`xclip`, Japanese text rendered as tofu for want of a host CJK font, the last database forgotten
forever after one transient open failure. The fix is a stance, not a list of patches.

## Decision

A **host capability** is a facility blunderDB consumes from the machine/OS/desktop but does not
own (glossary: `CONTEXT.md`, "The host environment"). Never assume one: detect it, fall back on
an embedded or native substitute, and block the user's gesture only when the capability is
essential.

1. **Two tiers.** *Essential* — exactly two: a writable config/data directory and the WebView
   renderer; without either the app fails loud and early with an actionable message.
   *Optional* — image clipboard, CJK/Latin fonts, single-instance behaviour, keyboard layout,
   host locale; absence never blocks the core product, the app degrades and shows a
   non-blocking notice.
2. **Fallback ladder, in order:** a substitute blunderDB ships (embedded font) → a native
   mechanism (the WebView's clipboard) → an external tool (`xclip`/`wl-copy`) → a non-blocking
   notice saying what is missing and how to restore it. Never jump straight to "install a tool".
3. **Each capability = a Capability probe + a Fallback policy.** The probe reports raw facts as
   plain data (`{ HasXclip, HasWlCopy, SessionType }`) and decides nothing. The policy is a pure
   function, facts in / chosen rung out, no I/O — so all the risk is unit-testable with literal
   fact values.
4. **Validation:** unit tests of every policy, plus one hostile image (no clipboard tool, no
   system fonts, an exotic locale) as a backstop. Users' workstations are not reproduced.
5. **Non-goal: blunderDB never reads the host locale.** Formatting uses internal fixed formats
   (`fr-FR`, `sv-SE`), never `LANG`.

Applied: image copy tries `navigator.clipboard.write` (PNG `ClipboardItem`), then the external
tool, then writes the PNG to a file and names it in the notice. The embedded CJK font ends the
global `body` font stack; no component declares its own stack. The remembered database path is
purged only on a definitive error (file absent, not a database), never on I/O, lock or transient
permission errors.

## Consequences

- The embedded CJK font (~5.7 MB) is an accepted binary-size cost.
- New host-facing code adds a probe and a pure policy; calling `exec.LookPath`/`os.Getenv`
  inline at the point of use is a fault.
- Two instances opening one database read-write is a concurrency concern, not a host
  capability; it is out of scope here.
- Rejected: reproducing each reported environment in VMs (brittle, never exhaustive; kept only
  as the one hostile image). Rejected: a mocked `HostEnvironment` interface (keeps the decision
  inside I/O code, every test needs a behaving mock). Rejected: failing loud on optional
  capabilities (lets a convenience block the product). Rejected: requiring users to install
  tools or fonts (pushes host variability onto the user).

## Guard

`internal/gui/clipboard_test.go` (policy unit tests); `Dockerfile.hostile`, run by the
`hostile-smoke` job in `.github/workflows/build.yml`.
