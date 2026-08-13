package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ykanthropic "byakugan/internal/anthropic"
	"byakugan/internal/corpus"
	"byakugan/internal/store"
	"byakugan/internal/voyage"

	"github.com/joho/godotenv"
)

type askRequest struct {
	Question string      `json:"question"`
	Lang     corpus.Lang `json:"lang"`
	Stream   bool        `json:"stream"`
}

type askResponse struct {
	Question  string     `json:"question"`
	Lang      string     `json:"lang"`
	Answer    string     `json:"answer"`
	Citations []citation `json:"citations"`
}

// citation carries the full identity a quote needs to be read to an officer:
// which act, which section, and — when the chunk is a slice — which subsection.
// Related marks sections pulled in by the statute's own cross-references
// (refs expansion) rather than by similarity to the question.
type citation struct {
	Statute     string `json:"statute"`      // "Dangerous Drugs Act 1952"
	StatuteAbbr string `json:"statute_abbr"` // "DDA 1952"
	ActNumber   string `json:"act_number"`   // "234"
	Section     string `json:"section"`      // "37"
	Subsection  string `json:"subsection,omitempty"`
	Heading     string `json:"heading"`
	Text        string `json:"text"`
	SourceURL   string `json:"source_url"`
	Related     bool   `json:"related"`
}

func hitToCitation(h store.Hit, related bool) citation {
	return citation{
		Statute:     h.Statute,
		StatuteAbbr: h.StatuteAbbr,
		ActNumber:   h.ActNumber,
		Section:     h.Section,
		Subsection:  h.Subsection,
		Heading:     h.Heading,
		Text:        h.Text,
		SourceURL:   h.SourceURL,
		Related:     related,
	}
}

type byakuganServer struct {
	voyage    *voyage.Client
	store     *store.Store
	anthropic *ykanthropic.Client
}

// refsK caps how many related ROWS refs expansion may add to the prompt. The
// cap is on rows, not refs: one section-level ref (DDA s37) legitimately fans
// out to several subsection slices, and related sections bypass the reranker
// (they're graph edges, not similarity matches) — the cap is the bloat guard.
const refsK = 3

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx := context.Background()
	mux := http.NewServeMux()
	_ = godotenv.Load()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      withCORS(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 240 * time.Second,
		IdleTimeout:  420 * time.Second,
	}

	voyageKey := os.Getenv("VOYAGE_API_KEY")
	if voyageKey == "" {
		log.Fatal("Voyage API key not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://byakugan:byakugan@localhost:5433/byakugan"
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Fatal("Anthropic API key not set")
	}

	st, err := store.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Could not connect to store: %v", err)
	}
	defer st.Close()

	// Dependency injection
	srv := &byakuganServer{
		voyage:    voyage.New(voyageKey),
		store:     st,
		anthropic: ykanthropic.New(anthropicKey),
	}

	mux.HandleFunc("POST /ask", srv.handleAsk)
	mux.HandleFunc("POST /users/anonymous", srv.handleAnonymousUser)

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}

