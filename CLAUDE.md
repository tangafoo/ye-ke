# byakugan

> Working codename (sibling to shikamaru/pigweed/krakoa); ships publicly as
> **Ye Ke**. *Byakugan = the eye that sees through deception.* Fitting: it sees
> the law clearly, sees through a bad-faith stop — and now it watches back.

A **police transparency** app for Malaysians (scope widened 2026-08-13; was
know-your-rights only). Three surfaces, one stance:

1. **The law engine** — grounded RAG over Malaysian statute law: one-tap common
   scenarios plus a free-text ask. In the moment, see **exactly which section
   applies** and **what you can lawfully do** — and quote it. The original
   core; every hard rule below still binds it, untouched.
2. **The ledger** — a living map encyclopedia of police occurrences in
   Malaysia: incidents, history, patterns, pinned to the place they happened.
3. **Live** — urgent broadcast: in a bad situation you go live, nearby users
   with notifications on get pinged, and the stream carries a location pin.

The three feed each other: the law engine tells you where you stand, live is
what you do when it's happening, the ledger is what remains — a public record
that makes the next stop different.

> This file is the intended design. Read this first, then build one phase at a
> time. Do NOT boil the ocean.

## Why this exists

- Malaysia has a huge body of open, public law (road-transport, criminal
  procedure, police-powers statutes) that almost nobody — including the author —
  has read. The asymmetry at a roadside stop is *information*: the officer knows
  (or claims to know) the law; the citizen doesn't.
- **Cold-start-proof at the core.** The law engine grounds on a **static
  external corpus** — the statutes themselves — *not* user-generated content,
  so the app is useful on day one with zero users. (shikamaru's
  grounded-RAG-over-external-data pattern.) The ledger and live are the
  network layer *on top*: they get better with users, but the app never
  depends on them to be worth installing.
- **Sunlight is the second defense.** Knowing the law helps at the stop;
  a public, geographic record of how power actually behaves helps every stop
  after. The ledger turns isolated incidents into visible patterns; live turns
  a lone bad moment into a witnessed one.
- The flagship scenario: *"I did not drink. The officer stopped my car and
  insists the breathalyzer is positive."* The app surfaces the relevant
  provisions, what a lawful test/procedure actually requires, and the citizen's
  options — verbatim section + plain-language framing.
- **Which authority matters as much as which law.** Malaysia has several
  enforcement bodies (PDRM, JPJ, DBKL, state religious authorities, and more),
  each with *different* statutory powers. The same situation can have a different
  answer depending on who stopped you — so the app is structured around that.

## Voice & identity

**Power to the people.** The brand wears the energy of protest — antifa, the
Hong Kong / Taiwan street movements, the Nepal and Bangladesh student uprisings —
and a raw riso / graffiti / punk aesthetic (same hand as the trgf work). Loud
skin.

**byakugan is on the citizen's side — openly.** Not a neutral referee: a
defense lawyer with riso ink on its hands. Built for rabble-rousers. The stance
is **we help people get away** — *lawfully*: every exit the law actually
provides. The condition the officer skipped, the rank he doesn't hold, the six
hours' notice they never gave, the "on-the-spot settlement" that appears in no
compounding list, the presumption that can be rebutted and the exact fact that
rebuts it, the complaint that costs the officer something. The framing energy
splits roughly **60/40**: six parts fighting for the citizen's ground, four
parts cold strict law — because the four parts are what make the six land.

What survives untouched: **the facts are never partisan.** The quote is
verbatim, the grounding is absolute, and when the law is dead against you the
app says so straight and fast — false hope is the one thing that gets a citizen
hurt at a stop. That honesty isn't neutrality; it's *tactics*. A defense lawyer
who lies to their client about the odds is a bad defense lawyer. And the
get-away stance is what rasengan (see "Surfaces & layering") eventually turns
into action: don't just tell them the complaint channel exists — have the form
filled and ready.

## Hard product rules (non-negotiable)

