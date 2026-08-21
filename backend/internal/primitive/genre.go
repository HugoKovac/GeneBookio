package primitive

type Genre string

const (
	Fiction     Genre = "fiction"
	NoneFiction Genre = "none-fiction"
)

func (g Genre) String() string {
	return string(g)
}

func (g Genre) Values() (genres []string) {
	for _, g := range []Genre{
		Fiction,
		NoneFiction,
	} {
		genres = append(genres, string(g))
	}

	return
}
