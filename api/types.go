package api

type Mapper struct {
	Addr   string
	Ja3    bool
	Routes []Route
}

type Route struct {
	Path    string
	Rewrite string
	Action  []string
}