- **Verbatim law, framed — never generated.** The statute text shown is
  retrieved **word-for-word** with its section number and source. The LLM
  explains what it means and what you can do; it **never** writes the quote.
  A hallucinated section quoted to an officer makes things worse.
- **Honest like a defense lawyer, not neutral like a referee.** The app is
  openly in the citizen's corner — it hunts every lawful exit the sections
  provide. But it **never inflates a right or invents ground to stand on**:
  when the authority genuinely has the power to enter, search, or act, it says
  so straight, because a citizen acting on false hope picks a fight they lose.
  Honesty is tactical — knowing when they're within their power is as
  protective as knowing when they're not. Sometimes you have ground; sometimes
  you don't. The app's job is the truth, delivered by someone unmistakably on
  your side.
- **Never dead-end on a "no" — always hand over the next move.** When the answer
  is "yes, they can," that's where the card *continues*, not where it stops. The
  app always leaves you with agency: the limits that still bind them even while
  they act, and the concrete actions you can still take (now and after). A verdict
  with no next step is a betrayal in the moment — see "Answer shape".
- **Fail-soft, always.** No grounded provision found → say so plainly
  ("I can't find a specific section for this — here's the general right and a
  source to check"). **Never improvise law.**
- **Offline-first for the common path.** The most-used scenarios must work with
  no signal. See "Offline-first".
- **Not legal advice.** Persistent, honest disclaimer. The app surfaces *the law
  and its source*; it is an information tool, not a lawyer.
- **Fire in the voice, ice in the advice.** The voice can burn; the *moves* it
  recommends keep the encounter cool — "here is your right / here is the
  procedure / here is the complaint that lands afterward," never "tell the
  officer he's corrupt to his face." Escalate on paper, de-escalate at the
  stop. The counterpunch is the complaint form, not the confrontation.
- **Scope before you answer.** Powers vary by **authority, state, and (for
  religious enforcement) religion**. The app resolves these first and answers
  *for that exact combination* — never a one-size answer across branches or
  states. Unknown combo → fail-soft, don't guess.
- **The ledger never touches the law.** Incident pins, streams, and user
  reports are *testimony*, not authority — same rule as news: they supply the
  situation, **never** the legal answer. The statute corpus stays the only
  grounding for what the law says. An unverified pin also never presents as
  established fact — the ledger labels what's confirmed vs. reported.

## Authority / jurisdiction model

The law alone isn't the answer — **which authority is in front of you** is half
of it. Different bodies have different statutory powers, and the *same* situation
has *different* answers depending on who's asking. "Which branch?" is a
first-class routing dimension, not an afterthought.

Bodies the app must distinguish (starting set; expand):

- **PDRM** (Royal Malaysia Police) — general criminal law / Criminal Procedure
  Code powers (arrest, search, roadblocks).
- **JPJ** (Road Transport Dept) — vehicle, licensing, road-transport matters.
- **DBKL** (KL City Hall) — municipal / local-council enforcement (by-laws,
  licensing, parking, premises).
- **State religious authority** (e.g. JAKIM / state JAIN / JAWI) — syariah
  enforcement. Two qualifiers the corpus MUST encode: it is **state-administered**
  (powers vary by state) and generally applies to **Muslims** — so applicability
  depends on state + religion. The app must never overgeneralize here.
- More as the corpus grows (RELA, Immigration/JIM, Customs, AADK, etc.).

Build implications:

- Every corpus chunk is **tagged by authority + statute + state** so retrieval is
  scoped to the right body.
- Scenario cards **route first**: "who is in front of you?" → resolve rights for
  that authority (+ state, + religion where it matters).
- Unknown or uncovered combo → **fail-soft loudly**: say what it depends on, give
  what's grounded, cite the source. Never assert a power *or a limit* the corpus
  doesn't ground.

## Scenarios (the one-tap buttons)

Curated, precomputed. **Organized into categories** (situation × authority), not
a flat list — because the right answer depends on who's in front of you. Scope is
**not only adversarial cop-stops**: it spans everyday, low-drama "where do I
stand?" moments too — including ones where the honest answer is your *own*
exposure (a fine, a penalty), not a right.

**On the road / in a vehicle**
- Roadblock / vehicle stop — what the relevant body may and may not require.
- Breathalyzer dispute — lawful test procedure + your options if you dispute it.
- Intimacy in a parked car — **routes by authority first** (PDRM vs JPJ vs DBKL
  vs religious), then: what that body may require, and your entry/search/ID
  rights for that body.
- Phone / vehicle search — when consent is required.
- Riding / driving without a licence — what you're exposed to (offence, penalty,
  which authority), and your footing if stopped.