// Ye Ke API contract types: "citations" "delta" "error" "done"
func (s *byakuganServer) handleAsk(w http.ResponseWriter, r *http.Request) {

	var req askRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("error decoding req: %v", err)
		http.Error(w, "request does not follow expected json schema", http.StatusBadRequest)
		return
	}

	if len(req.Question) == 0 {
		http.Error(w, "Cannot ask an empty question", http.StatusBadRequest)
		return
	}

	// Default to english on invalid Lang param
	if !req.Lang.Valid() {
		log.Println("Defaulting to en...")
		req.Lang = "en"
	}

	// Optional identity: stamps last_seen; later slices (history sync,
	// per-user quotas) will use the returned id. Never logged with the
	// question — who asked and what they asked stay unjoined in our logs.
	_ = s.userID(r)

	vectors, err := s.voyage.Embed(r.Context(), voyage.Query, []string{req.Question})
	if err != nil {
		log.Printf("failed to embed question: %v", err)
		http.Error(w, "byakugan is having trouble", http.StatusServiceUnavailable)
		return
	}

	if len(vectors) == 0 {
		log.Printf("embed returned no vectors")
		http.Error(w, "byakugan is having trouble", http.StatusServiceUnavailable)
		return
	}

	results, err := s.store.Search(r.Context(), vectors[0], store.SearchK, req.Lang, store.MaxDist)
	if err != nil {
		log.Printf("trouble searching DB: %v", err)
		http.Error(w, "byakugan's DB is having a sick day", http.StatusServiceUnavailable)
		return
	}

	if len(results) == 0 {
		json.NewEncoder(w).Encode(askResponse{
			Question:  req.Question,
			Lang:      string(req.Lang),
			Answer:    "I can't find an answer for that - please try rephrasing (✖╭╮✖)",
			Citations: []citation{},
		})
		return
	}

	var resultTexts []string
	for _, result := range results {
		resultTexts = append(resultTexts, result.Text)
	}

	reranked, err := s.voyage.Rerank(r.Context(), req.Question, resultTexts, store.TopK)
	if err != nil {
		log.Printf("Reranker had some trouble: %v", err)
		http.Error(w, "byakugan's brain had a spasm", http.StatusServiceUnavailable)
		return
	}

	var citations []citation

	// reranked hits
	newHits := make([]store.Hit, 0, len(reranked))
	for _, rr := range reranked {
		newHits = append(newHits, results[rr.Index])
		citations = append(citations, hitToCitation(results[rr.Index], false))
	}

	// Refs expansion — follow the statutes' own cross-references one hop.
	// A lawyer who reads DDA s12 (possession) reads s37 (presumptions) next
	// because s12's meaning depends on it; embeddings don't know that, the
	// refs edges do. Fail-soft everywhere: a broken expansion must never
	// break the answer.
	related := s.expandRefs(r.Context(), newHits, req.Lang)
	for _, h := range related {
		citations = append(citations, hitToCitation(h, true))
	}

	flusher, ok := w.(http.Flusher)

	if req.Stream && ok {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		if err := writeSSE(w, flusher, "citations", citations); err != nil {
			log.Printf("[stream] citations flush encountered an error. Claude API call aborted.")
			return
		}

		interrupted, err := s.anthropic.FrameStream(r.Context(), req.Question, newHits, related, func(text string) error {
			return writeSSE(w, flusher, "delta", map[string]string{"text": text})
		})
		if err != nil {
			log.Printf("[stream] framing failed: %v", err)
			writeSSE(w, flusher, "error", map[string]string{"message": "Ye Ke is having difficulty streaming at the moment ლ(́◉◞౪◟◉‵ლ"})
			return
		}

		if interrupted {
			log.Println("[stream] framing was interrupted")
		}

		writeSSE(w, flusher, "done", map[string]bool{"interrupted": interrupted})

	} else {
		w.Header().Set("Content-Type", "application/json")

		answer, interrupted, err := s.anthropic.Frame(r.Context(), req.Question, newHits, related)
		if err != nil {
			log.Printf("Claude API error: %v", err)
			http.Error(w, "Ye Ke is OK. our LLM ai is not. please try again later. (^._.^)ﾉ", http.StatusServiceUnavailable)
			return
		}
		if interrupted {
			log.Printf("[claude] answer may be truncated")
		}
		res := askResponse{
			Question:  req.Question,
			Lang:      string(req.Lang),
			Answer:    answer,
			Citations: citations,
		}

		json.NewEncoder(w).Encode(res)
	}

}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sse marshal: %w", err)
	}

	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonBytes); err != nil {
		return err
	}
	flusher.Flush()

	return nil
}

// collectRefs walks the hits in rank order and gathers their cross-references,
// kept DB-free so it's testable.
func collectRefs(hits []store.Hit, max int) []corpus.RelatedSection {
	present := make(map[corpus.RelatedSection]bool, len(hits))
	for _, h := range hits {
		present[corpus.RelatedSection{Statute: h.StatuteCode, Section: h.Section}] = true
	}

	var refs []corpus.RelatedSection
	collected := make(map[corpus.RelatedSection]bool)

	for _, h := range hits { // hits arrive in rerank order — best hit's refs first
		for _, r := range h.Refs {
			if present[r] || collected[r] {
				continue
			}
			collected[r] = true

			refs = append(refs, r)
			// To account for limit reached in indices not the last
			if len(refs) == max {
				return refs
			}
		}
	}
	return refs
}

// expandRefs collects the cross-references of the retrieved hits and fetches
// their chunks, capped at refsK rows (const above). Dedupe rules: skip refs pointing at a
// section already retrieved, and drop fetched rows whose ID is already among
// the hits (a subsection slice of the same section can be both).
func (s *byakuganServer) expandRefs(ctx context.Context, hits []store.Hit, lang corpus.Lang) []store.Hit {
	presentIDs := make(map[string]bool, len(hits))

	for _, h := range hits {
		presentIDs[h.ID] = true
	}

	refs := collectRefs(hits, refsK)
	if len(refs) == 0 {
		return nil
	}

	fetched, err := s.store.FetchSections(ctx, refs, lang)
	if err != nil {
		log.Printf("[refs] expansion failed (continuing without): %v", err)
		return nil
	}

	var related []store.Hit
	for _, h := range fetched {
		if presentIDs[h.ID] {
			continue
		}
		related = append(related, h)

		if len(related) == refsK {
			break
		}
	}
	return related
}
