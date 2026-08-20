package primitive

type Language string

const (
	French  Language = "fr"
	English Language = "en"
)

func (l Language) String() string {
	return string(l)
}

func (l Language) Values() (languages []string) {
	for _, l := range []Language{
		French,
		English,
	} {
		languages = append(languages, string(l))
	}

	return
}
