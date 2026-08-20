export type Language = 'fr' | 'en';

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
};
