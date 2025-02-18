module bdd.fi/x/runitor

go 1.24

retract v1.1.0 // URI contruction bug affecting self hosted instances. GH #75.

tool github.com/dmarkham/enumer

require (
	github.com/dmarkham/enumer v1.5.11 // indirect
	github.com/pascaldekloe/name v1.0.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
)