**Being stopped / detained**
- "Am I detained, or free to go?"
- Asking for the officer's ID / station / basis for the stop.
- "They're asking for money."

**At a private space / someone at the door**
- Someone wants to enter a private space (incl. religious enforcement) — entry
  without consent, warrant requirements, who that authority's jurisdiction
  actually covers, and your rights about whether to let them in. Calm,
  know-your-ground framing.

**Everyday / where do I stand** *(not adversarial — just "what's my footing?")*
- Caught doing graffiti — what you're actually exposed to: the offence, the
  likely penalty / fine, and which authority handles it. (The mirror pointed at
  *you* — honest about your own exposure, per "honest both ways".)
- Nowhere to stay tonight / sleeping in your car on the roadside — what's
  lawfully allowed, where, and what an officer may do if they approach.
- (More small, everyday situations to come — busking, loitering, street art,
  public-space use. This category is the quiet-utility heart of the app.)

Each card maps to a **precomputed, statute-grounded answer** shipped in the app —
instant, identical for everyone, generated once.

## Answer shape (every card, every answer)

No answer ever dead-ends on a verdict — *especially* not a disempowering one.
Every card and every `Ask` response carries three beats, then an optional
footnote:

1. **The verdict** — honest, both ways: can they, or can't they.
2. **The limits** — what they can and can't do *even when they're within their
   power* (the boundaries that still bind them while they act).
3. **Your move** — the concrete next actions you can still take: what to say, what
   to ask for (ID, basis, warrant), what you can document, what you don't have to
   consent to even if they proceed, and what recourse you have afterward
   (complaint channel, lawyer, follow-up). Agency, not just a ruling.
4. **Footnote (optional, minimal): "this has happened here."** After beat 3 — and
   *only* after — a small footnote may surface a couple of real news articles of
   similar situations. Context, never grounding: it never touches the legal
   answer, it's the last thing on the card, and it stays tiny. If there's no good
   article, it's simply absent.

A "yes, they can" is where the answer *keeps going*, not where it stops. Evals
gate on this: a verdict-only answer (missing beat 3) is a **failed** answer — even
when the law is dead against you, the card must still hand back a next step. The
footnote is the only beat allowed to be empty.

## Capabilities

- **Scenario cards** — precomputed grounded answers for the common set above.
  Static, offline, free.
- **Ask** — free-text grounded RAG for the long-tail question. Online only;
  streams answer + verbatim section citations.
- **Quote view** — the exact statute text, section number, and source link, to
  show/read out.
- **Source-out** — every answer links to the official source of the provision.
- **Ledger map** — browse incidents by place; later, add pins.
- **Live** — one-tap urgent stream with location pin + nearby-user push.

## The transparency layer (ledger + live)

The map screen is the home of both. Build in small steps — each step useful on
its own:

1. **Map v1** — `flutter_map` (chosen: open-source, tile-provider-agnostic, no
   billing, markers are plain widgets, offline-cacheable later) + OSM tiles for
   dev. Center on user via the location permission we already ask for.
   Production tiles: MapTiler/Stadia free tier — one-line `urlTemplate` swap;
   OSM's public server is dev-only by usage policy.
2. **Pins from DB** — `incidents` table (lat/lng, kind, time, description,
   status), `GET /incidents?bbox=` on map pan, marker layer + clustering
   (`flutter_map_supercluster`). Read-only ledger first.
