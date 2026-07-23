package primitive

type UserRole string

const Basic UserRole = "basic"

func (r UserRole) String() string {
	return string(r)
}

func (r UserRole) Values() (roles []string) {
	for _, r := range []UserRole{
		Basic,
	} {
		roles = append(roles, string(r))
	}

	return
}
