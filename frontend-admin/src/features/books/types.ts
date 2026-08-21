export type Language = 'fr' | 'en';

// ModelUsage mirrors primitive.ModelUsage — accumulated usage for one AI
// model. For character-priced models (e.g. tts-1) input_tokens holds a
// character count instead of a token count.
export type ModelUsage = {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
};

// TokenUsage mirrors primitive.TokenUsage — usage per model, keyed by model
// name (e.g. "gpt-5-mini", "tts-1").
export type TokenUsage = Record<string, ModelUsage>;

// SearchResult mirrors book.Book as returned by GET /search — an ad hoc
// Google Books lookup, not yet a saved catalog entry.
export type SearchResult = {
  Key: string;
  Title: string;
  AuthorNames: string[] | null;
  CoverURL: string;
  Description: string;
};

// CatalogBook mirrors book.Book as returned by GET /books/ — a saved entry
// with its pipeline progress flags.
export type CatalogBook = {
  ID: string;
  Title: string;
  AuthorNames: string[] | null;
  CoverURL: string;
  Key: string;
  Description: string;
  Language: Language;
  Uploaded: boolean;
  Parsed: boolean;
  Prepared: boolean;
  ScriptGenerated: boolean;
  TTSGenerated: boolean;
  Failed: boolean;
  // FailedStage is the backend queue channel that failed (e.g. "prepare"),
  // not a display label — see FAILED_STAGE_TO_PROGRESS_KEY in CataloguePage.
  FailedStage: string;
  ErrorMessage: string;
  TokenUsage: TokenUsage | null;
  CostUSD: number;
  // Omitted by the backend when no exchange rate is available right now.
  CostEUR?: number;
};