3. **Pin adding** — user-submitted incidents. This is the first UGC surface
   and the guardrails are the feature (see below).
4. **Live** — stream from the camera FAB's "urgent" path; store location pin;
   push to nearby users (FCM; devices register a coarse home area or last
   location — decide granularity with privacy in mind); stream lands in the
   ledger afterward as an incident with footage.

**UGC guardrails (non-negotiable for the ledger).** A false pin does to the
ledger what a hallucinated section does to the law engine — poisons trust in
the one thing the app has. So: incidents carry a status (`reported` →
`corroborated` → `verified`), unverified content is visibly labeled and never
framed as established fact, there's a report/dispute path, and accusations
naming individuals need review before public display (defamation exposure is
real in Malaysia — get actual legal input before pin-adding ships publicly).
Moderation is editorial work; budget for it or gate submissions.

**Safety design for live.** Streamer safety first: no public "person X is
streaming here" to the whole world — nearby pings only, radius tuned;
consider delayed pin precision; retention and takedown policy decided before
launch, not after the first subpoena. The camera FAB's existing
urgent-mode/livestream stubs in the app shell are the entry point.

## Architecture

- **Frontend: Flutter** — one codebase, ships to **Android, iOS, and web**.
  Houses the on-device corpus + on-device retrieval for offline scenarios.
- **Backend: Go** — API for the online `Ask` path: retrieve relevant sections
  (pgvector) → call Anthropic to frame → stream back answer + citations.
  Flat, simple, fast.
- **On-device / offline RAG (the niche core tech).** The common scenarios and
  their statute sections ship *in the app* with an embedded vector index, so
  one-tap answers work with **no network**. Optional: a small local model for
  offline framing; otherwise offline cards are precomputed text. This is the
  standout engineering piece — and it's *required*, not decorative.
- **Corpus pipeline (worker/script).** Ingest statutes → chunk → embed → build
  (a) the server pgvector store and (b) the shippable on-device index. Run
  offline, versioned; the law changes rarely.
- **Inference:** Anthropic for framing (cheap model that passes evals). Verbatim
  quote is retrieval, not generation — no model cost on the quote itself.

## Surfaces & layering (the eye and the hands)

byakugan is split into **two layers along one hard line: who is allowed to touch
the law.** This keeps the safety-critical core dead simple and lets the
*useful, autonomous, world-changing* stuff live somewhere it can't corrupt the
quote. **byakugan sees; rasengan acts.**

- **Layer 0 — `byakugan` (the engine).** The brain we build brains-first
  (corpus → retrieval → verbatim citation → framing → eval gate). It is a
  **grounded RAG engine for Malaysian law and nothing more**: it never plans, it
  never acts on the world, it only *answers with cited, word-for-word law*. It
  carries the bounded-agentic internals that make grounding **better** —
  notably a **self-critique re-retrieval** step (the model grades its own
  citation and re-queries if grounding is weak) — but **zero user-facing
  autonomy over control flow.** A workflow, not an agent. On purpose.
  - **Exposed as tools, for humans *and other agents*:** `search_law()`,
    `get_section()`, and a higher-level `ask()`. Shipped as an **MCP server /
    HTTP API**, so any external agent (or app) that needs Malaysian statute law
    can plug in. This is the open-source + monetizable surface: open the engine
    for trust and adoption; charge for the hosted endpoint. The
    verbatim-citation guarantee is the moat.

- **Layer 1 — `rasengan` (the hands; separate repo/folder).** The
  **action agent** — *the explosion.* This is where real, bounded-agentic
  autonomy lives: it plans and **acts** — fills government forms for a Malaysian
  who has never read the law, drafts a complaint or a real legal-action letter
  to a practitioner, prepares an IPCMC/SUHAKAM submission, sets follow-ups.
  (As of 2026-08-13 rasengan is in scope for this effort, not a someday-repo —
  the ledger feeds it naturally: an incident + the engine's grounded sections
  = a complaint draft with evidence attached.)
  - **Hard sandbox: rasengan never holds the law.** It can only obtain facts by
    calling byakugan's grounded tools. So it **cannot hallucinate a section** —
    it can only quote what the engine hands it. The hard product rules survive
    contact with autonomy because the autonomous layer is structurally blind to
    raw statute text.

