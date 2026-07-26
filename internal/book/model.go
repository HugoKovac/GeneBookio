package book

type BookAuthor struct {
	Author struct {
		Key string `json:"key"`
	} `json:"author"`
}

type Book struct {
	Title       string
	AuthorNames []string
	CoverURL    string
	Key         string
	AuthorIDs   []BookAuthor
	Description string
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

type DescriptionWorksApiResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AuthorWorksApiResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type WorksApiResponse struct {
	Description      DescriptionWorksApiResponse `json:"description"`
	Title            string                      `json:"title"`
	Key              string                      `json:"key"`
	FirstPublishDate string                      `json:"first_publish_date"`
	Covers           []int                       `json:"covers"`
	Authors          []BookAuthor
}

type QueryURI struct {
	Query string `uri:"query" validate:"required"`
}

type BookDTO struct {
	Title       string       `json:"title"`
	Authors     []BookAuthor `json:"authors,omitempty"`
	AuthorNames []string     `json:"author_names,omitempty"`
	CoverURL    string       `json:"cover_url"`
	Key         string       `json:"key"`
	Descriptiom string       `json:"description"`
}
