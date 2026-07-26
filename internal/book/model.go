package book

type Book struct {
	Title       string
	AuthorNames []string
	CoverURL    string
	DetailsURL  string
}

type DocsSearchAPIResponse struct {
	AuthorNames []string `json:"author_name"`
	Languages   []string `json:"language"`
	Title       string   `json:"title"`
	CoverID     int      `json:"cover_i"`
	Key         string   `json:"key"`
}

type SearchAPIResponse struct {
	Docs []DocsSearchAPIResponse `json:"docs"`
}