**Three lanes, no latency tax (resolves the routing-by-questioning trap).** You
only pay for what you invoke:
- **Button** → precomputed card. Instant. Touches no agent.
- **Free-text** → `ask()`. One online round-trip. No multi-turn interrogation.
- **"Fill my form / draft the letter"** → *only this* wakes rasengan's slower
  agentic loop.

Buttons stay buttons. The agent runs only when there's a genuine *action* a
button can't precompute.

## Offline-first

- **Common path = offline.** Scenario cards + their sections are bundled and
  served from the on-device index. Zero network, instant, works in a dead-signal
  spot.
- **Online path = long-tail only.** Free-text `Ask` hits the Go backend. If
  offline, the app says so and still offers the closest bundled scenario.
- Precompute everything that's identical for all users (every scenario card);
  only the genuinely novel question touches the API. (shikamaru's cost rule.)

## Language & localization (English-first to build; Malay is the gov-sale unlock)

**Build-phase stance (NOW): English-first.** During the build we optimize for
*legal accuracy*, not language coverage. The framing, the evals, the default
answer language, and the dev focus are all English. Reason: where an act has an
**official AGC English version** (e.g. RTA 1987), that text is accurate enough to
ground on, so English buys us correctness without the cost of getting legal Malay
*framing* right. Malay-first UX and nuanced Malay framing are deferred to the
**gov-sale milestone** (selling to a Malaysian government buyer is when Malay
stops being optional).

**What we still keep bilingual now, because it's free:** the **verbatim quote**.
The quote is *retrieval, not generation* — storing the official BM text next to
the EN costs one `pdftotext`, and the BM is the legally-authoritative wording
(the text you'd actually read to an officer). So the corpus stays bilingual where
both official versions exist; we just don't spend effort on Malay *framing* yet.

**Caveat:** not every act has an official English version — some are BM-only. For
those, English-first means we either defer the act or accept a BM-only quote with
English framing. Don't machine-translate a statute into a "verbatim" quote — that
breaks the hard rule.

*(Original Malay-first vision, deferred — restore at the gov-sale milestone:*
*Malay for the people who need this most; "dense legal BM → plain English" as a*
*core feature; every answer in both, user picks, app remembers.)*
- **Retrieval must be multilingual.** Use an embedding model that genuinely
  handles BM (and mixed BM/English queries), or retrieval silently fails on the
  language the corpus is actually written in.
- **Framing must be localized, not literal.** Legal Malay is formal and dense;
  the plain-language framing has to read like a real person speaking natural
  Malay — right register, right particles, right *emotional* tone — not stiff
  machine translation. Choose the framing model for real Malay fluency and tune
  the prompt per language. Evals include Malay quality, not just English.
- **On-device = English-only, by design.** A multilingual *embedding* model
  on-device is fine, so on-device retrieval works in both languages. But the tiny
  local *generation* model is **English-only** — small models write bad legal
  Malay, so we don't ask them to. The offline/on-device framing speaks English;
  **nuanced Malay framing comes from precomputed cards + the cloud (a strong
  multilingual model)**. Accepted scope: offline path is English, the good Malay
  lives in the precomputed cards and the online `Ask`.

## Stack / tooling

- **Flutter** (Dart) — mobile + web. Embedded vector search for offline RAG.
- **Go** — backend API, pgvector retrieval, Anthropic SDK, SSE streaming.
- **Postgres + pgvector** — server-side corpus store.
- **Anthropic SDK** — answer framing (small model; escalate only if evals demand).
- Corpus pipeline: Go or Python script (one-off-ish; law updates rarely).
- CI: lint + tests + an **eval gate** (see below). Deploy backend to Railway.

## Build order (one phase, then stop and review)

1. **Corpus.** Get one statute area in clean (road-transport / breathalyzer).
   Chunk, store, link each chunk to its real section number + source URL.
2. **Eval set first.** A golden set: for the flagship scenarios, the *correct*
   section(s). Groundedness + "did it cite the right section" — before building
   the UI. Prove retrieval is right or nothing else matters.
3. **Online `Ask` v1.** Go backend: retrieve → frame with Anthropic → stream
   answer + verbatim citation. Behind the eval gate.
4. **Flutter shell + one scenario card**, online, end-to-end on a phone.
5. **Offline: bundle the common scenarios + on-device index.** Common path now
   works with no signal. This is the niche-tech milestone.
6. **Expand scenarios + the quote/source-out view.**
7. **(Phase 2 — minimal) News.** Two small uses, both kept tiny: (a) *discovery* —
   skim real stories (stops, raids, fines, incidents) to find situations people
   *actually* hit and turn the common ones into cards; (b) *footnote* — the
   optional "this has happened here" links at the very end of an answer (beat 4).
   Hard rule for both: the article supplies the *situation*, **never** the legal
   answer. The statute + corpus stays the only source of truth. News is context
   and discovery, never grounding.
8. **Transparency layer, in baby steps** (see "The transparency layer"):
   map v1 (flutter_map + tiles) → pins from DB (read-only ledger) → pin
   adding (only after UGC guardrails) → live + nearby push. Promoted from
   out-of-scope on 2026-08-13 — this is now a core surface, not an extra.
9. **Expose the engine (`byakugan` as a tool).** Wrap `ask` / `search_law` /
   `get_section` as an **MCP server + HTTP API** so external agents and apps can
   consume grounded Malaysian law. Small once phase 3 works; this is the
   open-source + paid-endpoint surface. See "Surfaces & layering".
10. **`rasengan` — the action agent (separate repo).** The bounded-agentic hands
    that *act*: fill government forms, draft complaint / legal-action letters,
    prepare IPCMC/SUHAKAM submissions. Consumes byakugan's grounded tools only —
    never holds raw statute text, so it cannot hallucinate a section. This is the
    "more useful than it has any right to be" payoff; build it only after the
    engine is solid behind the eval gate.

## Conventions

- Verbatim statute text is **retrieved, never generated**. Every quote carries
  section number + source.
- Every LLM output is a typed model — no free-form parsing.
- Prompts are versioned; prompt changes pass evals before merge.
- **Fail-soft** on retrieval misses. Never fabricate a provision.
- Persistent "not legal advice / information tool" disclaimer.

## Cost discipline

- **Precompute the common scenarios once**, ship static/offline. The expensive
  per-request path only fires on novel free-text questions.
- Cheapest model that passes the eval gate; small top-k retrieval.
- Cache `Ask` answers for repeated questions. The corpus rarely changes — embed
  once, hash-skip unchanged.

## Open questions (resolve as phases land)

- Source of truth for the corpus: which official text, and how to keep it fresh
  when law amends. Version the corpus.
- On-device offline framing: bundled precomputed text vs. a tiny local model —
  decide at phase 5 based on size/quality.
- Disclaimer + scope review so the framing stays informational and accurate.
- **Religious-enforcement (syariah) content is high-stakes and varies by state.**
  It's legally nuanced and state-administered — this corpus area needs careful,
  well-sourced grounding and, ideally, review by someone with legal expertise
  before it ships. Treat accuracy here as critical; fail-soft hard until it's solid.
- Authority/state tagging schema for the corpus — settle it early (phase 1), since
  retrieval scoping depends on it.
- Which embedding + framing models are actually strong at Bahasa Malaysia (incl.
  legal register) — test this early; it gates both retrieval quality and whether
  the Malay framing reads human. This decision also shapes the on-device plan.
- **Reach vs. need.** The people most exposed (e.g. young riders with no licence)
  may be the least likely to have the app — or a phone — on them. Worth holding
  honestly; may later justify lightweight reach (posters / QR, shareable cards, an
  SMS or ultra-light fallback). Doesn't change v1, but shapes who we're really
  serving.
